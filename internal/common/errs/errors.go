// Package errs defines typed application errors and their HTTP mapping. The
// global error-handling middleware in internal/common/web converts these to
// ApiResponse envelopes.
package errs

import (
	"errors"
	"fmt"
	"net/http"
)

// AppError is the base type for typed application errors. Status maps directly
// to the HTTP status returned by the error handler. Use the constructors
// below — avoid building these literally so the status always stays correct.
type AppError struct {
	Status  int
	Code    string      // optional machine-readable code, e.g. "RESOURCE_NOT_FOUND"
	Message string      // human-readable, safe to surface to clients
	Errors  interface{} // optional structured details (validation map, etc.)
	Cause   error       // wrapped underlying error, for logs
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *AppError) Unwrap() error { return e.Cause }

// NotFound — 404. Use for entities the caller tried to read/update/delete
// that don't exist (or are filtered out by soft-delete).
func NotFound(resource string, id any) *AppError {
	return &AppError{
		Status:  http.StatusNotFound,
		Code:    "RESOURCE_NOT_FOUND",
		Message: fmt.Sprintf("%s not found: %v", resource, id),
	}
}

// BadRequest — 400. Use for failed preconditions discovered in service code.
func BadRequest(msg string) *AppError {
	return &AppError{
		Status:  http.StatusBadRequest,
		Code:    "BAD_REQUEST",
		Message: msg,
	}
}

// Validation — 400 with a field-error map. The web layer also produces this
// automatically from validator.v10 errors on bound DTOs.
func Validation(fieldErrors map[string]string) *AppError {
	return &AppError{
		Status:  http.StatusBadRequest,
		Code:    "VALIDATION_FAILED",
		Message: "Validation failed",
		Errors:  fieldErrors,
	}
}

// Unauthorized — 401. Authentication failed or missing.
func Unauthorized(msg string) *AppError {
	if msg == "" {
		msg = "Unauthenticated"
	}
	return &AppError{Status: http.StatusUnauthorized, Code: "UNAUTHENTICATED", Message: msg}
}

// Forbidden — 403. Authenticated but not authorized.
func Forbidden(msg string) *AppError {
	if msg == "" {
		msg = "Access denied"
	}
	return &AppError{Status: http.StatusForbidden, Code: "FORBIDDEN", Message: msg}
}

// Conflict — 409.
func Conflict(msg string) *AppError {
	return &AppError{Status: http.StatusConflict, Code: "CONFLICT", Message: msg}
}

// Duplicate — 409, semantically narrower than Conflict.
func Duplicate(msg string) *AppError {
	return &AppError{Status: http.StatusConflict, Code: "DUPLICATE_RESOURCE", Message: msg}
}

// TooManyRequests — 429.
func TooManyRequests(msg string) *AppError {
	return &AppError{Status: http.StatusTooManyRequests, Code: "RATE_LIMITED", Message: msg}
}

// Internal — 500. Prefer wrapping with WithCause so the cause stays in logs.
func Internal(msg string) *AppError {
	return &AppError{Status: http.StatusInternalServerError, Code: "INTERNAL_ERROR", Message: msg}
}

// WithCause attaches an underlying error for logging without changing the
// caller-visible message. Returns the receiver for fluent chaining.
func (e *AppError) WithCause(err error) *AppError {
	e.Cause = err
	return e
}

// As is a convenience over errors.As, returning the *AppError + ok.
func As(err error) (*AppError, bool) {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae, true
	}
	return nil, false
}
