package idempotency

import (
	"context"
	"time"

	"github.com/AkashAhmed66/gin-template/internal/config"
	"go.uber.org/zap"
)

// StartCleanup runs DeleteExpired on a ticker until ctx is cancelled.
// Returns immediately when Idempotency is disabled.
func StartCleanup(ctx context.Context, store Store, cfg config.IdempotencyConfig, log *zap.Logger) {
	if !cfg.Enabled {
		return
	}
	ticker := time.NewTicker(cfg.CleanupInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case t := <-ticker.C:
				n, err := store.DeleteExpired(context.Background(), t)
				if err != nil {
					log.Warn("idempotency cleanup failed", zap.Error(err))
					continue
				}
				if n > 0 {
					log.Info("idempotency cleanup", zap.Int64("deleted", n))
				}
			}
		}
	}()
}
