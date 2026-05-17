package permission

import "github.com/AkashAhmed66/gin-template/internal/common/dto"

type Request struct {
	Name        string `json:"name" binding:"required,min=2,max=100"`
	Description string `json:"description" binding:"max=255"`
}

type Response struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	dto.BaseResponse
}

type Filter struct {
	Q string `form:"q"`
}

// ToResponse converts a Permission to its public DTO.
func ToResponse(p Permission) Response {
	r := Response{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
	}
	r.CreatedAt = p.CreatedAt
	r.UpdatedAt = p.UpdatedAt
	r.CreatedBy = p.CreatedBy
	r.UpdatedBy = p.UpdatedBy
	if p.DeletedAt.Valid {
		t := p.DeletedAt.Time
		r.DeletedAt = &t
		r.DeletedBy = p.DeletedBy
	}
	return r
}

// ToResponses converts a slice.
func ToResponses(ps []Permission) []Response {
	out := make([]Response, len(ps))
	for i, p := range ps {
		out[i] = ToResponse(p)
	}
	return out
}
