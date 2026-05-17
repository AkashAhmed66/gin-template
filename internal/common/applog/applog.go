// Package applog routes domain log lines to a named file with request-scoped
// enrichment automatically applied. Use this from services/repos whenever you
// want a dedicated rotating log file per domain (products.log, orders.log,
// audit.log, ...).
//
// Compared to internal/logger directly:
//   - Picks up the request fields installed by middleware (requestId, etc.)
//   - Adds actorId + actor from the security principal when one is present
//   - Routes to LOG_DIR/<file> with the global rotation settings applied
package applog

import (
	"context"

	"github.com/AkashAhmed66/gin-template/internal/common/security"
	"github.com/AkashAhmed66/gin-template/internal/logger"
	"go.uber.org/zap"
)

// Channel binds a filename once and returns a function that produces a
// request-scoped logger for that file. Declare one at package level per
// domain so call sites don't have to repeat the filename.
//
// Usage:
//
//	// package-level (once)
//	var plog = applog.Channel("products.log")
//
//	// call sites (just pass ctx)
//	plog(ctx).Info("created", zap.Uint64("id", p.ID))
//
//	// multiple lines in one method — capture and reuse
//	log := plog(ctx)
//	log.Info("step a")
//	log.Warn("step b")
//
// The ctx is still required because requestId / username / actor fields live
// on the request context — they can't be recovered globally.
func Channel(file string) func(context.Context) *zap.Logger {
	return func(ctx context.Context) *zap.Logger {
		return For(ctx, file)
	}
}

// For returns a zap.Logger that writes to stdout AND LOG_DIR/file, carrying
// any request-scoped fields plus actorId/actor when the context has a
// principal. Pass file="" to fall back to the default global logger.
//
// Usage:
//
//	applog.For(ctx, "products.log").Info("created",
//	    zap.Uint64("id", p.ID),
//	    zap.String("sku", p.SKU),
//	)
//
// For repeated logging in the same method, capture once:
//
//	log := applog.For(ctx, "products.log")
//	log.Info("step a", ...)
//	log.Warn("step b", ...)
//
// The returned logger is cached per filename inside the logger package, so
// repeated calls with the same file are cheap.
func For(ctx context.Context, file string) *zap.Logger {
	l := logger.FromContextFile(ctx, file)
	if p, ok := security.FromContext(ctx); ok {
		l = l.With(zap.Uint64("actorId", p.UserID), zap.String("actor", p.Username))
	}
	return l
}
