// Package product is the reference domain module — full CRUD with optional
// image upload. Copy this shape when adding a new module.
package product

import "github.com/AkashAhmed66/gin-template/internal/common/audit"

type Status string

const (
	StatusDraft     Status = "DRAFT"
	StatusPublished Status = "PUBLISHED"
	StatusArchived  Status = "ARCHIVED"
)

type Product struct {
	audit.BaseModel
	Name        string  `gorm:"size:200;not null;index" json:"name"`
	SKU         string  `gorm:"size:100;not null;uniqueIndex" json:"sku"`
	Description string  `gorm:"type:text" json:"description,omitempty"`
	Price       float64 `gorm:"not null" json:"price"`
	Stock       int     `gorm:"not null;default:0" json:"stock"`
	ImageURL    string  `gorm:"size:500" json:"imageUrl,omitempty"`
	Status      Status  `gorm:"size:20;not null;default:'DRAFT';index" json:"status"`
}

func (Product) TableName() string { return "products" }
