// Package session manages per-device authentication sessions. Every login
// creates one row, embeds its id as the JWT `sid` claim, and the auth
// middleware looks the row up on every request — so revoking a row logs out
// exactly one device.
package session

import (
	"time"
)

// UserSession is the per-device authentication record.
type UserSession struct {
	ID             uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID         uint64     `gorm:"not null;index" json:"userId"`
	ImpersonatorID uint64     `gorm:"default:0;index" json:"impersonatorId,omitempty"`
	DeviceName     string     `gorm:"size:200" json:"deviceName,omitempty"`
	UserAgent      string     `gorm:"size:500" json:"userAgent,omitempty"`
	IPAddress      string     `gorm:"size:64" json:"ipAddress,omitempty"`
	IssuedAt       time.Time  `gorm:"not null;index" json:"issuedAt"`
	LastUsedAt     time.Time  `gorm:"not null" json:"lastUsedAt"`
	ExpiresAt      time.Time  `gorm:"not null;index" json:"expiresAt"`
	RevokedAt      *time.Time `gorm:"index" json:"revokedAt,omitempty"`
	RevokedReason  string     `gorm:"size:64" json:"revokedReason,omitempty"`
}

func (UserSession) TableName() string { return "user_sessions" }

// Active reports whether the session is currently usable.
func (s *UserSession) Active() bool {
	if s.RevokedAt != nil {
		return false
	}
	return time.Now().UTC().Before(s.ExpiresAt)
}
