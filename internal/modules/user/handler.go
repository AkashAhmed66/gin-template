package user

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
	g := r.Group("/users")
	g.GET("/me", web.Handler(h.me))
	g.GET("", security.HasPermission(permission.UserRead), web.Handler(h.list))
	g.GET("/:id", security.HasPermission(permission.UserRead), web.Handler(h.get))
	g.PUT("/:id", security.HasPermission(permission.UserWrite), web.Handler(h.update))
	g.POST("/:id/activate", security.HasPermission(permission.UserWrite), web.Handler(h.activate))
	g.POST("/:id/deactivate", security.HasPermission(permission.UserWrite), web.Handler(h.deactivate))
	g.POST("/:id/force-logout", security.HasPermission(permission.UserWrite), web.Handler(h.forceLogout))
	g.PUT("/:id/roles", security.HasPermission(permission.UserWrite), web.Handler(h.assignRoles))
	g.DELETE("/:id", security.HasPermission(permission.UserDelete), web.Handler(h.delete))
}

// me godoc
// @Summary      Get current user
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  dto.ApiResponse{data=user.Response}
// @Failure      401  {object}  dto.ApiResponse
// @Router       /api/v1/users/me [get]
func (h *Handler) me(c *gin.Context) (*dto.ApiResponse, error) {
	p, err := security.CurrentPrincipal(c)
	if err != nil {
		return nil, err
	}
	res, err := h.svc.GetByID(c.Request.Context(), p.UserID)
	if err != nil {
		return nil, err
	}
	return dto.OK(res), nil
}

// list godoc
// @Summary      List users
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Param        q        query     string  false  "free-text query"
// @Param        role     query     string  false  "filter by role name"
// @Param        enabled  query     bool    false  "filter by enabled flag"
// @Param        page     query     int     false  "page index (0-based)"
// @Param        size     query     int     false  "page size (max 200)"
// @Success      200      {object}  dto.ApiResponse{data=dto.PageResponse[user.Response]}
// @Failure      401      {object}  dto.ApiResponse
// @Failure      403      {object}  dto.ApiResponse
// @Router       /api/v1/users [get]
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
// @Summary      Get user by id
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "user id"
// @Success      200  {object}  dto.ApiResponse{data=user.Response}
// @Failure      404  {object}  dto.ApiResponse
// @Router       /api/v1/users/{id} [get]
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

// update godoc
// @Summary      Update user
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int            true  "user id"
// @Param        body  body      UpdateRequest  true  "update payload"
// @Success      200   {object}  dto.ApiResponse{data=user.Response}
// @Failure      400   {object}  dto.ApiResponse
// @Failure      404   {object}  dto.ApiResponse
// @Failure      409   {object}  dto.ApiResponse
// @Router       /api/v1/users/{id} [put]
func (h *Handler) update(c *gin.Context) (*dto.ApiResponse, error) {
	id, err := parseID(c)
	if err != nil {
		return nil, err
	}
	var req UpdateRequest
	if err := web.BindJSON(c, &req); err != nil {
		return nil, err
	}
	res, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		return nil, err
	}
	return dto.OK(res), nil
}

// activate godoc
// @Summary      Activate user
// @Description  Sets enabled=true. Pre-existing revoked sessions remain revoked.
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "user id"
// @Success      200  {object}  dto.ApiResponse{data=user.Response}
// @Failure      404  {object}  dto.ApiResponse
// @Router       /api/v1/users/{id}/activate [post]
func (h *Handler) activate(c *gin.Context) (*dto.ApiResponse, error) {
	id, err := parseID(c)
	if err != nil {
		return nil, err
	}
	res, err := h.svc.Activate(c.Request.Context(), id)
	if err != nil {
		return nil, err
	}
	return dto.OK(res), nil
}

// deactivate godoc
// @Summary      Deactivate user
// @Description  Sets enabled=false and revokes every active session for the user. Refuses to act on the caller's own account.
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "user id"
// @Success      200  {object}  dto.ApiResponse{data=user.Response}
// @Failure      400  {object}  dto.ApiResponse
// @Failure      404  {object}  dto.ApiResponse
// @Router       /api/v1/users/{id}/deactivate [post]
func (h *Handler) deactivate(c *gin.Context) (*dto.ApiResponse, error) {
	id, err := parseID(c)
	if err != nil {
		return nil, err
	}
	p, err := security.CurrentPrincipal(c)
	if err != nil {
		return nil, err
	}
	res, err := h.svc.Deactivate(c.Request.Context(), id, p.UserID)
	if err != nil {
		return nil, err
	}
	return dto.OK(res), nil
}

// forceLogout godoc
// @Summary      Force-logout user
// @Description  Revokes every session without changing enabled. Useful after a suspected token leak.
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "user id"
// @Success      200  {object}  dto.ApiResponse
// @Failure      404  {object}  dto.ApiResponse
// @Router       /api/v1/users/{id}/force-logout [post]
func (h *Handler) forceLogout(c *gin.Context) (*dto.ApiResponse, error) {
	id, err := parseID(c)
	if err != nil {
		return nil, err
	}
	if err := h.svc.ForceLogout(c.Request.Context(), id); err != nil {
		return nil, err
	}
	return dto.Message("User sessions revoked"), nil
}

// assignRoles godoc
// @Summary      Replace user's roles
// @Description  Replaces the user's role set with the given names. Triggers a session revoke.
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int                 true  "user id"
// @Param        body  body      AssignRolesRequest  true  "role names payload"
// @Success      200   {object}  dto.ApiResponse{data=user.Response}
// @Failure      400   {object}  dto.ApiResponse
// @Failure      404   {object}  dto.ApiResponse
// @Router       /api/v1/users/{id}/roles [put]
func (h *Handler) assignRoles(c *gin.Context) (*dto.ApiResponse, error) {
	id, err := parseID(c)
	if err != nil {
		return nil, err
	}
	var req AssignRolesRequest
	if err := web.BindJSON(c, &req); err != nil {
		return nil, err
	}
	res, err := h.svc.AssignRoles(c.Request.Context(), id, req.Roles)
	if err != nil {
		return nil, err
	}
	return dto.OK(res), nil
}

// delete godoc
// @Summary      Delete user
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "user id"
// @Success      200  {object}  dto.ApiResponse
// @Failure      404  {object}  dto.ApiResponse
// @Router       /api/v1/users/{id} [delete]
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
