package web

import (
	"net/http"
	"runtime/debug"

	"github.com/AkashAhmed66/gin-template/internal/common/dto"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Recovery catches panics from handlers, logs the stack, and writes a 500
// envelope. Always register before any module middleware so it's the outermost
// catch.
func Recovery(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("panic recovered",
					zap.Any("error", r),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
					zap.String("request_id", GetRequestID(c)),
					zap.ByteString("stack", debug.Stack()),
				)
				if !c.Writer.Written() {
					WriteResponse(c, dto.Error(http.StatusInternalServerError, "Internal server error", nil))
				}
				c.Abort()
			}
		}()
		c.Next()
	}
}
