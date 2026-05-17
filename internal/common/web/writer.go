package web

import (
	"github.com/AkashAhmed66/gin-template/internal/common/dto"
	"github.com/gin-gonic/gin"
)

// WriteResponse serializes an ApiResponse with its embedded HTTP status.
// Use this anywhere you need to short-circuit (middleware errors, custom
// handlers that don't go through the Handler wrapper).
func WriteResponse(c *gin.Context, r *dto.ApiResponse) {
	if r == nil {
		return
	}
	status := r.Status
	if status == 0 {
		status = 200
	}
	if status == 204 {
		c.Status(status)
		return
	}
	c.JSON(status, r)
}
