package permission

import (
	"strconv"

	"github.com/AkashAhmed66/gin-template/internal/common/dto"
	"github.com/AkashAhmed66/gin-template/internal/common/errs"
	"github.com/AkashAhmed66/gin-template/internal/common/security"
	"github.com/AkashAhmed66/gin-template/internal/common/web"
	"github.com/gin-gonic/gin"
)

// Handler is the HTTP layer for the permission module.
type Handler struct{ svc Service }

func NewHandler(svc Service) *Handler { return &Handler{svc: svc} }

// Register wires the permission routes into r (assumed to be /api/v1).
func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/permissions")
	g.GET("", security.HasPermission(PermissionRead), web.Handler(h.list))
	g.GET("/:id", security.HasPermission(PermissionRead), web.Handler(h.get))
	g.POST("", security.HasPermission(PermissionWrite), web.Handler(h.create))
	g.PUT("/:id", security.HasPermission(PermissionWrite), web.Handler(h.update))
	g.DELETE("/:id", security.HasPermission(PermissionDelete), web.Handler(h.delete))
}

// list godoc
// @Summary      List permissions
// @Tags         permissions
// @Produce      json
// @Security     BearerAuth
// @Param        q     query     string  false  "free-text query"
// @Param        page  query     int     false  "page index (0-based)"
// @Param        size  query     int     false  "page size (max 200)"
// @Success      200   {object}  dto.ApiResponse{data=dto.PageResponse[permission.Response]}
// @Router       /api/v1/permissions [get]
func (h *Handler) list(c *gin.Context) (*dto.ApiResponse, error) {
	var f Filter
	_ = c.ShouldBindQuery(&f)
	page := web.BindPage(c)
	res, err := h.svc.Search(c.Request.Context(), f, page)
	if err != nil {
		return nil, err
	}
	return dto.OK(res), nil
}

// get godoc
// @Summary      Get permission by id
// @Tags         permissions
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "permission id"
// @Success      200  {object}  dto.ApiResponse{data=permission.Response}
// @Failure      404  {object}  dto.ApiResponse
// @Router       /api/v1/permissions/{id} [get]
func (h *Handler) get(c *gin.Context) (*dto.ApiResponse, error) {
	id, err := parseID(c)
	if err != nil {
		return nil, err
	}
	res, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		return nil, err
	}
	return dto.OK(res), nil
}

// create godoc
// @Summary      Create permission
// @Tags         permissions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      Request  true  "permission payload"
// @Success      201   {object}  dto.ApiResponse{data=permission.Response}
// @Failure      400   {object}  dto.ApiResponse
// @Failure      409   {object}  dto.ApiResponse
// @Router       /api/v1/permissions [post]
func (h *Handler) create(c *gin.Context) (*dto.ApiResponse, error) {
	var req Request
	if err := web.BindJSON(c, &req); err != nil {
		return nil, err
	}
	res, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		return nil, err
	}
	return dto.Created(res), nil
}

// update godoc
// @Summary      Update permission
// @Tags         permissions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int      true  "permission id"
// @Param        body  body      Request  true  "permission payload"
// @Success      200   {object}  dto.ApiResponse{data=permission.Response}
// @Failure      400   {object}  dto.ApiResponse
// @Failure      404   {object}  dto.ApiResponse
// @Router       /api/v1/permissions/{id} [put]
func (h *Handler) update(c *gin.Context) (*dto.ApiResponse, error) {
	id, err := parseID(c)
	if err != nil {
		return nil, err
	}
	var req Request
	if err := web.BindJSON(c, &req); err != nil {
		return nil, err
	}
	res, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		return nil, err
	}
	return dto.OK(res), nil
}

// delete godoc
// @Summary      Delete permission
// @Tags         permissions
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "permission id"
// @Success      200  {object}  dto.ApiResponse
// @Failure      404  {object}  dto.ApiResponse
// @Router       /api/v1/permissions/{id} [delete]
func (h *Handler) delete(c *gin.Context) (*dto.ApiResponse, error) {
	id, err := parseID(c)
	if err != nil {
		return nil, err
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		return nil, err
	}
	return dto.Message("Deleted"), nil
}

func parseID(c *gin.Context) (uint64, error) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return 0, errs.BadRequest("Invalid id")
	}
	return id, nil
}
