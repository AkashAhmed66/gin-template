// Package security carries the JWT service, auth middleware, current-user
// helpers, and the HasRole / HasPermission middleware factories. The two halves
// of the package are paired by intent — JwtService issues + parses tokens;
// the rest is consumed by every protected route.
package security

// Context keys used by middleware to stash request-scoped identity on
// gin.Context. The Gin helpers MustGet/Get accept strings, so these are
// untyped on purpose.
const (
	CtxUserID      = "user_id"
	CtxUsername    = "username"
	CtxSessionID   = "session_id"
	CtxImpersonate = "impersonator_id"
	CtxAuthorities = "authorities"
	CtxPrincipal   = "principal"
)
