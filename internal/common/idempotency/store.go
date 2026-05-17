// Package idempotency persists request/response pairs keyed by the
// Idempotency-Key header so retries within TTL replay the cached response
// instead of executing the side effect twice. Mirrors spring-boot's
// IdempotencyAspect + IdempotencyStore.
package idempotency

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// Record is the persisted idempotency entry. Each (key, method, path, userID)
// tuple maps to exactly one historical response that's replayed on retry.
type Record struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Key          string    `gorm:"size:200;not null;uniqueIndex:idx_idem_key,priority:1" json:"key"`
	Method       string    `gorm:"size:10;not null;uniqueIndex:idx_idem_key,priority:2" json:"method"`
	Path         string    `gorm:"size:500;not null;uniqueIndex:idx_idem_key,priority:3" json:"path"`
	UserID       uint64    `gorm:"not null;uniqueIndex:idx_idem_key,priority:4" json:"userId"`
	RequestHash  string    `gorm:"size:128;not null" json:"requestHash"`
	StatusCode   int       `gorm:"not null" json:"statusCode"`
	ResponseBody []byte    `gorm:"type:bytea" json:"-"`
	CreatedAt    time.Time `gorm:"not null;index" json:"createdAt"`
	ExpiresAt    time.Time `gorm:"not null;index" json:"expiresAt"`
}

// TableName overrides the default GORM pluralization.
func (Record) TableName() string { return "idempotency_records" }

// Store is the persistence interface used by the middleware.
type Store interface {
	Find(ctx context.Context, key, method, path string, userID uint64) (*Record, error)
	Save(ctx context.Context, r *Record) error
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
}

type gormStore struct{ db *gorm.DB }

// NewStore returns a GORM-backed Store.
func NewStore(db *gorm.DB) Store { return &gormStore{db: db} }

func (s *gormStore) Find(ctx context.Context, key, method, path string, userID uint64) (*Record, error) {
	var r Record
	err := s.db.WithContext(ctx).
		Where("key = ? AND method = ? AND path = ? AND user_id = ?", key, method, path, userID).
		Where("expires_at > ?", time.Now().UTC()).
		First(&r).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &r, nil
}

func (s *gormStore) Save(ctx context.Context, r *Record) error {
	return s.db.WithContext(ctx).Create(r).Error
}

func (s *gormStore) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	res := s.db.WithContext(ctx).Where("expires_at <= ?", before).Delete(&Record{})
	return res.RowsAffected, res.Error
}
