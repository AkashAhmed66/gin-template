package product

import (
	"context"

	"github.com/AkashAhmed66/gin-template/internal/common/applog"
	"github.com/AkashAhmed66/gin-template/internal/common/dto"
	"github.com/AkashAhmed66/gin-template/internal/common/errs"
	"go.uber.org/zap"
)

// plog produces a request-scoped logger that writes to logs/products.log.
// Declared once so call sites just write plog(ctx).Info(...).
var plog = applog.Channel("products.log")

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
	log := plog(ctx)
	existing, err := s.repo.GetBySKU(ctx, req.SKU)
	if err != nil {
		log.Error("create: lookup by sku failed", zap.String("sku", req.SKU), zap.Error(err))
		return Response{}, err
	}
	if existing != nil {
		log.Warn("create: duplicate sku rejected", zap.String("sku", req.SKU))
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
		log.Error("create: repo insert failed", zap.String("sku", req.SKU), zap.Error(err))
		return Response{}, err
	}
	log.Info("create: product created",
		zap.Uint64("productId", p.ID),
		zap.String("sku", p.SKU),
		zap.String("status", string(p.Status)),
		zap.Float64("price", p.Price),
	)
	return ToResponse(*p), nil
}

func (s *service) Update(ctx context.Context, id uint64, req Request) (Response, error) {
	log := plog(ctx).With(zap.Uint64("productId", id))
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.Error("update: lookup failed", zap.Error(err))
		return Response{}, err
	}
	if p == nil {
		log.Warn("update: product not found")
		return Response{}, errs.NotFound("Product", id)
	}
	if req.SKU != p.SKU {
		log.Info("update: sku change requested",
			zap.String("oldSku", p.SKU),
			zap.String("newSku", req.SKU),
		)
		existing, err := s.repo.GetBySKU(ctx, req.SKU)
		if err != nil {
			log.Error("update: sku lookup failed", zap.String("sku", req.SKU), zap.Error(err))
			return Response{}, err
		}
		if existing != nil && existing.ID != id {
			log.Warn("update: sku collision rejected", zap.String("sku", req.SKU))
			return Response{}, errs.Duplicate("SKU already in use")
		}
	}
	oldStatus := p.Status
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
		log.Error("update: repo save failed", zap.Error(err))
		return Response{}, err
	}
	log.Info("update: product updated",
		zap.String("sku", p.SKU),
		zap.String("oldStatus", string(oldStatus)),
		zap.String("newStatus", string(p.Status)),
	)
	return ToResponse(*p), nil
}

func (s *service) Delete(ctx context.Context, id uint64) error {
	log := plog(ctx).With(zap.Uint64("productId", id))
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.Error("delete: lookup failed", zap.Error(err))
		return err
	}
	if p == nil {
		log.Warn("delete: product not found")
		return errs.NotFound("Product", id)
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		log.Error("delete: repo delete failed", zap.Error(err))
		return err
	}
	log.Info("delete: product deleted", zap.String("sku", p.SKU))
	return nil
}

func (s *service) GetByID(ctx context.Context, id uint64) (Response, error) {
	log := plog(ctx).With(zap.Uint64("productId", id))
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.Error("getById: lookup failed", zap.Error(err))
		return Response{}, err
	}
	if p == nil {
		log.Debug("getById: product not found")
		return Response{}, errs.NotFound("Product", id)
	}
	log.Debug("getById: product fetched", zap.String("sku", p.SKU))
	return ToResponse(*p), nil
}

func (s *service) Search(ctx context.Context, f Filter, page dto.PageRequest) (dto.PageResponse[Response], error) {
	log := plog(ctx)
	rows, total, err := s.repo.Search(ctx, f, page.Offset(), page.Limit())
	if err != nil {
		log.Error("search: repo query failed",
			zap.String("q", f.Q),
			zap.String("status", string(f.Status)),
			zap.Int("page", page.Page),
			zap.Int("size", page.Size),
			zap.Error(err),
		)
		return dto.PageResponse[Response]{}, err
	}
	log.Debug("search: results returned",
		zap.String("q", f.Q),
		zap.Int("page", page.Page),
		zap.Int("size", page.Size),
		zap.Int("returned", len(rows)),
		zap.Int64("total", total),
	)
	return dto.NewPage(ToResponses(rows), page, total), nil
}
