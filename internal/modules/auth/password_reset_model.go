package auth

import "time"

// PasswordResetToken is single-use. ConsumedAt is set when the token has been
// used so a replay is rejected.
type PasswordResetToken struct {
	ID         uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     uint64     `gorm:"not null;index" json:"userId"`
	TokenHash  string     `gorm:"size:128;not null;uniqueIndex" json:"-"`
	ExpiresAt  time.Time  `gorm:"not null;index" json:"expiresAt"`
	ConsumedAt *time.Time `json:"consumedAt,omitempty"`
	CreatedAt  time.Time  `gorm:"not null" json:"createdAt"`
}

func (PasswordResetToken) TableName() string { return "password_reset_tokens" }
