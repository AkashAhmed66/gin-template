package auth

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

// Register wires the auth routes. r is expected to be the /api/v1 group.
// Public endpoints (register, login, refresh, forgot/reset) need no auth;
// the rest sit behind RequireAuth at the call site.
func (h *Handler) Register(r *gin.RouterGroup, requireAuth gin.HandlerFunc) {
	pub := r.Group("/auth")
	pub.POST("/register", web.Handler(h.register))
	pub.POST("/login", web.Handler(h.login))
	pub.POST("/refresh", web.Handler(h.refresh))
	pub.POST("/forgot-password", web.Handler(h.forgotPassword))
	pub.POST("/reset-password", web.Handler(h.resetPassword))

	priv := r.Group("/auth")
	priv.Use(requireAuth)
	priv.POST("/logout", web.Handler(h.logout))
	priv.POST("/logout-all", web.Handler(h.logoutAll))
	priv.GET("/sessions", web.Handler(h.mySessions))
	priv.DELETE("/sessions/:id", web.Handler(h.revokeSession))
	priv.POST("/impersonate/:userId",
		security.HasPermission(permission.UserImpersonate),
		web.Handler(h.impersonate))
}

// register godoc
// @Summary      Register a new user
// @Description  Creates an account, assigns the default USER role, and returns a token pair.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      RegisterRequest  true  "registration payload"
// @Success      201   {object}  dto.ApiResponse{data=AuthResponse}
// @Failure      400   {object}  dto.ApiResponse
// @Failure      409   {object}  dto.ApiResponse
// @Router       /api/v1/auth/register [post]
func (h *Handler) register(c *gin.Context) (*dto.ApiResponse, error) {
	var req RegisterRequest
	if err := web.BindJSON(c, &req); err != nil {
		return nil, err
	}
	res, err := h.svc.Register(c.Request.Context(), req, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		return nil, err
	}
	return dto.Created(res), nil
}

// login godoc
// @Summary      Login
// @Description  Exchanges credentials for an access + refresh token pair.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      LoginRequest  true  "login payload"
// @Success      200   {object}  dto.ApiResponse{data=AuthResponse}
// @Failure      400   {object}  dto.ApiResponse
// @Failure      401   {object}  dto.ApiResponse
// @Router       /api/v1/auth/login [post]
func (h *Handler) login(c *gin.Context) (*dto.ApiResponse, error) {
	var req LoginRequest
	if err := web.BindJSON(c, &req); err != nil {
		return nil, err
	}
	res, err := h.svc.Login(c.Request.Context(), req, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		return nil, err
	}
	return dto.OK(res), nil
}

// refresh godoc
// @Summary      Refresh tokens
// @Description  Rotates an access + refresh token pair using a still-valid refresh token.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      RefreshRequest  true  "refresh payload"
// @Success      200   {object}  dto.ApiResponse{data=AuthResponse}
// @Failure      401   {object}  dto.ApiResponse
// @Router       /api/v1/auth/refresh [post]
func (h *Handler) refresh(c *gin.Context) (*dto.ApiResponse, error) {
	var req RefreshRequest
	if err := web.BindJSON(c, &req); err != nil {
		return nil, err
	}
	res, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		return nil, err
	}
	return dto.OK(res), nil
}

// forgotPassword godoc
// @Summary      Request password reset
// @Description  Sends a reset link if the email matches an enabled user. Response is identical for known/unknown emails to avoid enumeration.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      ForgotPasswordRequest  true  "email payload"
// @Success      200   {object}  dto.ApiResponse
// @Router       /api/v1/auth/forgot-password [post]
func (h *Handler) forgotPassword(c *gin.Context) (*dto.ApiResponse, error) {
	var req ForgotPasswordRequest
	if err := web.BindJSON(c, &req); err != nil {
		return nil, err
	}
	if err := h.svc.ForgotPassword(c.Request.Context(), req); err != nil {
		return nil, err
	}
	return dto.Message("If that email exists, a reset link has been sent"), nil
}

