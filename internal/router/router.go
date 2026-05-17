// Package router builds the gin.Engine and mounts every module.
//
// Middleware order is significant:
//  1. Recovery     — outermost: catches panics anywhere downstream
//  2. RequestID    — every later layer logs with a stable id
//  3. RequestLog   — observability
//  4. CORS         — must run before auth so preflight succeeds without a token
//  5. RateLimit    — keys on auth identity when present, IP otherwise
//  6. Audit        — captures every API request
//  7. JWTAuth      — runs on /api/v1, populates Principal
package router

import (
	"net/http"

	"github.com/AkashAhmed66/gin-template/internal/bootstrap"
	"github.com/AkashAhmed66/gin-template/internal/common/dto"
	"github.com/AkashAhmed66/gin-template/internal/common/idempotency"
	"github.com/AkashAhmed66/gin-template/internal/common/ratelimit"
	"github.com/AkashAhmed66/gin-template/internal/common/security"
	"github.com/AkashAhmed66/gin-template/internal/common/web"
	"github.com/AkashAhmed66/gin-template/internal/config"
	auditmod "github.com/AkashAhmed66/gin-template/internal/modules/audit"
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginswagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

// New builds the wired Gin engine.
func New(cfg *config.Config, log *zap.Logger, deps *bootstrap.Deps) *gin.Engine {
	if cfg.App.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.New()

	r.Use(web.Recovery(log))
	r.Use(web.RequestID())
	r.Use(web.RequestLogging(log))
	r.Use(web.CORS(cfg.CORS))

	r.NoRoute(func(c *gin.Context) {
		web.WriteResponse(c, dto.Error(http.StatusNotFound,
			"Endpoint not found: "+c.Request.URL.Path, nil))
	})
	r.NoMethod(func(c *gin.Context) {
		web.WriteResponse(c, dto.Error(http.StatusMethodNotAllowed,
			"Method not allowed", nil))
	})

	r.GET("/", func(c *gin.Context) {
		web.WriteResponse(c, dto.OK(gin.H{
			"name":    cfg.App.Name,
			"env":     cfg.App.Env,
			"version": "1.0",
			"docs":    "/swagger/index.html",
		}))
	})

	// Swagger UI. Generated spec lives in /docs (run `swag init -g cmd/api/main.go -o docs`).
	// Before the first generation this returns a "doc.json not found" page; the rest of
	// the API works normally either way.
	r.GET("/swagger/*any", ginswagger.WrapHandler(swaggerfiles.Handler))

	r.GET("/health", func(c *gin.Context) {
		web.WriteResponse(c, dto.OK(gin.H{
			"status": "ok",
		}))
	})

	// API surface — everything under /api goes through rate limit + audit.
	api := r.Group("/api")
	api.Use(ratelimit.Middleware(deps.RateLimit))
	api.Use(auditmod.Middleware(deps.Audit, cfg.Audit))
	api.Use(security.JWTAuth(deps.JWT, deps.Sessions))

	v1 := api.Group("/v1")

	deps.FileHandler.Register(v1, security.RequireAuth())
	deps.AuthHandler.Register(v1, security.RequireAuth())

	// Apply idempotency to mutation-heavy modules. Each route may opt in
	// independently by passing the middleware in-line — here we apply it to
	// the /products group as a representative example.
	{
		mutators := v1.Group("")
		mutators.Use(security.RequireAuth())
		mutators.Use(idempotency.Middleware(deps.Idem, cfg.Idem))
		deps.ProductHandler.Register(mutators)
	}

	{
		protected := v1.Group("")
		protected.Use(security.RequireAuth())
		deps.UserHandler.Register(protected)
		deps.RoleHandler.Register(protected)
		deps.PermHandler.Register(protected)
		deps.SessionHandler.Register(protected)
		deps.AuditHandler.Register(protected)
	}

	return r
}
