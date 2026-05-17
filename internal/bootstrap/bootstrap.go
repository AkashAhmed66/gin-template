// Package bootstrap is the composition root. It wires every concrete service
// together, registers audit GORM callbacks, and seeds the default admin on
// first boot. Keep this thin — it just glues the pieces; modules own their
// own constructors.
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/AkashAhmed66/gin-template/internal/common/audit"
	"github.com/AkashAhmed66/gin-template/internal/common/idempotency"
	"github.com/AkashAhmed66/gin-template/internal/common/jobs"
	"github.com/AkashAhmed66/gin-template/internal/common/mail"
	"github.com/AkashAhmed66/gin-template/internal/common/queue"
	"github.com/AkashAhmed66/gin-template/internal/common/ratelimit"
	"github.com/AkashAhmed66/gin-template/internal/common/scheduler"
	"github.com/AkashAhmed66/gin-template/internal/common/security"
	"github.com/AkashAhmed66/gin-template/internal/common/worker"
	"github.com/AkashAhmed66/gin-template/internal/config"
	auditmod "github.com/AkashAhmed66/gin-template/internal/modules/audit"
	authmod "github.com/AkashAhmed66/gin-template/internal/modules/auth"
	filemod "github.com/AkashAhmed66/gin-template/internal/modules/file"
	"github.com/AkashAhmed66/gin-template/internal/modules/permission"
	productmod "github.com/AkashAhmed66/gin-template/internal/modules/product"
	"github.com/AkashAhmed66/gin-template/internal/modules/role"
	sessionmod "github.com/AkashAhmed66/gin-template/internal/modules/session"
	usermod "github.com/AkashAhmed66/gin-template/internal/modules/user"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Deps is the wired-up application graph passed into the router.
type Deps struct {
	Cfg       *config.Config
	DB        *gorm.DB
	Log       *zap.Logger
	JWT       *security.JwtService
	RateLimit *ratelimit.Service
	Idem      idempotency.Store
	Mail      mail.Service

	// Repos exposed so other layers (handlers, bootstrap) can compose freely.
	PermissionRepo permission.Repository
	RoleRepo       role.Repository
	UserRepo       usermod.Repository

	Permissions permission.Service
	Roles       role.Service
	Users       usermod.Service
	Sessions    sessionmod.Service
	Auth        authmod.Service
	Audit       auditmod.Service
	Files       filemod.Service
	Products    productmod.Service

	PermHandler    *permission.Handler
	RoleHandler    *role.Handler
	UserHandler    *usermod.Handler
	SessionHandler *sessionmod.Handler
	AuthHandler    *authmod.Handler
	AuditHandler   *auditmod.Handler
	FileHandler    *filemod.Handler
	ProductHandler *productmod.Handler

	Scheduler *scheduler.Scheduler
	Pool      *worker.Pool
	Queue     *queue.Manager
}

