package idempotency

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"time"

	"github.com/AkashAhmed66/gin-template/internal/common/security"
	"github.com/AkashAhmed66/gin-template/internal/config"
	"github.com/AkashAhmed66/gin-template/internal/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const HeaderKey = "Idempotency-Key"

// Middleware enforces idempotency on mutating methods. Apply per-route to the
// endpoints that should be replay-safe (e.g. POST /orders).
//
// Behavior:
//   - GET/HEAD/OPTIONS: passthrough (idempotent already).
//   - Missing header: passthrough.
//   - Hit cache with matching request hash: replay stored response, never
//     reach the handler.
//   - Hit cache with mismatched request hash: 409 (caller reused the key for a
//     different payload — almost always a bug).
//   - Miss: capture the response and write it to the store after the handler.
func Middleware(store Store, cfg config.IdempotencyConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}
		key := c.GetHeader(HeaderKey)
		if key == "" {
			c.Next()
			return
		}

		var userID uint64
		if p, ok := c.Get(security.CtxUserID); ok {
			if u, ok := p.(uint64); ok {
				userID = u
			}
		}

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Next()
			return
		}
		_ = c.Request.Body.Close()
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		hash := sha256.Sum256(body)
		bodyHash := hex.EncodeToString(hash[:])

		existing, err := store.Find(c.Request.Context(), key, c.Request.Method, c.Request.URL.Path, userID)
		if err != nil {
			logger.FromContext(c.Request.Context()).Warn("idempotency lookup failed", zap.Error(err))
		}
		if existing != nil {
			if existing.RequestHash != bodyHash {
				c.AbortWithStatusJSON(http.StatusConflict, gin.H{
					"success": false,
					"message": "Idempotency-Key reused with different payload",
				})
				return
			}
			c.Status(existing.StatusCode)
			c.Writer.Header().Set("Content-Type", "application/json")
			_, _ = c.Writer.Write(existing.ResponseBody)
			c.Abort()
			return
		}

		bw := &bufferedWriter{ResponseWriter: c.Writer, buf: &bytes.Buffer{}}
		c.Writer = bw
		c.Next()

		if c.IsAborted() {
			return
		}
		rec := &Record{
			Key:          key,
			Method:       c.Request.Method,
			Path:         c.Request.URL.Path,
			UserID:       userID,
			RequestHash:  bodyHash,
			StatusCode:   bw.status,
			ResponseBody: bw.buf.Bytes(),
			CreatedAt:    time.Now().UTC(),
			ExpiresAt:    time.Now().UTC().Add(cfg.TTL),
		}
		if err := store.Save(c.Request.Context(), rec); err != nil {
			logger.FromContext(c.Request.Context()).Warn("idempotency save failed", zap.Error(err))
		}
	}
}

type bufferedWriter struct {
	gin.ResponseWriter
	buf    *bytes.Buffer
	status int
}

func (b *bufferedWriter) WriteHeader(code int) {
	b.status = code
	b.ResponseWriter.WriteHeader(code)
}

func (b *bufferedWriter) Write(p []byte) (int, error) {
	if b.status == 0 {
		b.status = http.StatusOK
	}
	b.buf.Write(p)
	return b.ResponseWriter.Write(p)
}
