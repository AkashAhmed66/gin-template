package web

import (
	"time"

	"github.com/AkashAhmed66/gin-template/internal/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RequestLogging logs one structured line per request at INFO. 5xx responses
// are logged at ERROR so they surface in alerts without raising the overall
// log level. The line carries the request id so it's joinable with the
// matching audit row.
func RequestLogging(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Set(CtxStart, start)

		// Forward request_id into the request context as a zap field so any
		// downstream service that calls logger.FromContext(ctx) sees it.
		rid := GetRequestID(c)
		ctx := logger.WithFields(c.Request.Context(), zap.String("request_id", rid))
		c.Request = c.Request.WithContext(ctx)

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		fields := []zap.Field{
			zap.String("request_id", rid),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("query", c.Request.URL.RawQuery),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
			zap.Int("bytes", c.Writer.Size()),
		}
		if u, ok := c.Get("username"); ok {
			if s, ok := u.(string); ok && s != "" {
				fields = append(fields, zap.String("username", s))
			}
		}
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.String()))
		}

		switch {
		case status >= 500:
			log.Error("http", fields...)
		case status >= 400:
			log.Warn("http", fields...)
		default:
			log.Info("http", fields...)
		}
	}
}
