package auth

import (
	"context"
	"time"

	"github.com/AkashAhmed66/gin-template/internal/config"
	"go.uber.org/zap"
)

// StartResetTokenCleanup runs CleanupExpiredResetTokens on a ticker.
func StartResetTokenCleanup(ctx context.Context, svc Service, cfg config.MailConfig, log *zap.Logger) {
	interval := cfg.PasswordResetCleanupInterval
	if interval <= 0 {
		interval = time.Hour
	}
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				n, err := svc.CleanupExpiredResetTokens(context.Background())
				if err != nil {
					log.Warn("password reset cleanup failed", zap.Error(err))
					continue
				}
				if n > 0 {
					log.Info("password reset cleanup", zap.Int64("deleted", n))
				}
			}
		}
	}()
}
