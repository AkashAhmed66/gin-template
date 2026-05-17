package web

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const requestIDHeader = "X-Request-Id"

// RequestID middleware: trust an inbound X-Request-Id if present, otherwise
// generate a UUID. Either way, store on the context and echo on the response
// so clients can correlate.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(requestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}
		c.Set(CtxRequestID, id)
		c.Writer.Header().Set(requestIDHeader, id)
		c.Next()
	}
}

// GetRequestID returns the request id stashed by the RequestID middleware
// (empty string if absent, which would indicate a misconfigured router).
func GetRequestID(c *gin.Context) string {
	if v, ok := c.Get(CtxRequestID); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
