// Package logger wraps zap with a request-scoped context and an idiomatic
// console/json encoder selectable from config. Mirrors the spring-boot
// MDC pattern of attaching `requestId` and `username` to every log line.
package logger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/AkashAhmed66/gin-template/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ctxKey int

const fieldsKey ctxKey = iota

var (
	mu     sync.RWMutex
	global *zap.Logger
)

// New builds a configured zap.Logger. Stdout always receives logs; if Dir is
// non-empty an additional app.log file gets the same stream.
func New(cfg config.LogConfig) (*zap.Logger, error) {
	lvl, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	encCfg := zap.NewProductionEncoderConfig()
	encCfg.TimeKey = "ts"
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encCfg.EncodeLevel = zapcore.LowercaseLevelEncoder
	encCfg.EncodeDuration = zapcore.MillisDurationEncoder

	var enc zapcore.Encoder
	if strings.EqualFold(cfg.Format, "json") {
		enc = zapcore.NewJSONEncoder(encCfg)
	} else {
		encCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		enc = zapcore.NewConsoleEncoder(encCfg)
	}

	cores := []zapcore.Core{
		zapcore.NewCore(enc, zapcore.AddSync(os.Stdout), lvl),
	}

	if cfg.Dir != "" {
		if err := os.MkdirAll(cfg.Dir, 0o755); err == nil {
			f, ferr := os.OpenFile(filepath.Join(cfg.Dir, "app.log"),
				os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
			if ferr == nil {
				cores = append(cores,
					zapcore.NewCore(zapcore.NewJSONEncoder(encCfg), zapcore.AddSync(f), lvl))
			}
		}
	}

	return zap.New(zapcore.NewTee(cores...), zap.AddCaller(), zap.AddCallerSkip(0)), nil
}

func parseLevel(s string) (zapcore.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info", "":
		return zapcore.InfoLevel, nil
	case "warn", "warning":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	case "fatal":
		return zapcore.FatalLevel, nil
	}
	return zapcore.InfoLevel, fmt.Errorf("unknown log level %q", s)
}

// SetGlobal installs the application-wide logger.
func SetGlobal(l *zap.Logger) {
	mu.Lock()
	defer mu.Unlock()
	global = l
}

// L returns the global logger (must be set via SetGlobal first; falls back to
// no-op if absent so library code never panics).
func L() *zap.Logger {
	mu.RLock()
	defer mu.RUnlock()
	if global == nil {
		return zap.NewNop()
	}
	return global
}

// WithFields stores fields on the context so any later FromContext call picks
// them up. Used by middleware to thread request-scoped metadata
// (requestId, username, sid) through call chains.
func WithFields(ctx context.Context, fields ...zap.Field) context.Context {
	if len(fields) == 0 {
		return ctx
	}
	existing, _ := ctx.Value(fieldsKey).([]zap.Field)
	merged := make([]zap.Field, 0, len(existing)+len(fields))
	merged = append(merged, existing...)
	merged = append(merged, fields...)
	return context.WithValue(ctx, fieldsKey, merged)
}

// FromContext returns the global logger annotated with any fields previously
// attached via WithFields.
func FromContext(ctx context.Context) *zap.Logger {
	l := L()
	if ctx == nil {
		return l
	}
	fields, _ := ctx.Value(fieldsKey).([]zap.Field)
	if len(fields) == 0 {
		return l
	}
	return l.With(fields...)
}