// NewDependencies wires the full graph. Anything that needs to be a singleton
// (services, handlers, JWT secret) is created exactly once here.
func NewDependencies(cfg *config.Config, db *gorm.DB, log *zap.Logger) (*Deps, error) {
	if err := audit.RegisterCallbacks(db); err != nil {
		return nil, fmt.Errorf("register audit callbacks: %w", err)
	}

	jwtSvc, err := security.NewJwtService(cfg.JWT)
	if err != nil {
		return nil, err
	}

	rl := ratelimit.NewService(cfg.RateLimit)
	idemStore := idempotency.NewStore(db)
	mailSvc := mail.New(cfg.Mail, "templates/email", log)

	permRepo := permission.NewRepository(db)
	roleRepo := role.NewRepository(db)
	userRepo := usermod.NewRepository(db)

	sessionSvc := sessionmod.NewService(db)
	userSvc := usermod.NewService(userRepo, roleRepo, sessionSvc)
	roleSvc := role.NewService(roleRepo, permRepo)
	permSvc := permission.NewService(permRepo)
	authSvc := authmod.NewService(db, userRepo, roleRepo, sessionSvc, jwtSvc, mailSvc, cfg.Mail, log)
	auditSvc := auditmod.NewService(db, log)
	fileSvc := filemod.New(cfg.Storage, cfg.Mail.AppURL)
	productSvc := productmod.NewService(productmod.NewRepository(db))

	tz, err := time.LoadLocation(cfg.Jobs.Timezone)
	if err != nil {
		log.Warn("invalid JOBS_TIMEZONE, falling back to UTC",
			zap.String("value", cfg.Jobs.Timezone), zap.Error(err))
		tz = time.UTC
	}
	sched := scheduler.New(log, tz)
	pool := worker.NewPool(cfg.Jobs.WorkerPoolSize, cfg.Jobs.QueueSize, log)
	jobs.Set(&jobs.Manager{Scheduler: sched, Pool: pool})

	qmgr, err := queue.New(cfg.Queue, cfg.Redis, log)
	if err != nil {
		return nil, fmt.Errorf("queue init: %w", err)
	}
	queue.Set(qmgr)
	if qmgr != nil {
		log.Info("redis queue enabled",
			zap.String("addr", cfg.Redis.Addr),
			zap.Int("concurrency", cfg.Queue.Concurrency),
		)
	}

	d := &Deps{
		Cfg: cfg, DB: db, Log: log,
		JWT: jwtSvc, RateLimit: rl, Idem: idemStore, Mail: mailSvc,
		PermissionRepo: permRepo, RoleRepo: roleRepo, UserRepo: userRepo,
		Permissions: permSvc, Roles: roleSvc, Users: userSvc,
		Sessions: sessionSvc, Auth: authSvc, Audit: auditSvc,
		Files: fileSvc, Products: productSvc,

		PermHandler:    permission.NewHandler(permSvc),
		RoleHandler:    role.NewHandler(roleSvc),
		UserHandler:    usermod.NewHandler(userSvc),
		SessionHandler: sessionmod.NewHandler(sessionSvc),
		AuthHandler:    authmod.NewHandler(authSvc),
		AuditHandler:   auditmod.NewHandler(auditSvc, cfg.Audit),
		FileHandler:    filemod.NewHandler(fileSvc),
		ProductHandler: productmod.NewHandler(productSvc, fileSvc),

		Scheduler: sched,
		Pool:      pool,
		Queue:     qmgr,
	}

	// Register asynq handlers. Each module that owns a task type registers it
	// here (or from its own constructor). Handlers must be registered BEFORE
	// the queue server starts in StartBackgroundJobs.
	if d.Queue != nil {
		authmod.RegisterQueueHandlers(d.Queue, d.Mail, d.Log)
	}

	return d, nil
}

// StartBackgroundJobs spins up the worker pool and registers the built-in
// recurring cleanup jobs on the scheduler, then starts the scheduler. Any
// service can register additional jobs via jobs.Every / jobs.Cron / jobs.Submit.
func (d *Deps) StartBackgroundJobs() {
	if !d.Cfg.Jobs.Enabled {
		d.Log.Info("background jobs disabled (JOBS_ENABLED=false)")
		return
	}
	d.Pool.Start()
	d.registerBuiltInJobs()
	d.Scheduler.Start()

	if d.Queue != nil {
		go func() {
			if err := d.Queue.Start(); err != nil {
				d.Log.Error("queue server stopped", zap.Error(err))
			}
		}()
	}
}

// StopBackgroundJobs cancels scheduled jobs and drains the worker pool + queue.
func (d *Deps) StopBackgroundJobs(ctx context.Context) {
	if !d.Cfg.Jobs.Enabled {
		return
	}
	if d.Queue != nil {
		d.Queue.Stop(ctx)
	}
	d.Scheduler.Stop(ctx)
	d.Pool.Stop(ctx)
}

