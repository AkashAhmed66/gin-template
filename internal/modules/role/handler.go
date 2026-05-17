package role

import (
	"strconv"

	"github.com/AkashAhmed66/gin-template/internal/common/dto"
	"github.com/AkashAhmed66/gin-template/internal/common/errs"
	"github.com/AkashAhmed66/gin-template/internal/common/security"
	"github.com/AkashAhmed66/gin-template/internal/common/web"
	"github.com/AkashAhmed66/gin-template/internal/modules/permission"
	"github.com/gin-gonic/gin"
)

type Handler struct{ svc Service }

func NewHandler(svc Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/roles")
	g.GET("", security.HasPermission(permission.RoleRead), web.Handler(h.list))
	g.GET("/:id", security.HasPermission(permission.RoleRead), web.Handler(h.get))
	g.POST("", security.HasPermission(permission.RoleWrite), web.Handler(h.create))
	g.PUT("/:id", security.HasPermission(permission.RoleWrite), web.Handler(h.update))
	g.PUT("/:id/permissions", security.HasPermission(permission.RoleWrite), web.Handler(h.assignPermissions))
	g.DELETE("/:id", security.HasPermission(permission.RoleDelete), web.Handler(h.delete))
}

// list godoc
// @Summary      List roles
// @Tags         roles
// @Produce      json
// @Security     BearerAuth
// @Param        q     query     string  false  "free-text query"
// @Param        page  query     int     false  "page index (0-based)"
// @Param        size  query     int     false  "page size (max 200)"
// @Success      200   {object}  dto.ApiResponse{data=dto.PageResponse[role.Response]}
// @Router       /api/v1/roles [get]
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
// @Summary      Get role by id
// @Tags         roles
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "role id"
// @Success      200  {object}  dto.ApiResponse{data=role.Response}
// @Failure      404  {object}  dto.ApiResponse
// @Router       /api/v1/roles/{id} [get]
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
// @Summary      Create role
// @Tags         roles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      Request  true  "role payload"
// @Success      201   {object}  dto.ApiResponse{data=role.Response}
// @Failure      400   {object}  dto.ApiResponse
// @Failure      409   {object}  dto.ApiResponse
// @Router       /api/v1/roles [post]
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
// @Summary      Update role
// @Tags         roles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int      true  "role id"
// @Param        body  body      Request  true  "role payload"
// @Success      200   {object}  dto.ApiResponse{data=role.Response}
// @Failure      400   {object}  dto.ApiResponse
// @Failure      404   {object}  dto.ApiResponse
// @Router       /api/v1/roles/{id} [put]
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

// assignPermissions godoc
// @Summary      Replace role's permissions
// @Tags         roles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int                        true  "role id"
// @Param        body  body      AssignPermissionsRequest   true  "permission names"
// @Success      200   {object}  dto.ApiResponse{data=role.Response}
// @Failure      400   {object}  dto.ApiResponse
// @Failure      404   {object}  dto.ApiResponse
// @Router       /api/v1/roles/{id}/permissions [put]
func (h *Handler) assignPermissions(c *gin.Context) (*dto.ApiResponse, error) {
	id, err := parseID(c)
	if err != nil {
		return nil, err
	}
	var req AssignPermissionsRequest
	if err := web.BindJSON(c, &req); err != nil {
		return nil, err
	}
	res, err := h.svc.AssignPermissions(c.Request.Context(), id, req.Permissions)
	if err != nil {
		return nil, err
	}
	return dto.OK(res), nil
}

// delete godoc
// @Summary      Delete role
// @Tags         roles
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "role id"
// @Success      200  {object}  dto.ApiResponse
// @Failure      404  {object}  dto.ApiResponse
// @Router       /api/v1/roles/{id} [delete]
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
