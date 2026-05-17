// Package dto contains the shared response envelopes used by every controller.
//
// Controllers return *ApiResponse — the middleware in internal/common/web
// serializes it with the embedded HTTP status, so handlers don't write to
// gin.Context directly. This mirrors spring's ApiResponseBodyAdvice pattern.
package dto

import (
	"net/http"
	"time"
)

// ApiResponse is the canonical envelope. Marshalled with the JSON shape:
//
//	{
//	  "success":   true,
//	  "message":   "OK",
//	  "data":      { ... } | null,
//	  "errors":    null | { ... },
//	  "timestamp": "2026-04-29T12:34:56Z"
//	}
//
// Status is omitted from JSON; the writer applies it to the HTTP response.
type ApiResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Errors    interface{} `json:"errors,omitempty"`
	Timestamp time.Time   `json:"timestamp"`

	Status int `json:"-"`
}

// OK builds a 200 response with the supplied data. Pass nil to omit the data field.
func OK(data interface{}) *ApiResponse {
	return &ApiResponse{
		Success:   true,
		Message:   "OK",
		Data:      unwrap(data),
		Timestamp: time.Now().UTC(),
		Status:    http.StatusOK,
	}
}

// OKWithMessage is OK with a custom message.
func OKWithMessage(data interface{}, message string) *ApiResponse {
	r := OK(data)
	r.Message = message
	return r
}

// Created builds a 201 response.
func Created(data interface{}) *ApiResponse {
	r := OK(data)
	r.Message = "Created"
	r.Status = http.StatusCreated
	return r
}

// CreatedWithMessage is Created with a custom message.
func CreatedWithMessage(data interface{}, message string) *ApiResponse {
	r := Created(data)
	r.Message = message
	return r
}

// NoContent builds a 204 response — no data, no message body.
func NoContent() *ApiResponse {
	r := OK(nil)
	r.Status = http.StatusNoContent
	r.Message = ""
	return r
}

// Message builds a 200 response carrying only a message (no data).
func Message(message string) *ApiResponse {
	r := OK(nil)
	r.Message = message
	return r
}

// Error builds an error response with the given message, optional structured
// errors payload (e.g. field validation map), and HTTP status.
func Error(status int, message string, errs interface{}) *ApiResponse {
	if status == 0 {
		status = http.StatusBadRequest
	}
	return &ApiResponse{
		Success:   false,
		Message:   message,
		Errors:    errs,
		Timestamp: time.Now().UTC(),
		Status:    status,
	}
}

// unwrap converts a *PageResponse-shaped value or its source to the
// page envelope shape. Currently a passthrough; reserved for future
// auto-wrapping (e.g. detecting `Page[T]` types).
func unwrap(data interface{}) interface{} {
	return data
}
