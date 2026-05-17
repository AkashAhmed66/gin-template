package session

import (
	"strconv"

	"github.com/AkashAhmed66/gin-template/internal/common/dto"
	"github.com/AkashAhmed66/gin-template/internal/common/errs"
	"github.com/AkashAhmed66/gin-template/internal/common/security"
	"github.com/AkashAhmed66/gin-template/internal/common/web"
	"github.com/AkashAhmed66/gin-template/internal/modules/permission"
	"github.com/gin-gonic/gin"
)

// Handler exposes the admin-side session endpoints. The user-facing endpoints
// (list mine, revoke mine) live on the auth controller — see modules/auth.
type Handler struct{ svc Service }

func NewHandler(svc Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/admin/sessions")
	g.GET("", security.HasPermission(permission.SessionRead), web.Handler(h.search))
	g.GET("/:id", security.HasPermission(permission.SessionRead), web.Handler(h.get))
	g.DELETE("/:id", security.HasPermission(permission.SessionRevoke), web.Handler(h.revoke))
}

// search godoc
// @Summary      Search sessions (admin)
// @Description  Paginated cross-user search. Filter by userId, active, or free-text q.
// @Tags         admin-sessions
// @Produce      json
// @Security     BearerAuth
// @Param        userId  query     int     false  "filter by user id"
// @Param        active  query     bool    false  "true=only active, false=only revoked/expired"
// @Param        q       query     string  false  "match against deviceName/userAgent/ipAddress"
// @Param        page    query     int     false  "page index (0-based)"
// @Param        size    query     int     false  "page size (max 200)"
// @Success      200     {object}  dto.ApiResponse{data=dto.PageResponse[session.Response]}
// @Router       /api/v1/admin/sessions [get]
func (h *Handler) search(c *gin.Context) (*dto.ApiResponse, error) {
	var f Filter
	_ = c.ShouldBindQuery(&f)
	page := web.BindPage(c)
	p, _ := security.CurrentPrincipal(c)
	var sid uint64
	if p != nil {
		sid = p.SessionID
	}
	res, err := h.svc.Search(c.Request.Context(), f, page, sid)
	if err != nil {
		return nil, err
	}
	return dto.OK(res), nil
}

// get godoc
// @Summary      Get session by id (admin)
// @Tags         admin-sessions
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "session id"
// @Success      200  {object}  dto.ApiResponse{data=session.Response}
// @Failure      404  {object}  dto.ApiResponse
// @Router       /api/v1/admin/sessions/{id} [get]
func (h *Handler) get(c *gin.Context) (*dto.ApiResponse, error) {
	id, err := parseID(c)
	if err != nil {
		return nil, err
	}
	s, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		return nil, err
	}
	p, _ := security.CurrentPrincipal(c)
	var sid uint64
	if p != nil {
		sid = p.SessionID
	}
	return dto.OK(ToResponse(*s, sid)), nil
}

// revoke godoc
// @Summary      Revoke a session (admin)
// @Tags         admin-sessions
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "session id"
// @Success      200  {object}  dto.ApiResponse
// @Failure      404  {object}  dto.ApiResponse
// @Router       /api/v1/admin/sessions/{id} [delete]
func (h *Handler) revoke(c *gin.Context) (*dto.ApiResponse, error) {
	id, err := parseID(c)
	if err != nil {
		return nil, err
	}
	if err := h.svc.Revoke(c.Request.Context(), id, "admin-revoked"); err != nil {
		return nil, err
	}
	return dto.Message("Session revoked"), nil
}

func parseID(c *gin.Context) (uint64, error) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return 0, errs.BadRequest("Invalid id")
	}
	return id, nil
}
