package permission

// Stable permission names. Treat as a closed enum — add new constants here
// AND a matching INSERT in the next migration so RBAC stays declarative.
const (
	// User & RBAC management
	UserRead    = "USER_READ"
	UserWrite   = "USER_WRITE"
	UserDelete  = "USER_DELETE"
	RoleRead    = "ROLE_READ"
	RoleWrite   = "ROLE_WRITE"
	RoleDelete  = "ROLE_DELETE"
	PermissionRead   = "PERMISSION_READ"
	PermissionWrite  = "PERMISSION_WRITE"
	PermissionDelete = "PERMISSION_DELETE"

	// Sessions
	SessionRead   = "SESSION_READ"
	SessionRevoke = "SESSION_REVOKE"

	// Audit
	AuditRead = "AUDIT_READ"

	// Admin override — coarse-grained scope expansion. Holders bypass per-record
	// ownership checks at the service layer (see SecurityUtils.HasAnyAuthority).
	AdminRead   = "ADMIN_READ"
	AdminWrite  = "ADMIN_WRITE"
	AdminEdit   = "ADMIN_EDIT"
	AdminDelete = "ADMIN_DELETE"

	// Impersonation
	UserImpersonate = "USER_IMPERSONATE"

	// Reference domain — Product
	ProductRead   = "PRODUCT_READ"
	ProductWrite  = "PRODUCT_WRITE"
	ProductDelete = "PRODUCT_DELETE"
)

// AdminAny is the OR-set of all four admin override permissions. Pair with
// security.Principal.HasAnyAuthority(AdminAny...) at service-layer checks
// where any admin permission should expand scope.
var AdminAny = []string{AdminRead, AdminWrite, AdminEdit, AdminDelete}

// All is every named permission. Used by the bootstrap admin to grant a full
// catalogue without manual enumeration.
var All = []string{
	UserRead, UserWrite, UserDelete,
	RoleRead, RoleWrite, RoleDelete,
	PermissionRead, PermissionWrite, PermissionDelete,
	SessionRead, SessionRevoke,
	AuditRead,
	AdminRead, AdminWrite, AdminEdit, AdminDelete,
	UserImpersonate,
	ProductRead, ProductWrite, ProductDelete,
}
