package permission

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// Repository is the persistence interface for permissions.
type Repository interface {
	Create(ctx context.Context, p *Permission) error
	Update(ctx context.Context, p *Permission) error
	Delete(ctx context.Context, id uint64) error
	GetByID(ctx context.Context, id uint64) (*Permission, error)
	GetByName(ctx context.Context, name string) (*Permission, error)
	FindAllByNames(ctx context.Context, names []string) ([]Permission, error)
	Search(ctx context.Context, f Filter, offset, limit int) ([]Permission, int64, error)
	List(ctx context.Context) ([]Permission, error)
}

type gormRepo struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) Repository { return &gormRepo{db: db} }

func (r *gormRepo) Create(ctx context.Context, p *Permission) error {
	return r.db.WithContext(ctx).Create(p).Error
}
func (r *gormRepo) Update(ctx context.Context, p *Permission) error {
	return r.db.WithContext(ctx).Save(p).Error
}
func (r *gormRepo) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&Permission{}, id).Error
}
func (r *gormRepo) GetByID(ctx context.Context, id uint64) (*Permission, error) {
	var p Permission
	err := r.db.WithContext(ctx).First(&p, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}
func (r *gormRepo) GetByName(ctx context.Context, name string) (*Permission, error) {
	var p Permission
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}
func (r *gormRepo) FindAllByNames(ctx context.Context, names []string) ([]Permission, error) {
	if len(names) == 0 {
		return nil, nil
	}
	var out []Permission
	err := r.db.WithContext(ctx).Where("name IN ?", names).Find(&out).Error
	return out, err
}
func (r *gormRepo) Search(ctx context.Context, f Filter, offset, limit int) ([]Permission, int64, error) {
	var (
		rows  []Permission
		total int64
	)
	q := r.db.WithContext(ctx).Model(&Permission{})
	if f.Q != "" {
		like := "%" + f.Q + "%"
		q = q.Where("name LIKE ? OR description LIKE ?", like, like)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Order("name ASC").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}
func (r *gormRepo) List(ctx context.Context) ([]Permission, error) {
	var rows []Permission
	err := r.db.WithContext(ctx).Order("name ASC").Find(&rows).Error
	return rows, err
}
