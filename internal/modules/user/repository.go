package user

import (
	"context"
	"errors"

	"github.com/AkashAhmed66/gin-template/internal/modules/role"
	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, u *User) error
	Update(ctx context.Context, u *User) error
	Delete(ctx context.Context, id uint64) error
	GetByID(ctx context.Context, id uint64) (*User, error)
	GetByUsernameOrEmail(ctx context.Context, identifier string) (*User, error)
	ExistsByUsername(ctx context.Context, username string) (bool, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	Search(ctx context.Context, f Filter, offset, limit int) ([]User, int64, error)
	ReplaceRoles(ctx context.Context, u *User, roles []role.Role) error
	IncrementTokenVersion(ctx context.Context, id uint64) error
}

type gormRepo struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) Repository { return &gormRepo{db: db} }

func (r *gormRepo) Create(ctx context.Context, u *User) error {
	return r.db.WithContext(ctx).Create(u).Error
}
func (r *gormRepo) Update(ctx context.Context, u *User) error {
	return r.db.WithContext(ctx).Save(u).Error
}
func (r *gormRepo) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&User{}, id).Error
}
func (r *gormRepo) GetByID(ctx context.Context, id uint64) (*User, error) {
	var u User
	err := r.db.WithContext(ctx).
		Preload("Roles.Permissions").
		First(&u, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}
func (r *gormRepo) GetByUsernameOrEmail(ctx context.Context, identifier string) (*User, error) {
	var u User
	err := r.db.WithContext(ctx).
		Preload("Roles.Permissions").
		Where("username = ? OR email = ?", identifier, identifier).
		First(&u).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}
func (r *gormRepo) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&User{}).Where("username = ?", username).Count(&n).Error
	return n > 0, err
}
func (r *gormRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&User{}).Where("email = ?", email).Count(&n).Error
	return n > 0, err
}
func (r *gormRepo) Search(ctx context.Context, f Filter, offset, limit int) ([]User, int64, error) {
	var (
		rows  []User
		total int64
	)
	q := r.db.WithContext(ctx).Model(&User{})
	if f.Q != "" {
		like := "%" + f.Q + "%"
		q = q.Where("username LIKE ? OR email LIKE ? OR full_name LIKE ?", like, like, like)
	}
	if f.Enabled != nil {
		q = q.Where("enabled = ?", *f.Enabled)
	}
	if f.Role != "" {
		q = q.Joins("JOIN user_roles ur ON ur.user_id = users.id").
			Joins("JOIN roles r ON r.id = ur.role_id").
			Where("r.name = ?", f.Role)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := q.Preload("Roles.Permissions").
		Order("users.id DESC").
		Offset(offset).Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}
func (r *gormRepo) ReplaceRoles(ctx context.Context, u *User, roles []role.Role) error {
	return r.db.WithContext(ctx).Model(u).Association("Roles").Replace(roles)
}
func (r *gormRepo) IncrementTokenVersion(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Model(&User{}).
		Where("id = ?", id).
		Update("token_version", gorm.Expr("token_version + 1")).Error
}
