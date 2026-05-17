package dto

import "time"

// BaseResponse mirrors common/audit.BaseModel — embed it in every entity-backed
// response DTO so the audit/lifecycle columns flow through automatically.
type BaseResponse struct {
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	CreatedBy string     `json:"createdBy,omitempty"`
	UpdatedBy string     `json:"updatedBy,omitempty"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
	DeletedBy string     `json:"deletedBy,omitempty"`
}
