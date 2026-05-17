// Package permission owns the Permission entity and the named-constant
// catalogue. Permissions are the leaves of the RBAC tree — every controller
// gate references one.
package permission

import "github.com/AkashAhmed66/gin-template/internal/common/audit"

// Permission is a named capability gate. Names are stable, machine-readable
// strings (see Names).
type Permission struct {
	audit.BaseModel
	Name        string `gorm:"size:100;not null;uniqueIndex" json:"name"`
	Description string `gorm:"size:255" json:"description"`
}

func (Permission) TableName() string { return "permissions" }
