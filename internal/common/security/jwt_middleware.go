package security

import (
	"net/http"
	"strings"

	"github.com/AkashAhmed66/gin-template/internal/common/audit"
	"github.com/AkashAhmed66/gin-template/internal/common/dto"
	"github.com/AkashAhmed66/gin-template/internal/common/web"
	"github.com/gin-gonic/gin"
)

// SessionValidator is implemented by the session service. The JWT middleware
// calls Validate on every request to check the JWT's `sid` is still active —
// this is what makes server-side revocation possible.
type SessionValidator interface {
	Validate(sessionID, userID uint64) error
	Touch(sessionID uint64)
}

// JWTAuth is the request-level authentication middleware.
//   - Missing/invalid header → next() with no principal (route may still be public).
//   - Header present but token bad → 401 with envelope.
//   - Valid token but session revoked/expired → 401.
//   - Valid → stash Principal on gin.Context + request.Context().
func JWTAuth(jwtSvc *JwtService, sessions SessionValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := extractBearer(c.GetHeader("Authorization"))
		if raw == "" {
			c.Next()
			return
		}

		claims, err := jwtSvc.Parse(raw)
		if err != nil {
			web.WriteResponse(c, dto.Error(http.StatusUnauthorized, "Invalid or expired token", nil))
			c.Abort()
			return
		}
		if claims.Type != TokenTypeAccess {
			web.WriteResponse(c, dto.Error(http.StatusUnauthorized, "Access token required", nil))
			c.Abort()
			return
		}

		if claims.SessionID != 0 && sessions != nil {
			if err := sessions.Validate(claims.SessionID, claims.UserID); err != nil {
				web.WriteResponse(c, dto.Error(http.StatusUnauthorized, "Session no longer active", nil))
				c.Abort()
				return
			}
			sessions.Touch(claims.SessionID)
		}

		p := &Principal{
			UserID:         claims.UserID,
			Username:       claims.Username,
			SessionID:      claims.SessionID,
			ImpersonatorID: claims.ImpersonatorID,
			Authorities:    claims.Authorities,
		}
		c.Set(CtxPrincipal, p)
		c.Set(CtxUserID, p.UserID)
		c.Set(CtxUsername, p.Username)
		c.Set(CtxSessionID, p.SessionID)
		c.Set(CtxAuthorities, p.Authorities)
		ctx := WithPrincipal(c.Request.Context(), p)
		ctx = audit.SetUsernameOnContext(ctx, p.Username)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// RequireAuth aborts with 401 if no principal is on the context. Use in
// addition to JWTAuth on routes that require authentication.
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := CurrentPrincipal(c); err != nil {
			web.WriteResponse(c, dto.Error(http.StatusUnauthorized, "Authentication required", nil))
			c.Abort()
			return
		}
		c.Next()
	}
}

func extractBearer(h string) string {
	const prefix = "Bearer "
	if len(h) <= len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(h[len(prefix):])
}
