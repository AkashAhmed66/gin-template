package security

import (
	"net/http"

	"github.com/AkashAhmed66/gin-template/internal/common/dto"
	"github.com/AkashAhmed66/gin-template/internal/common/web"
	"github.com/gin-gonic/gin"
)

// HasPermission is middleware that allows the request only if the caller's
// authorities include any of the named permission names. Matches Spring's
// @HasPermission annotation.
func HasPermission(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, err := CurrentPrincipal(c)
		if err != nil {
			web.WriteResponse(c, dto.Error(http.StatusUnauthorized, "Authentication required", nil))
			c.Abort()
			return
		}
		if !p.HasAnyAuthority(permissions...) {
			web.WriteResponse(c, dto.Error(http.StatusForbidden, "Access denied", nil))
			c.Abort()
			return
		}
		c.Next()
	}
}

// HasRole is middleware that allows the request only if the caller owns any of
// the named roles. Matches Spring's @HasRole annotation.
func HasRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, err := CurrentPrincipal(c)
		if err != nil {
			web.WriteResponse(c, dto.Error(http.StatusUnauthorized, "Authentication required", nil))
			c.Abort()
			return
		}
		for _, r := range roles {
			if p.HasRole(r) {
				c.Next()
				return
			}
		}
		web.WriteResponse(c, dto.Error(http.StatusForbidden, "Access denied", nil))
		c.Abort()
	}
}
