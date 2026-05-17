package role

import (
	"context"
	"errors"

	"github.com/AkashAhmed66/gin-template/internal/modules/permission"
	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, r *Role) error
	Update(ctx context.Context, r *Role) error
	Delete(ctx context.Context, id uint64) error
	GetByID(ctx context.Context, id uint64) (*Role, error)
	GetByIDs(ctx context.Context, ids []uint64) ([]Role, error)
	GetByName(ctx context.Context, name string) (*Role, error)
	Search(ctx context.Context, f Filter, offset, limit int) ([]Role, int64, error)
	List(ctx context.Context) ([]Role, error)
	ReplacePermissions(ctx context.Context, r *Role, perms []permission.Permission) error
}

type gormRepo struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) Repository { return &gormRepo{db: db} }

func (r *gormRepo) Create(ctx context.Context, e *Role) error {
	return r.db.WithContext(ctx).Create(e).Error
}
func (r *gormRepo) Update(ctx context.Context, e *Role) error {
	return r.db.WithContext(ctx).Save(e).Error
}
func (r *gormRepo) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&Role{}, id).Error
}
func (r *gormRepo) GetByID(ctx context.Context, id uint64) (*Role, error) {
	var e Role
	err := r.db.WithContext(ctx).Preload("Permissions").First(&e, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &e, nil
}
func (r *gormRepo) GetByIDs(ctx context.Context, ids []uint64) ([]Role, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var out []Role
	err := r.db.WithContext(ctx).Preload("Permissions").Where("id IN ?", ids).Find(&out).Error
	return out, err
}
func (r *gormRepo) GetByName(ctx context.Context, name string) (*Role, error) {
	var e Role
	err := r.db.WithContext(ctx).Preload("Permissions").Where("name = ?", name).First(&e).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &e, nil
}
func (r *gormRepo) Search(ctx context.Context, f Filter, offset, limit int) ([]Role, int64, error) {
	var (
		rows  []Role
		total int64
	)
	q := r.db.WithContext(ctx).Model(&Role{})
	if f.Q != "" {
		like := "%" + f.Q + "%"
		q = q.Where("name LIKE ? OR description LIKE ?", like, like)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Preload("Permissions").Order("name ASC").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}
func (r *gormRepo) List(ctx context.Context) ([]Role, error) {
	var rows []Role
	err := r.db.WithContext(ctx).Preload("Permissions").Order("name ASC").Find(&rows).Error
	return rows, err
}
func (r *gormRepo) ReplacePermissions(ctx context.Context, e *Role, perms []permission.Permission) error {
	return r.db.WithContext(ctx).Model(e).Association("Permissions").Replace(perms)
}
