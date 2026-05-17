// Package main is the entry point for the Gin API template.
//
// Boot order: env → logger → DB connect → migrations → bootstrap admin → router → HTTP server.
// Graceful shutdown on SIGINT/SIGTERM cancels in-flight requests within SERVER_SHUTDOWN_TIMEOUT.
//
// Bare `swag init` from the repo root regenerates the Swagger spec — main.go
// is here, the docs/ output directory is here, no flags needed.
//go:generate swag init --parseDependency --parseInternal
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/AkashAhmed66/gin-template/internal/bootstrap"
	"github.com/AkashAhmed66/gin-template/internal/config"
	"github.com/AkashAhmed66/gin-template/internal/database"
	"github.com/AkashAhmed66/gin-template/internal/logger"
	"github.com/AkashAhmed66/gin-template/internal/router"
	"go.uber.org/zap"

	// Blank-import the generated Swagger docs so init() registers the spec
	// with the swag runtime. The placeholder docs.go does nothing; once you
	// run `swag init`, the generated file's init() makes the spec available
	// at /swagger/doc.json.
	_ "github.com/AkashAhmed66/gin-template/docs"
)

// @title           Gin Template API
// @version         1.0
// @description     Reusable Gin starter with JWT auth, RBAC, sessions, audit logging, rate limiting, idempotency, and file uploads.
// @BasePath        /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	cfg, err := config.Load()
	if err != nil {
		// stderr because the logger isn't up yet.
		_, _ = os.Stderr.WriteString("config error: " + err.Error() + "\n")
		os.Exit(1)
	}

	log, err := logger.New(cfg.Log)
	if err != nil {
		_, _ = os.Stderr.WriteString("logger error: " + err.Error() + "\n")
		os.Exit(1)
	}
	defer func() { _ = log.Sync() }()
	logger.SetGlobal(log)

	db, err := database.Open(cfg.DB, log)
	if err != nil {
		log.Fatal("database connect failed", zap.Error(err))
	}
	defer database.Close(db)

	if cfg.Migrations.AutoRun {
		if err := database.RunMigrations(db, cfg.DB.Driver, cfg.Migrations.Dir, log); err != nil {
			log.Fatal("migrations failed", zap.Error(err))
		}
	}

	deps, err := bootstrap.NewDependencies(cfg, db, log)
	if err != nil {
		log.Fatal("bootstrap failed", zap.Error(err))
	}

	if err := deps.SeedAdmin(); err != nil {
		log.Fatal("bootstrap admin failed", zap.Error(err))
	}

	engine := router.New(cfg, log, deps)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      engine,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		log.Info("http server listening",
			zap.String("addr", srv.Addr),
			zap.String("env", cfg.App.Env),
		)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("http server failed", zap.Error(err))
		}
	}()

	deps.StartBackgroundJobs()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Info("shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	deps.StopBackgroundJobs(ctx)

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("graceful shutdown failed", zap.Error(err))
		_ = srv.Close()
	}

	// Drain any tail-end async work (audit, mail, idempotency cleanup).
	deps.Wait(ctx)
	log.Info("shutdown complete")
}
