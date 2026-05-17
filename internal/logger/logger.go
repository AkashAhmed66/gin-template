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
	"gopkg.in/natefinch/lumberjack.v2"
)

type ctxKey int

const fieldsKey ctxKey = iota

var (
	mu     sync.RWMutex
	global *zap.Logger

	// Captured during New() so File(name) can build per-file loggers that
	// mirror the global rotation/level/encoder settings. Each unique filename
	// gets one cached *zap.Logger; the underlying *lumberjack.Logger handles
	// rotation independently per file. Don't pass dynamic/unbounded filenames
	// to File() — the cache grows once per distinct name and never shrinks.
	builderMu   sync.RWMutex
	builderCfg  config.LogConfig
	builderLvl  zapcore.Level
	builderEnc  zapcore.EncoderConfig
	stdoutCore  zapcore.Core
	fileLoggers = map[string]*zap.Logger{}
)

// New builds a configured zap.Logger. Stdout always receives logs; if Dir and
// File are both non-empty, an additional file sink (Dir/File) gets the same
// stream, with rotation handled by lumberjack: rotate on MaxSizeMB, keep at
// most MaxBackups rotated files, delete files older than MaxAgeDays, optionally
// gzip-compressed.
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

	stdout := zapcore.NewCore(enc, zapcore.AddSync(os.Stdout), lvl)
	cores := []zapcore.Core{stdout}

	if cfg.Dir != "" && cfg.File != "" {
		if err := os.MkdirAll(cfg.Dir, 0o755); err == nil {
			cores = append(cores, fileCore(cfg, cfg.File, encCfg, lvl))
		}
	}

	builderMu.Lock()
	builderCfg = cfg
	builderLvl = lvl
	builderEnc = encCfg
	stdoutCore = stdout
	fileLoggers = map[string]*zap.Logger{}
	builderMu.Unlock()

	return zap.New(zapcore.NewTee(cores...), zap.AddCaller(), zap.AddCallerSkip(0)), nil
}

// fileCore builds a rotating file core writing to cfg.Dir/name.
func fileCore(cfg config.LogConfig, name string, encCfg zapcore.EncoderConfig, lvl zapcore.Level) zapcore.Core {
	rot := &lumberjack.Logger{
		Filename:   filepath.Join(cfg.Dir, name),
		MaxSize:    cfg.MaxSizeMB,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAgeDays,
		Compress:   cfg.Compress,
		LocalTime:  true,
	}
	return zapcore.NewCore(zapcore.NewJSONEncoder(encCfg), zapcore.AddSync(rot), lvl)
}

// File returns a logger that writes to stdout and the named file inside
// LOG_DIR. Rotation settings (size, backups, age, compress) are inherited
// from the global LogConfig. Lines logged via this logger do NOT also go
// to the default LOG_FILE.
//
// Pass an empty name (or call when file output is disabled) and you get the
// default global logger back.
//
// The name is sanitized with filepath.Base, so pass plain filenames like
// "audit.log" — directory components are stripped to prevent path traversal.
// Loggers are cached per filename; calls are cheap after the first.
func File(name string) *zap.Logger {
	if name == "" {
		return L()
	}
	name = filepath.Base(filepath.Clean(name))
	if name == "" || name == "." || name == ".." || name == string(os.PathSeparator) {
		return L()
	}

	builderMu.RLock()
	l, ok := fileLoggers[name]
	builderMu.RUnlock()
	if ok {
		return l
	}

	builderMu.Lock()
	defer builderMu.Unlock()
	if l, ok := fileLoggers[name]; ok {
		return l
	}
	if builderCfg.Dir == "" {
		l := zap.New(stdoutCore, zap.AddCaller(), zap.AddCallerSkip(0))
		fileLoggers[name] = l
		return l
	}
	if err := os.MkdirAll(builderCfg.Dir, 0o755); err != nil {
		return L()
	}
	tee := zapcore.NewTee(stdoutCore, fileCore(builderCfg, name, builderEnc, builderLvl))
	l = zap.New(tee, zap.AddCaller(), zap.AddCallerSkip(0))
	fileLoggers[name] = l
	return l
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

// FromContextFile is FromContext but routes the line to the named file (under
// LOG_DIR) instead of the default LOG_FILE. The name is sanitized — see File
// for details.
func FromContextFile(ctx context.Context, name string) *zap.Logger {
	l := File(name)
	if ctx == nil {
		return l
	}
	fields, _ := ctx.Value(fieldsKey).([]zap.Field)
	if len(fields) == 0 {
		return l
	}
	return l.With(fields...)
}
