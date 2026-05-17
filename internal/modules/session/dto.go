package session

import "time"

type Response struct {
	ID             uint64     `json:"id"`
	UserID         uint64     `json:"userId"`
	ImpersonatorID uint64     `json:"impersonatorId,omitempty"`
	DeviceName     string     `json:"deviceName,omitempty"`
	UserAgent      string     `json:"userAgent,omitempty"`
	IPAddress      string     `json:"ipAddress,omitempty"`
	IssuedAt       time.Time  `json:"issuedAt"`
	LastUsedAt     time.Time  `json:"lastUsedAt"`
	ExpiresAt      time.Time  `json:"expiresAt"`
	RevokedAt      *time.Time `json:"revokedAt,omitempty"`
	RevokedReason  string     `json:"revokedReason,omitempty"`
	Current        bool       `json:"current"`
}

type Filter struct {
	UserID *uint64 `form:"userId"`
	Active *bool   `form:"active"`
	Q      string  `form:"q"`
}

// ToResponse builds the response DTO. currentID lets the response flag the
// row that matches the caller's own session.
func ToResponse(s UserSession, currentID uint64) Response {
	return Response{
		ID:             s.ID,
		UserID:         s.UserID,
		ImpersonatorID: s.ImpersonatorID,
		DeviceName:     s.DeviceName,
		UserAgent:      s.UserAgent,
		IPAddress:      s.IPAddress,
		IssuedAt:       s.IssuedAt,
		LastUsedAt:     s.LastUsedAt,
		ExpiresAt:      s.ExpiresAt,
		RevokedAt:      s.RevokedAt,
		RevokedReason:  s.RevokedReason,
		Current:        s.ID == currentID,
	}
}

func ToResponses(rows []UserSession, currentID uint64) []Response {
	out := make([]Response, len(rows))
	for i, s := range rows {
		out[i] = ToResponse(s, currentID)
	}
	return out
}
