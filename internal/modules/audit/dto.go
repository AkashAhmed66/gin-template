package audit

import "time"

type Filter struct {
	Username     string     `form:"username"`
	UserID       *uint64    `form:"userId"`
	Method       string     `form:"method"`
	Path         string     `form:"path"`
	Action       string     `form:"action"`
	ResourceType string     `form:"resourceType"`
	ResourceID   string     `form:"resourceId"`
	StatusCode   *int       `form:"statusCode"`
	RequestID    string     `form:"requestId"`
	From         *time.Time `form:"from" time_format:"2006-01-02T15:04:05Z07:00"`
	To           *time.Time `form:"to" time_format:"2006-01-02T15:04:05Z07:00"`
}
