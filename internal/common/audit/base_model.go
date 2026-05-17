// Package audit holds the BaseModel that every domain entity embeds for
// automatic created/updated timestamps, blame columns, soft delete, and
// optimistic locking. Mirrors spring-boot's BaseEntity.
package audit

import (
	"time"

	"gorm.io/gorm"
)

// BaseModel is the GORM-aware base for every persistent entity.
//
// GORM populates CreatedAt / UpdatedAt automatically. CreatedBy / UpdatedBy /
// DeletedBy are filled by the AuditCallbacks installed in database.Open via
// the request-scoped username (see internal/common/security.CurrentUserKey).
type BaseModel struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt time.Time      `gorm:"not null" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"not null" json:"updatedAt"`
	CreatedBy string         `gorm:"size:100" json:"createdBy,omitempty"`
	UpdatedBy string         `gorm:"size:100" json:"updatedBy,omitempty"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`
	DeletedBy string         `gorm:"size:100" json:"deletedBy,omitempty"`
	Version   uint64         `gorm:"not null;default:0" json:"version"`
}

// MarkDeleted is a manual helper for cases where you need to soft-delete
// without using GORM's Delete (e.g. setting custom DeletedBy in a transaction).
func (b *BaseModel) MarkDeleted(by string) {
	now := time.Now().UTC()
	b.DeletedAt = gorm.DeletedAt{Time: now, Valid: true}
	b.DeletedBy = by
}

// Restore clears the soft-delete columns.
func (b *BaseModel) Restore() {
	b.DeletedAt = gorm.DeletedAt{}
	b.DeletedBy = ""
}
