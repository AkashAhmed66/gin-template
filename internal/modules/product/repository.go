package product

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, p *Product) error
	Update(ctx context.Context, p *Product) error
	Delete(ctx context.Context, id uint64) error
	GetByID(ctx context.Context, id uint64) (*Product, error)
	GetBySKU(ctx context.Context, sku string) (*Product, error)
	Search(ctx context.Context, f Filter, offset, limit int) ([]Product, int64, error)
}

type gormRepo struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) Repository { return &gormRepo{db: db} }

func (r *gormRepo) Create(ctx context.Context, p *Product) error {
	return r.db.WithContext(ctx).Create(p).Error
}
func (r *gormRepo) Update(ctx context.Context, p *Product) error {
	return r.db.WithContext(ctx).Save(p).Error
}
func (r *gormRepo) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&Product{}, id).Error
}
func (r *gormRepo) GetByID(ctx context.Context, id uint64) (*Product, error) {
	var p Product
	err := r.db.WithContext(ctx).First(&p, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}
func (r *gormRepo) GetBySKU(ctx context.Context, sku string) (*Product, error) {
	var p Product
	err := r.db.WithContext(ctx).Where("sku = ?", sku).First(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}
func (r *gormRepo) Search(ctx context.Context, f Filter, offset, limit int) ([]Product, int64, error) {
	var (
		rows  []Product
		total int64
	)
	q := r.db.WithContext(ctx).Model(&Product{})
	if f.Q != "" {
		like := "%" + f.Q + "%"
		q = q.Where("name LIKE ? OR sku LIKE ? OR description LIKE ?", like, like, like)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.MinP != nil {
		q = q.Where("price >= ?", *f.MinP)
	}
	if f.MaxP != nil {
		q = q.Where("price <= ?", *f.MaxP)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("id DESC").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}
