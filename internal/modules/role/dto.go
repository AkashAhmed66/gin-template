package role

import "github.com/AkashAhmed66/gin-template/internal/common/dto"

type Request struct {
	Name        string `json:"name" binding:"required,min=2,max=64"`
	Description string `json:"description" binding:"max=255"`
}

type AssignPermissionsRequest struct {
	Permissions []string `json:"permissions" binding:"required,dive,required"`
}

type Response struct {
	ID          uint64   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Permissions []string `json:"permissions"`
	dto.BaseResponse
}

type Filter struct {
	Q string `form:"q"`
}

// ToResponse converts a Role to its public DTO (permissions flattened to names).
func ToResponse(r Role) Response {
	perms := make([]string, len(r.Permissions))
	for i, p := range r.Permissions {
		perms[i] = p.Name
	}
	resp := Response{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		Permissions: perms,
	}
	resp.CreatedAt = r.CreatedAt
	resp.UpdatedAt = r.UpdatedAt
	resp.CreatedBy = r.CreatedBy
	resp.UpdatedBy = r.UpdatedBy
	if r.DeletedAt.Valid {
		t := r.DeletedAt.Time
		resp.DeletedAt = &t
		resp.DeletedBy = r.DeletedBy
	}
	return resp
}

func ToResponses(rs []Role) []Response {
	out := make([]Response, len(rs))
	for i, r := range rs {
		out[i] = ToResponse(r)
	}
	return out
}
