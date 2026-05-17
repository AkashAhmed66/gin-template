package audit

import (
	"github.com/AkashAhmed66/gin-template/internal/common/dto"
	"github.com/AkashAhmed66/gin-template/internal/common/security"
	"github.com/AkashAhmed66/gin-template/internal/common/web"
	"github.com/AkashAhmed66/gin-template/internal/config"
	"github.com/AkashAhmed66/gin-template/internal/modules/permission"
	"github.com/gin-gonic/gin"
)

// Handler exposes the read API. It is only mounted when cfg.ExposeAPI is true.
type Handler struct {
	svc Service
	cfg config.AuditConfig
}

func NewHandler(svc Service, cfg config.AuditConfig) *Handler {
	return &Handler{svc: svc, cfg: cfg}
}

func (h *Handler) Register(r *gin.RouterGroup) {
	if !h.cfg.Enabled || !h.cfg.ExposeAPI {
		return
	}
	r.GET("/audit-logs",
		security.HasPermission(permission.AuditRead),
		web.Handler(h.search))
}

// search godoc
// @Summary      Search audit logs
// @Description  Paginated query over the captured audit_logs table.
// @Tags         audit
// @Produce      json
// @Security     BearerAuth
// @Param        username      query     string  false  "exact username"
// @Param        userId        query     int     false  "exact user id"
// @Param        method        query     string  false  "HTTP method (GET, POST, ...)"
// @Param        path          query     string  false  "substring match on path"
// @Param        action        query     string  false  "exact action label"
// @Param        resourceType  query     string  false  "exact resource type"
// @Param        resourceId    query     string  false  "exact resource id"
// @Param        statusCode    query     int     false  "exact status code"
// @Param        requestId     query     string  false  "exact request id"
// @Param        from          query     string  false  "RFC3339 lower bound (inclusive)"
// @Param        to            query     string  false  "RFC3339 upper bound (inclusive)"
// @Param        page          query     int     false  "page index (0-based)"
// @Param        size          query     int     false  "page size (max 200)"
// @Success      200           {object}  dto.ApiResponse{data=dto.PageResponse[audit.Log]}
// @Failure      403           {object}  dto.ApiResponse
// @Router       /api/v1/audit-logs [get]
func (h *Handler) search(c *gin.Context) (*dto.ApiResponse, error) {
	var f Filter
	_ = c.ShouldBindQuery(&f)
	page := web.BindPage(c)
	res, err := h.svc.Search(c.Request.Context(), f, page)
	if err != nil {
		return nil, err
	}
	return dto.OK(res), nil
}
