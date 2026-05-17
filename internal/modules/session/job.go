package session

import (
	"context"
	"time"

	"github.com/AkashAhmed66/gin-template/internal/config"
	"go.uber.org/zap"
)

// StartCleanup runs CleanupExpired on a ticker until ctx is cancelled.
func StartCleanup(ctx context.Context, svc Service, cfg config.SessionsConfig, log *zap.Logger) {
	ticker := time.NewTicker(cfg.CleanupInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				n, err := svc.CleanupExpired(context.Background(), cfg.CleanupRetention)
				if err != nil {
					log.Warn("session cleanup failed", zap.Error(err))
					continue
				}
				if n > 0 {
					log.Info("session cleanup", zap.Int64("deleted", n))
				}
			}
		}
	}()
}
