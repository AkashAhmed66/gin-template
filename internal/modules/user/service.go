package user

import (
	"context"

	"github.com/AkashAhmed66/gin-template/internal/common/dto"
	"github.com/AkashAhmed66/gin-template/internal/common/errs"
	"github.com/AkashAhmed66/gin-template/internal/modules/role"
)

// SessionRevoker is implemented by the session service; injecting it as an
// interface keeps the import graph acyclic (user → sessions would otherwise
// circle back via auth).
type SessionRevoker interface {
	RevokeAllForUser(ctx context.Context, userID uint64, reason string) error
}

// Service is the application-facing user API.
type Service interface {
	GetByID(ctx context.Context, id uint64) (Response, error)
	Search(ctx context.Context, f Filter, page dto.PageRequest) (dto.PageResponse[Response], error)
	Update(ctx context.Context, id uint64, req UpdateRequest) (Response, error)
	Delete(ctx context.Context, id uint64) error
	Activate(ctx context.Context, id uint64) (Response, error)
	Deactivate(ctx context.Context, id, callerID uint64) (Response, error)
	ForceLogout(ctx context.Context, id uint64) error
	AssignRoles(ctx context.Context, id uint64, roleNames []string) (Response, error)
}

type service struct {
	repo     Repository
	roleRepo role.Repository
	sessions SessionRevoker
}

func NewService(repo Repository, roleRepo role.Repository, sessions SessionRevoker) Service {
	return &service{repo: repo, roleRepo: roleRepo, sessions: sessions}
}

func (s *service) GetByID(ctx context.Context, id uint64) (Response, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Response{}, err
	}
	if u == nil {
		return Response{}, errs.NotFound("User", id)
	}
	return ToResponse(*u), nil
}

func (s *service) Search(ctx context.Context, f Filter, page dto.PageRequest) (dto.PageResponse[Response], error) {
	rows, total, err := s.repo.Search(ctx, f, page.Offset(), page.Limit())
	if err != nil {
		return dto.PageResponse[Response]{}, err
	}
	return dto.NewPage(ToResponses(rows), page, total), nil
}

func (s *service) Update(ctx context.Context, id uint64, req UpdateRequest) (Response, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Response{}, err
	}
	if u == nil {
		return Response{}, errs.NotFound("User", id)
	}
	if req.Email != "" && req.Email != u.Email {
		exists, err := s.repo.ExistsByEmail(ctx, req.Email)
		if err != nil {
			return Response{}, err
		}
		if exists {
			return Response{}, errs.Duplicate("Email already in use")
		}
		u.Email = req.Email
	}
	if req.FullName != "" {
		u.FullName = req.FullName
	}
	wasEnabled := u.Enabled
	if req.Enabled != nil {
		u.Enabled = *req.Enabled
	}
	if err := s.repo.Update(ctx, u); err != nil {
		return Response{}, err
	}
	if wasEnabled && !u.Enabled && s.sessions != nil {
		_ = s.sessions.RevokeAllForUser(ctx, u.ID, "account-deactivated")
		_ = s.repo.IncrementTokenVersion(ctx, u.ID)
	}
	return ToResponse(*u), nil
}

func (s *service) Delete(ctx context.Context, id uint64) error {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if u == nil {
		return errs.NotFound("User", id)
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	if s.sessions != nil {
		_ = s.sessions.RevokeAllForUser(ctx, id, "account-deleted")
	}
	return nil
}

func (s *service) Activate(ctx context.Context, id uint64) (Response, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Response{}, err
	}
	if u == nil {
		return Response{}, errs.NotFound("User", id)
	}
	u.Enabled = true
	if err := s.repo.Update(ctx, u); err != nil {
		return Response{}, err
	}
	return ToResponse(*u), nil
}

func (s *service) Deactivate(ctx context.Context, id, callerID uint64) (Response, error) {
	if id == callerID {
		return Response{}, errs.BadRequest("Cannot deactivate your own account")
	}
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Response{}, err
	}
	if u == nil {
		return Response{}, errs.NotFound("User", id)
	}
	u.Enabled = false
	if err := s.repo.Update(ctx, u); err != nil {
		return Response{}, err
	}
	if s.sessions != nil {
		_ = s.sessions.RevokeAllForUser(ctx, id, "account-deactivated")
	}
	_ = s.repo.IncrementTokenVersion(ctx, id)
	return ToResponse(*u), nil
}

func (s *service) ForceLogout(ctx context.Context, id uint64) error {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if u == nil {
		return errs.NotFound("User", id)
	}
	if s.sessions != nil {
		_ = s.sessions.RevokeAllForUser(ctx, id, "admin-force-logout")
	}
	return s.repo.IncrementTokenVersion(ctx, id)
}

func (s *service) AssignRoles(ctx context.Context, id uint64, roleNames []string) (Response, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Response{}, err
	}
	if u == nil {
		return Response{}, errs.NotFound("User", id)
	}
	resolved := make([]role.Role, 0, len(roleNames))
	seen := map[string]struct{}{}
	for _, name := range roleNames {
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		r, err := s.roleRepo.GetByName(ctx, name)
		if err != nil {
			return Response{}, err
		}
		if r == nil {
			return Response{}, errs.BadRequest("Role does not exist: " + name)
		}
		resolved = append(resolved, *r)
	}
	if err := s.repo.ReplaceRoles(ctx, u, resolved); err != nil {
		return Response{}, err
	}
	u.Roles = resolved
	if s.sessions != nil {
		_ = s.sessions.RevokeAllForUser(ctx, id, "roles-changed")
	}
	_ = s.repo.IncrementTokenVersion(ctx, id)
	return ToResponse(*u), nil
}