// registerBuiltInJobs wires the template's stock cleanup routines onto the
// scheduler. Add new application-level jobs anywhere via the jobs.* helpers.
func (d *Deps) registerBuiltInJobs() {
	if iv := d.Cfg.Sessions.CleanupInterval; iv > 0 {
		_ = d.Scheduler.Every("session-cleanup", iv, func(ctx context.Context) error {
			n, err := d.Sessions.CleanupExpired(ctx, d.Cfg.Sessions.CleanupRetention)
			if err != nil {
				return err
			}
			if n > 0 {
				d.Log.Info("session cleanup", zap.Int64("deleted", n))
			}
			return nil
		})
	}

	if d.Cfg.Idem.Enabled {
		if iv := d.Cfg.Idem.CleanupInterval; iv > 0 {
			_ = d.Scheduler.Every("idempotency-cleanup", iv, func(ctx context.Context) error {
				n, err := d.Idem.DeleteExpired(ctx, time.Now().UTC())
				if err != nil {
					return err
				}
				if n > 0 {
					d.Log.Info("idempotency cleanup", zap.Int64("deleted", n))
				}
				return nil
			})
		}
	}

	if iv := d.Cfg.Mail.PasswordResetCleanupInterval; iv > 0 {
		_ = d.Scheduler.Every("password-reset-cleanup", iv, func(ctx context.Context) error {
			n, err := d.Auth.CleanupExpiredResetTokens(ctx)
			if err != nil {
				return err
			}
			if n > 0 {
				d.Log.Info("password reset cleanup", zap.Int64("deleted", n))
			}
			return nil
		})
	}
}

// Wait flushes the audit writer with a deadline.
func (d *Deps) Wait(ctx context.Context) {
	d.Audit.Wait(ctx)
}

// SeedAdmin ensures the bootstrap admin user and ADMIN/USER roles exist with
// the full permission catalogue. Idempotent — safe to run on every boot.
func (d *Deps) SeedAdmin() error {
	ctx := context.Background()

	// 1. Permissions
	for _, name := range permission.All {
		existing, err := d.PermissionRepo.GetByName(ctx, name)
		if err != nil {
			return err
		}
		if existing == nil {
			p := &permission.Permission{Name: name, Description: name + " permission"}
			if err := d.PermissionRepo.Create(ctx, p); err != nil {
				return fmt.Errorf("seed permission %s: %w", name, err)
			}
		}
	}

	// 2. Roles
	if _, err := d.ensureRole(ctx, role.NameAdmin, "Full administrator", permission.All); err != nil {
		return err
	}
	if _, err := d.ensureRole(ctx, role.NameUser, "Default end user",
		[]string{permission.ProductRead}); err != nil {
		return err
	}

	// 3. Admin user
	bs := d.Cfg.Bootstrap
	existing, err := d.UserRepo.GetByUsernameOrEmail(ctx, bs.AdminUsername)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}
	hash, err := security.HashPassword(bs.AdminPassword)
	if err != nil {
		return err
	}
	adminRow, err := d.RoleRepo.GetByName(ctx, role.NameAdmin)
	if err != nil {
		return err
	}
	if adminRow == nil {
		return errors.New("admin role missing after seed")
	}
	u := &usermod.User{
		Username:     bs.AdminUsername,
		Email:        bs.AdminEmail,
		PasswordHash: hash,
		FullName:     "System Administrator",
		Enabled:      true,
		Roles:        []role.Role{*adminRow},
	}
	if err := d.UserRepo.Create(ctx, u); err != nil {
		return err
	}
	d.Log.Info("bootstrap admin created",
		zap.String("username", bs.AdminUsername),
		zap.String("email", bs.AdminEmail),
	)
	return nil
}

func (d *Deps) ensureRole(ctx context.Context, name, desc string, perms []string) (*role.Role, error) {
	existing, err := d.RoleRepo.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		r := &role.Role{Name: name, Description: desc}
		if err := d.RoleRepo.Create(ctx, r); err != nil {
			return nil, err
		}
		existing = r
	}
	resolved, err := d.PermissionRepo.FindAllByNames(ctx, perms)
	if err != nil {
		return nil, err
	}
	if err := d.RoleRepo.ReplacePermissions(ctx, existing, resolved); err != nil {
		return nil, err
	}
	existing.Permissions = resolved
	return existing, nil
}
