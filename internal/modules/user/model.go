// Package user owns the User entity. Users hold credentials, are assigned
// roles (many-to-many), and may be deactivated independently of deletion.
//
// TokenVersion is bumped on every credential change (password reset, role
// change, force-logout) so any access token issued before the bump fails
// validation even if the session row is still alive — defense in depth.
package user

import (
	"github.com/AkashAhmed66/gin-template/internal/common/audit"
	"github.com/AkashAhmed66/gin-template/internal/modules/role"
)

type User struct {
	audit.BaseModel
	Username     string      `gorm:"size:64;not null;uniqueIndex" json:"username"`
	Email        string      `gorm:"size:255;not null;uniqueIndex" json:"email"`
	PasswordHash string      `gorm:"size:255;not null" json:"-"`
	FullName     string      `gorm:"size:200" json:"fullName,omitempty"`
	Enabled      bool        `gorm:"not null;default:true" json:"enabled"`
	TokenVersion uint64      `gorm:"not null;default:0" json:"-"`
	Roles        []role.Role `gorm:"many2many:user_roles;" json:"roles,omitempty"`
}

func (User) TableName() string { return "users" }
