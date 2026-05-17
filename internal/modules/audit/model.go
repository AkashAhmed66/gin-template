// Package audit captures every API call into the audit_logs table for forensic
// review. Each request flows through middleware that buffers the body, hands
// off to the handler, then asynchronously writes one audit row with the
// matched request id, status, latency, masked body bodies, and the action
// label attached via the Action(...) helper.
package audit

import "time"

// Log is the persisted audit entry. One row per API request.
type Log struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	RequestID    string    `gorm:"size:64;index" json:"requestId"`
	Timestamp    time.Time `gorm:"not null;index" json:"timestamp"`
	DurationMs   int64     `json:"durationMs"`
	UserID       uint64    `gorm:"index" json:"userId,omitempty"`
	Username     string    `gorm:"size:100;index" json:"username,omitempty"`
	Method       string    `gorm:"size:10;index" json:"method"`
	Path         string    `gorm:"size:500;index" json:"path"`
	QueryString  string    `gorm:"size:1000" json:"queryString,omitempty"`
	StatusCode   int       `gorm:"index" json:"statusCode"`
	Action       string    `gorm:"size:100;index" json:"action,omitempty"`
	ResourceType string    `gorm:"size:100;index" json:"resourceType,omitempty"`
	ResourceID   string    `gorm:"size:100;index" json:"resourceId,omitempty"`
	ClientIP     string    `gorm:"size:64" json:"clientIp,omitempty"`
	UserAgent    string    `gorm:"size:500" json:"userAgent,omitempty"`
	RequestBody  string    `gorm:"type:text" json:"requestBody,omitempty"`
	ResponseBody string    `gorm:"type:text" json:"responseBody,omitempty"`
	ErrorMessage string    `gorm:"type:text" json:"errorMessage,omitempty"`
}

func (Log) TableName() string { return "audit_logs" }
