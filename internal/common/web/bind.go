package web

import (
	"github.com/AkashAhmed66/gin-template/internal/common/dto"
	"github.com/gin-gonic/gin"
)

// BindJSON binds and validates a JSON body. Returns the validator error
// unchanged so the Handler wrapper renders the standard "Validation failed"
// envelope.
func BindJSON(c *gin.Context, dst interface{}) error {
	return c.ShouldBindJSON(dst)
}

// BindQuery binds query parameters into dst.
func BindQuery(c *gin.Context, dst interface{}) error {
	return c.ShouldBindQuery(dst)
}

// BindPage extracts the standard ?page=&size=&sort= query params and
// normalizes them.
func BindPage(c *gin.Context) dto.PageRequest {
	var p dto.PageRequest
	_ = c.ShouldBindQuery(&p)
	p.Normalize()
	return p
}
