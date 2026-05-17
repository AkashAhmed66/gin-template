package web

import (
	"errors"
	"net/http"
	"strings"

	"github.com/AkashAhmed66/gin-template/internal/common/dto"
	"github.com/AkashAhmed66/gin-template/internal/common/errs"
	"github.com/AkashAhmed66/gin-template/internal/logger"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

// Handler wraps a function that returns (*ApiResponse, error). The wrapper:
//   - serializes the envelope with its embedded status on success
//   - maps known error types (*errs.AppError, validator.ValidationErrors) to
//     proper status codes + structured payloads
//   - falls back to a 500 envelope for unknown errors and logs the cause
//
// Use this everywhere instead of touching gin.Context.JSON directly — it's
// what gives every endpoint a consistent JSON shape.
func Handler(fn func(c *gin.Context) (*dto.ApiResponse, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := fn(c)
		if err != nil {
			writeError(c, err)
			return
		}
		if resp == nil {
			WriteResponse(c, dto.OK(nil))
			return
		}
		WriteResponse(c, resp)
	}
}

func writeError(c *gin.Context, err error) {
	var appErr *errs.AppError
	if errors.As(err, &appErr) {
		if appErr.Cause != nil {
			logger.FromContext(c.Request.Context()).
				Warn("app error", zap.String("code", appErr.Code), zap.Error(appErr.Cause))
		}
		WriteResponse(c, dto.Error(appErr.Status, appErr.Message, appErr.Errors))
		return
	}

	var ves validator.ValidationErrors
	if errors.As(err, &ves) {
		fieldErrors := make(map[string]string, len(ves))
		for _, fe := range ves {
			fieldErrors[lowerFirst(fe.Field())] = humanReadable(fe)
		}
		WriteResponse(c, dto.Error(http.StatusBadRequest, "Validation failed", fieldErrors))
		return
	}

	logger.FromContext(c.Request.Context()).Error("unhandled handler error", zap.Error(err))
	WriteResponse(c, dto.Error(http.StatusInternalServerError, "Internal server error", nil))
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func humanReadable(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a well-formed email address"
	case "min":
		return "must be at least " + fe.Param()
	case "max":
		return "must be at most " + fe.Param()
	case "len":
		return "must have length " + fe.Param()
	case "gte":
		return "must be greater than or equal to " + fe.Param()
	case "lte":
		return "must be less than or equal to " + fe.Param()
	case "oneof":
		return "must be one of: " + fe.Param()
	}
	return "is invalid"
}
