// Package web holds Gin-aware middleware and helpers shared by every module.
//
// The two key conventions this package enforces:
//
//   - Handlers return *dto.ApiResponse (via Handler/HandlerErr wrappers) and
//     never write to the response themselves. The writer applies the embedded
//     HTTP status before serialization.
//   - Every request has a stable RequestID exposed both on the response header
//     `X-Request-Id` and in the context, so logs + audit rows share the same id.
package web

const (
	// CtxRequestID is the key under which the request id is stashed.
	CtxRequestID = "request_id"
	// CtxStart is the request-start time, stamped by the logging middleware.
	CtxStart = "request_start"
)
