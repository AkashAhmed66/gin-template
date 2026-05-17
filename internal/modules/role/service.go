package role

import (
	"context"

	"github.com/AkashAhmed66/gin-template/internal/common/dto"
	"github.com/AkashAhmed66/gin-template/internal/common/errs"
	"github.com/AkashAhmed66/gin-template/internal/modules/permission"
)

// Service is the application-facing role API.
type Service interface {
	Create(ctx context.Context, req Request) (Response, error)
	Update(ctx context.Context, id uint64, req Request) (Response, error)
	Delete(ctx context.Context, id uint64) error
	GetByID(ctx context.Context, id uint64) (Response, error)
	Search(ctx context.Context, f Filter, page dto.PageRequest) (dto.PageResponse[Response], error)
	List(ctx context.Context) ([]Response, error)
	AssignPermissions(ctx context.Context, id uint64, names []string) (Response, error)
}

type service struct {
	repo     Repository
	permRepo permission.Repository
}

func NewService(repo Repository, permRepo permission.Repository) Service {
	return &service{repo: repo, permRepo: permRepo}
}

func (s *service) Create(ctx context.Context, req Request) (Response, error) {
	existing, err := s.repo.GetByName(ctx, req.Name)
	if err != nil {
		return Response{}, err
	}
	if existing != nil {
		return Response{}, errs.Duplicate("Role with that name already exists")
	}
	r := &Role{Name: req.Name, Description: req.Description}
	if err := s.repo.Create(ctx, r); err != nil {
		return Response{}, err
	}
	return ToResponse(*r), nil
}

func (s *service) Update(ctx context.Context, id uint64, req Request) (Response, error) {
	r, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Response{}, err
	}
	if r == nil {
		return Response{}, errs.NotFound("Role", id)
	}
	r.Name = req.Name
	r.Description = req.Description
	if err := s.repo.Update(ctx, r); err != nil {
		return Response{}, err
	}
	return ToResponse(*r), nil
}

func (s *service) Delete(ctx context.Context, id uint64) error {
	r, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if r == nil {
		return errs.NotFound("Role", id)
	}
	return s.repo.Delete(ctx, id)
}

func (s *service) GetByID(ctx context.Context, id uint64) (Response, error) {
	r, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Response{}, err
	}
	if r == nil {
		return Response{}, errs.NotFound("Role", id)
	}
	return ToResponse(*r), nil
}

func (s *service) Search(ctx context.Context, f Filter, page dto.PageRequest) (dto.PageResponse[Response], error) {
	rows, total, err := s.repo.Search(ctx, f, page.Offset(), page.Limit())
	if err != nil {
		return dto.PageResponse[Response]{}, err
	}
	return dto.NewPage(ToResponses(rows), page, total), nil
}

func (s *service) List(ctx context.Context) ([]Response, error) {
	rows, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	return ToResponses(rows), nil
}

func (s *service) AssignPermissions(ctx context.Context, id uint64, names []string) (Response, error) {
	r, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Response{}, err
	}
	if r == nil {
		return Response{}, errs.NotFound("Role", id)
	}
	perms, err := s.permRepo.FindAllByNames(ctx, names)
	if err != nil {
		return Response{}, err
	}
	if len(perms) != len(uniqueNames(names)) {
		return Response{}, errs.BadRequest("One or more permission names do not exist")
	}
	if err := s.repo.ReplacePermissions(ctx, r, perms); err != nil {
		return Response{}, err
	}
	r.Permissions = perms
	return ToResponse(*r), nil
}

func uniqueNames(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
