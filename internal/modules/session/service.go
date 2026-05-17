package session

import (
	"context"
	"errors"
	"time"

	"github.com/AkashAhmed66/gin-template/internal/common/dto"
	"github.com/AkashAhmed66/gin-template/internal/common/errs"
	"gorm.io/gorm"
)

// Service is the per-device session API used by auth + user modules.
type Service interface {
	// Create persists a new session and returns the row (so the caller can
	// embed its id in the JWT). impersonatorID may be 0.
	Create(ctx context.Context, userID, impersonatorID uint64, expiresAt time.Time, deviceName, userAgent, ip string) (*UserSession, error)
	// Validate returns nil if the session exists, belongs to userID, and is active.
	Validate(sessionID, userID uint64) error
	// Touch updates LastUsedAt. Failures are swallowed — this is best-effort.
	Touch(sessionID uint64)
	// Revoke flags one session inactive.
	Revoke(ctx context.Context, sessionID uint64, reason string) error
	// RevokeAllForUser revokes every active session for a user.
	RevokeAllForUser(ctx context.Context, userID uint64, reason string) error
	// GetByID returns a single session or NotFound.
	GetByID(ctx context.Context, id uint64) (*UserSession, error)
	// ListForUser returns non-revoked sessions for one user.
	ListForUser(ctx context.Context, userID, currentID uint64) ([]Response, error)
	// Search returns a paginated cross-user list (admin endpoint).
	Search(ctx context.Context, f Filter, page dto.PageRequest, currentID uint64) (dto.PageResponse[Response], error)
	// CleanupExpired purges rows past expiry/revocation retention.
	CleanupExpired(ctx context.Context, retention time.Duration) (int64, error)
}

type service struct{ db *gorm.DB }

func NewService(db *gorm.DB) Service { return &service{db: db} }

func (s *service) Create(ctx context.Context, userID, impersonatorID uint64, expiresAt time.Time, deviceName, userAgent, ip string) (*UserSession, error) {
	now := time.Now().UTC()
	row := &UserSession{
		UserID:         userID,
		ImpersonatorID: impersonatorID,
		DeviceName:     deviceName,
		UserAgent:      userAgent,
		IPAddress:      ip,
		IssuedAt:       now,
		LastUsedAt:     now,
		ExpiresAt:      expiresAt,
	}
	if err := s.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

func (s *service) Validate(sessionID, userID uint64) error {
	var row UserSession
	err := s.db.First(&row, sessionID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.Unauthorized("Session not found")
		}
		return err
	}
	if row.UserID != userID {
		return errs.Unauthorized("Session does not match user")
	}
	if !row.Active() {
		return errs.Unauthorized("Session is no longer active")
	}
	return nil
}

func (s *service) Touch(sessionID uint64) {
	_ = s.db.Model(&UserSession{}).
		Where("id = ?", sessionID).
		Update("last_used_at", time.Now().UTC()).Error
}

func (s *service) Revoke(ctx context.Context, sessionID uint64, reason string) error {
	now := time.Now().UTC()
	return s.db.WithContext(ctx).Model(&UserSession{}).
		Where("id = ? AND revoked_at IS NULL", sessionID).
		Updates(map[string]any{
			"revoked_at":     &now,
			"revoked_reason": reason,
		}).Error
}

func (s *service) RevokeAllForUser(ctx context.Context, userID uint64, reason string) error {
	now := time.Now().UTC()
	return s.db.WithContext(ctx).Model(&UserSession{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Updates(map[string]any{
			"revoked_at":     &now,
			"revoked_reason": reason,
		}).Error
}

func (s *service) GetByID(ctx context.Context, id uint64) (*UserSession, error) {
	var row UserSession
	err := s.db.WithContext(ctx).First(&row, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.NotFound("Session", id)
		}
		return nil, err
	}
	return &row, nil
}

func (s *service) ListForUser(ctx context.Context, userID, currentID uint64) ([]Response, error) {
	var rows []UserSession
	if err := s.db.WithContext(ctx).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Order("last_used_at DESC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return ToResponses(rows, currentID), nil
}

func (s *service) Search(ctx context.Context, f Filter, page dto.PageRequest, currentID uint64) (dto.PageResponse[Response], error) {
	var (
		rows  []UserSession
		total int64
	)
	q := s.db.WithContext(ctx).Model(&UserSession{})
	if f.UserID != nil {
		q = q.Where("user_id = ?", *f.UserID)
	}
	if f.Active != nil {
		if *f.Active {
			q = q.Where("revoked_at IS NULL AND expires_at > ?", time.Now().UTC())
		} else {
			q = q.Where("revoked_at IS NOT NULL OR expires_at <= ?", time.Now().UTC())
		}
	}
	if f.Q != "" {
		like := "%" + f.Q + "%"
		q = q.Where("device_name LIKE ? OR user_agent LIKE ? OR ip_address LIKE ?", like, like, like)
	}
	if err := q.Count(&total).Error; err != nil {
		return dto.PageResponse[Response]{}, err
	}
	if err := q.Order("last_used_at DESC").
		Offset(page.Offset()).Limit(page.Limit()).
		Find(&rows).Error; err != nil {
		return dto.PageResponse[Response]{}, err
	}
	return dto.NewPage(ToResponses(rows, currentID), page, total), nil
}

func (s *service) CleanupExpired(ctx context.Context, retention time.Duration) (int64, error) {
	cutoff := time.Now().UTC().Add(-retention)
	res := s.db.WithContext(ctx).
		Where("expires_at <= ? OR (revoked_at IS NOT NULL AND revoked_at <= ?)", cutoff, cutoff).
		Delete(&UserSession{})
	return res.RowsAffected, res.Error
}
