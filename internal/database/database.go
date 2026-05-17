// Package database wires GORM to either Postgres or MySQL based on config, and
// runs SQL migrations via goose. Connection pooling values come from config.
package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/AkashAhmed66/gin-template/internal/config"
	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open connects to the configured database, tunes the pool, and returns a
// GORM handle. Caller is responsible for calling Close on shutdown.
func Open(cfg config.DBConfig, log *zap.Logger) (*gorm.DB, error) {
	dialector, err := buildDialector(cfg)
	if err != nil {
		return nil, err
	}

	gormCfg := &gorm.Config{
		Logger:                                   gormLogger(log),
		DisableForeignKeyConstraintWhenMigrating: true,
		SkipDefaultTransaction:                   true,
	}

	db, err := gorm.Open(dialector, gormCfg)
	if err != nil {
		return nil, fmt.Errorf("gorm open: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("gorm sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	log.Info("database connected",
		zap.String("driver", cfg.Driver),
		zap.String("host", cfg.Host),
		zap.String("name", cfg.Name),
	)

	return db, nil
}

// Close closes the underlying *sql.DB. Safe to call with a nil handle.
func Close(db *gorm.DB) {
	if db == nil {
		return
	}
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
}

// RunMigrations applies any pending goose migrations from dir.
func RunMigrations(db *gorm.DB, driver, dir string, log *zap.Logger) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if err := goose.SetDialect(gooseDialect(driver)); err != nil {
		return fmt.Errorf("goose dialect: %w", err)
	}
	goose.SetLogger(&gooseZapLogger{l: log})

	log.Info("running migrations", zap.String("dir", dir))
	return goose.Up(sqlDB, dir)
}

func buildDialector(cfg config.DBConfig) (gorm.Dialector, error) {
	switch strings.ToLower(cfg.Driver) {
	case "postgres", "postgresql":
		dsn := fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
			cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode, cfg.TimeZone,
		)
		return postgres.Open(dsn), nil
	case "mysql":
		dsn := fmt.Sprintf(
			"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=%s",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, cfg.TimeZone,
		)
		return mysql.Open(dsn), nil
	}
	return nil, fmt.Errorf("unsupported db driver %q (use postgres or mysql)", cfg.Driver)
}

func gooseDialect(driver string) string {
	switch strings.ToLower(driver) {
	case "postgres", "postgresql":
		return "postgres"
	case "mysql":
		return "mysql"
	}
	return driver
}

// gormLogger bridges gorm's logger interface to zap. Slow queries (>200ms)
// log at WARN; everything else at DEBUG. Silences gorm's noisy default logger.
func gormLogger(log *zap.Logger) logger.Interface {
	return logger.New(
		&zapWriter{l: log},
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)
}

type zapWriter struct{ l *zap.Logger }

func (w *zapWriter) Printf(format string, args ...any) {
	w.l.Sugar().Debugf(format, args...)
}

type gooseZapLogger struct{ l *zap.Logger }

func (g *gooseZapLogger) Fatalf(format string, v ...any) { g.l.Sugar().Fatalf(format, v...) }
func (g *gooseZapLogger) Printf(format string, v ...any) { g.l.Sugar().Infof(format, v...) }

// Ensure unused import elimination doesn't drop sql.
var _ = sql.ErrNoRows
