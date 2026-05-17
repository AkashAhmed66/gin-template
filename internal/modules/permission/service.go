package permission

import (
	"context"

	"github.com/AkashAhmed66/gin-template/internal/common/dto"
	"github.com/AkashAhmed66/gin-template/internal/common/errs"
)

// Service is the application-facing permission API.
type Service interface {
	Create(ctx context.Context, req Request) (Response, error)
	Update(ctx context.Context, id uint64, req Request) (Response, error)
	Delete(ctx context.Context, id uint64) error
	GetByID(ctx context.Context, id uint64) (Response, error)
	Search(ctx context.Context, f Filter, page dto.PageRequest) (dto.PageResponse[Response], error)
	List(ctx context.Context) ([]Response, error)
}

type service struct{ repo Repository }

func NewService(repo Repository) Service { return &service{repo: repo} }

func (s *service) Create(ctx context.Context, req Request) (Response, error) {
	existing, err := s.repo.GetByName(ctx, req.Name)
	if err != nil {
		return Response{}, err
	}
	if existing != nil {
		return Response{}, errs.Duplicate("Permission with that name already exists")
	}
	p := &Permission{Name: req.Name, Description: req.Description}
	if err := s.repo.Create(ctx, p); err != nil {
		return Response{}, err
	}
	return ToResponse(*p), nil
}

func (s *service) Update(ctx context.Context, id uint64, req Request) (Response, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Response{}, err
	}
	if p == nil {
		return Response{}, errs.NotFound("Permission", id)
	}
	p.Name = req.Name
	p.Description = req.Description
	if err := s.repo.Update(ctx, p); err != nil {
		return Response{}, err
	}
	return ToResponse(*p), nil
}

func (s *service) Delete(ctx context.Context, id uint64) error {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if p == nil {
		return errs.NotFound("Permission", id)
	}
	return s.repo.Delete(ctx, id)
}

func (s *service) GetByID(ctx context.Context, id uint64) (Response, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Response{}, err
	}
	if p == nil {
		return Response{}, errs.NotFound("Permission", id)
	}
	return ToResponse(*p), nil
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
