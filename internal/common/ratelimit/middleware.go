package ratelimit

import (
	"fmt"
	"net/http"
	"time"

	"github.com/AkashAhmed66/gin-template/internal/common/dto"
	"github.com/AkashAhmed66/gin-template/internal/common/security"
	"github.com/AkashAhmed66/gin-template/internal/common/web"
	"github.com/gin-gonic/gin"
)

// Middleware applies the rate-limit Service. Identity is the authenticated
// username if present, else the client IP. Adds X-RateLimit-Limit and
// X-RateLimit-Remaining on every response; adds Retry-After + 429 on rejection.
func Middleware(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !svc.Enabled() {
			c.Next()
			return
		}
		path := c.Request.URL.Path
		if !svc.IsIncluded(path) {
			c.Next()
			return
		}
		identity := c.ClientIP()
		if p, ok := c.Get(security.CtxUsername); ok {
			if s, ok := p.(string); ok && s != "" {
				identity = s
			}
		}

		var (
			allowed   bool
			remaining int
			limit     int
			retry     time.Duration
		)
		if svc.IsAuthPath(path) {
			allowed, remaining, limit, retry = svc.TakeAuth(identity)
		} else {
			allowed, remaining, limit, retry = svc.TakeDefault(identity)
		}

		c.Writer.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
		c.Writer.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

		if !allowed {
			c.Writer.Header().Set("Retry-After", fmt.Sprintf("%d", int(retry.Seconds())))
			web.WriteResponse(c, dto.Error(http.StatusTooManyRequests,
				fmt.Sprintf("Rate limit exceeded. Retry in %ds.", int(retry.Seconds())), nil))
			c.Abort()
			return
		}
		c.Next()
	}
}