// resetPassword godoc
// @Summary      Consume password reset token
// @Description  Sets a new password and revokes every active session for the user.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      ResetPasswordRequest  true  "reset payload"
// @Success      200   {object}  dto.ApiResponse
// @Failure      400   {object}  dto.ApiResponse
// @Router       /api/v1/auth/reset-password [post]
func (h *Handler) resetPassword(c *gin.Context) (*dto.ApiResponse, error) {
	var req ResetPasswordRequest
	if err := web.BindJSON(c, &req); err != nil {
		return nil, err
	}
	if err := h.svc.ResetPassword(c.Request.Context(), req); err != nil {
		return nil, err
	}
	return dto.Message("Password reset"), nil
}

// logout godoc
// @Summary      Logout current device
// @Description  Revokes the session row tied to the caller's current access token.
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  dto.ApiResponse
// @Failure      401  {object}  dto.ApiResponse
// @Router       /api/v1/auth/logout [post]
func (h *Handler) logout(c *gin.Context) (*dto.ApiResponse, error) {
	p, err := security.CurrentPrincipal(c)
	if err != nil {
		return nil, err
	}
	if err := h.svc.Logout(c.Request.Context(), p.SessionID); err != nil {
		return nil, err
	}
	return dto.Message("Logged out"), nil
}

// logoutAll godoc
// @Summary      Logout from all devices
// @Description  Revokes every active session for the caller.
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  dto.ApiResponse
// @Failure      401  {object}  dto.ApiResponse
// @Router       /api/v1/auth/logout-all [post]
func (h *Handler) logoutAll(c *gin.Context) (*dto.ApiResponse, error) {
	p, err := security.CurrentPrincipal(c)
	if err != nil {
		return nil, err
	}
	if err := h.svc.LogoutAll(c.Request.Context(), p.UserID); err != nil {
		return nil, err
	}
	return dto.Message("All sessions revoked"), nil
}

// mySessions godoc
// @Summary      List my active sessions
// @Description  Returns the caller's active sessions. Each row carries deviceName, ipAddress, lastUsedAt, and a `current` flag for the session backing this request.
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  dto.ApiResponse
// @Failure      401  {object}  dto.ApiResponse
// @Router       /api/v1/auth/sessions [get]
func (h *Handler) mySessions(c *gin.Context) (*dto.ApiResponse, error) {
	p, err := security.CurrentPrincipal(c)
	if err != nil {
		return nil, err
	}
	rows, err := h.svc.ListMySessions(c.Request.Context(), p.UserID, p.SessionID)
	if err != nil {
		return nil, err
	}
	return dto.OK(rows), nil
}

// revokeSession godoc
// @Summary      Revoke one of my sessions
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "session id"
// @Success      200  {object}  dto.ApiResponse
// @Failure      401  {object}  dto.ApiResponse
// @Failure      404  {object}  dto.ApiResponse
// @Router       /api/v1/auth/sessions/{id} [delete]
func (h *Handler) revokeSession(c *gin.Context) (*dto.ApiResponse, error) {
	p, err := security.CurrentPrincipal(c)
	if err != nil {
		return nil, err
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return nil, errs.BadRequest("Invalid id")
	}
	if err := h.svc.RevokeMySession(c.Request.Context(), p.UserID, id); err != nil {
		return nil, err
	}
	return dto.Message("Session revoked"), nil
}

// impersonate godoc
// @Summary      Impersonate another user (admin)
// @Description  Issues tokens scoped to the target user. The session row records both ids.
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Param        userId  path      int  true  "user id to impersonate"
// @Success      200     {object}  dto.ApiResponse{data=AuthResponse}
// @Failure      400     {object}  dto.ApiResponse
// @Failure      401     {object}  dto.ApiResponse
// @Failure      403     {object}  dto.ApiResponse
// @Failure      404     {object}  dto.ApiResponse
// @Router       /api/v1/auth/impersonate/{userId} [post]
func (h *Handler) impersonate(c *gin.Context) (*dto.ApiResponse, error) {
	p, err := security.CurrentPrincipal(c)
	if err != nil {
		return nil, err
	}
	target, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil {
		return nil, errs.BadRequest("Invalid userId")
	}
	res, err := h.svc.Impersonate(c.Request.Context(), p.UserID, target, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		return nil, err
	}
	return dto.OK(res), nil
}
