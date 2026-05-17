package user

import "github.com/AkashAhmed66/gin-template/internal/common/dto"

type UpdateRequest struct {
	Email    string `json:"email" binding:"omitempty,email,max=255"`
	FullName string `json:"fullName" binding:"max=200"`
	Enabled  *bool  `json:"enabled"`
}

type AssignRolesRequest struct {
	Roles []string `json:"roles" binding:"required,dive,required"`
}

type Response struct {
	ID          uint64   `json:"id"`
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	FullName    string   `json:"fullName,omitempty"`
	Enabled     bool     `json:"enabled"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions,omitempty"`
	dto.BaseResponse
}

type Filter struct {
	Q       string `form:"q"`
	Role    string `form:"role"`
	Enabled *bool  `form:"enabled"`
}

// ToResponse converts a User to its public DTO, flattening role names and
// (optionally) the union of permissions across the user's roles.
func ToResponse(u User) Response {
	roles := make([]string, len(u.Roles))
	permSet := map[string]struct{}{}
	for i, r := range u.Roles {
		roles[i] = r.Name
		for _, p := range r.Permissions {
			permSet[p.Name] = struct{}{}
		}
	}
	perms := make([]string, 0, len(permSet))
	for name := range permSet {
		perms = append(perms, name)
	}
	r := Response{
		ID:          u.ID,
		Username:    u.Username,
		Email:       u.Email,
		FullName:    u.FullName,
		Enabled:     u.Enabled,
		Roles:       roles,
		Permissions: perms,
	}
	r.CreatedAt = u.CreatedAt
	r.UpdatedAt = u.UpdatedAt
	r.CreatedBy = u.CreatedBy
	r.UpdatedBy = u.UpdatedBy
	if u.DeletedAt.Valid {
		t := u.DeletedAt.Time
		r.DeletedAt = &t
		r.DeletedBy = u.DeletedBy
	}
	return r
}

func ToResponses(us []User) []Response {
	out := make([]Response, len(us))
	for i, u := range us {
		out[i] = ToResponse(u)
	}
	return out
}

// AuthoritiesOf returns the union of role names (each prefixed with "ROLE_")
// and the flattened permission names. This is what gets baked into the JWT.
func AuthoritiesOf(u User) []string {
	out := make([]string, 0)
	seen := map[string]struct{}{}
	for _, r := range u.Roles {
		key := "ROLE_" + r.Name
		if _, ok := seen[key]; !ok {
			out = append(out, key)
			seen[key] = struct{}{}
		}
		for _, p := range r.Permissions {
			if _, ok := seen[p.Name]; !ok {
				out = append(out, p.Name)
				seen[p.Name] = struct{}{}
			}
		}
	}
	return out
}
