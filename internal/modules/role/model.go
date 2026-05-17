// Package role owns the Role entity. Roles are sets of permissions; users are
// assigned roles, and authority resolution flattens role.permissions ∪ user.roles
// into the JWT's `authorities` claim.
package role

import (
	"github.com/AkashAhmed66/gin-template/internal/common/audit"
	"github.com/AkashAhmed66/gin-template/internal/modules/permission"
)

// Stable role names — extend by adding constants and inserting rows in a new
// migration.
const (
	NameAdmin = "ADMIN"
	NameUser  = "USER"
)

// Role groups permissions and gets assigned to users (many-to-many on both sides).
type Role struct {
	audit.BaseModel
	Name        string                  `gorm:"size:64;not null;uniqueIndex" json:"name"`
	Description string                  `gorm:"size:255" json:"description"`
	Permissions []permission.Permission `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
}

func (Role) TableName() string { return "roles" }
