package product

import (
	"context"

	"github.com/AkashAhmed66/gin-template/internal/common/dto"
	"github.com/AkashAhmed66/gin-template/internal/common/errs"
)

type Service interface {
	Create(ctx context.Context, req Request) (Response, error)
	Update(ctx context.Context, id uint64, req Request) (Response, error)
	Delete(ctx context.Context, id uint64) error
	GetByID(ctx context.Context, id uint64) (Response, error)
	Search(ctx context.Context, f Filter, page dto.PageRequest) (dto.PageResponse[Response], error)
}

type service struct{ repo Repository }

func NewService(repo Repository) Service { return &service{repo: repo} }

func (s *service) Create(ctx context.Context, req Request) (Response, error) {
	existing, err := s.repo.GetBySKU(ctx, req.SKU)
	if err != nil {
		return Response{}, err
	}
	if existing != nil {
		return Response{}, errs.Duplicate("Product with SKU '" + req.SKU + "' already exists")
	}
	status := req.Status
	if status == "" {
		status = StatusDraft
	}
	p := &Product{
		Name:        req.Name,
		SKU:         req.SKU,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
		ImageURL:    req.ImageURL,
		Status:      status,
	}
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
		return Response{}, errs.NotFound("Product", id)
	}
	if req.SKU != p.SKU {
		existing, err := s.repo.GetBySKU(ctx, req.SKU)
		if err != nil {
			return Response{}, err
		}
		if existing != nil && existing.ID != id {
			return Response{}, errs.Duplicate("SKU already in use")
		}
	}
	p.Name = req.Name
	p.SKU = req.SKU
	p.Description = req.Description
	p.Price = req.Price
	p.Stock = req.Stock
	if req.ImageURL != "" {
		p.ImageURL = req.ImageURL
	}
	if req.Status != "" {
		p.Status = req.Status
	}
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
		return errs.NotFound("Product", id)
	}
	return s.repo.Delete(ctx, id)
}

func (s *service) GetByID(ctx context.Context, id uint64) (Response, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Response{}, err
	}
	if p == nil {
		return Response{}, errs.NotFound("Product", id)
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
