package security

import (
	"context"
	"net/http"

	"github.com/AkashAhmed66/gin-template/internal/common/dto"
	"github.com/AkashAhmed66/gin-template/internal/common/errs"
	"github.com/AkashAhmed66/gin-template/internal/common/web"
	"github.com/gin-gonic/gin"
)

// Principal is the authenticated subject for the current request. Built by the
// JWT middleware and stashed on gin.Context under CtxPrincipal.
type Principal struct {
	UserID         uint64
	Username       string
	SessionID      uint64
	ImpersonatorID uint64
	Authorities    []string // union of "ROLE_<name>" + permission names
}

// HasAuthority reports whether the principal owns the named role/permission.
func (p *Principal) HasAuthority(name string) bool {
	for _, a := range p.Authorities {
		if a == name {
			return true
		}
	}
	return false
}

// HasAnyAuthority reports whether the principal owns at least one of names.
func (p *Principal) HasAnyAuthority(names ...string) bool {
	for _, name := range names {
		if p.HasAuthority(name) {
			return true
		}
	}
	return false
}

// HasRole is HasAuthority with the "ROLE_" prefix added.
func (p *Principal) HasRole(role string) bool {
	return p.HasAuthority("ROLE_" + role)
}

// CurrentPrincipal returns the principal stashed on the request context, or
// an *errs.AppError (401) if absent.
func CurrentPrincipal(c *gin.Context) (*Principal, error) {
	if v, ok := c.Get(CtxPrincipal); ok {
		if p, ok := v.(*Principal); ok {
			return p, nil
		}
	}
	return nil, errs.Unauthorized("Authentication required")
}

// MustPrincipal is CurrentPrincipal that aborts with 401 on failure. Use only
// from handlers wrapped in web.Handler — it relies on the wrapper to render
// the error envelope.
func MustPrincipal(c *gin.Context) *Principal {
	p, err := CurrentPrincipal(c)
	if err != nil {
		web.WriteResponse(c, dto.Error(http.StatusUnauthorized, "Authentication required", nil))
		c.Abort()
		return nil
	}
	return p
}

// FromContext is the context.Context equivalent of CurrentPrincipal. Useful for
// service-layer code that receives a context rather than gin.Context.
func FromContext(ctx context.Context) (*Principal, bool) {
	if ctx == nil {
		return nil, false
	}
	if v := ctx.Value(principalKey{}); v != nil {
		if p, ok := v.(*Principal); ok {
			return p, true
		}
	}
	return nil, false
}

// WithPrincipal returns ctx with the principal attached. Used by the JWT
// middleware to mirror the gin.Context principal into the request context.
func WithPrincipal(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

type principalKey struct{}
