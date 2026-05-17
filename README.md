# Gin API Template

A reusable Gin (Go) starter for building secure REST APIs. Pre-wired with JWT authentication, RBAC (roles + permissions), **per-device sessions with selective revocation**, **admin impersonation**, **user activate/deactivate**, GORM with audit + soft delete, global error handling, file uploads, structured (zap) logging, OpenAPI annotations, goose migrations, **DB-backed audit logging**, **per-identity rate limiting**, **idempotency**, and **`.env`-driven configuration**.

This is the Gin port of [spring-boot-template](../spring-boot-template) — same feature set, idiomatic Go.

---

## Table of Contents

1. [Tech Stack](#tech-stack)
2. [Quick Start](#quick-start)
3. [Default Credentials](#default-credentials)
4. [Project Structure](#project-structure)
5. [Adding a New API Module](#adding-a-new-api-module)
6. [Cross-Cutting Concerns](#cross-cutting-concerns)
   - [Logging — zap + rotation + per-domain files](#logging--zap--rotation--per-domain-files)
   - [Background Scheduler — `jobs.Every` / `jobs.Cron` / `jobs.Once`](#background-scheduler--jobsevery--jobscron--jobsonce)
   - [In-process Worker Pool — `jobs.Submit`](#in-process-worker-pool--jobssubmit)
   - [Persistent Task Queue — `queue.Enqueue` (Redis-backed, asynq)](#persistent-task-queue--queueenqueue-redis-backed-asynq)
7. [Configuration Reference](#configuration-reference)
8. [Built-in Endpoints](#built-in-endpoints)
9. [Database Migrations](#database-migrations)
10. [Build, Test, Run](#build-test-run)

---

## Tech Stack

| Concern        | Choice |
|----------------|--------|
| Runtime        | Go 1.24+ |
| Web framework  | [Gin](https://github.com/gin-gonic/gin) |
| ORM            | [GORM](https://gorm.io) — Postgres + MySQL drivers ship |
| Migrations     | [goose](https://github.com/pressly/goose) |
| Security       | [golang-jwt/jwt v5](https://github.com/golang-jwt/jwt) (HS256), bcrypt |
| Validation     | [go-playground/validator v10](https://github.com/go-playground/validator) (built into Gin) |
| Logging        | [zap](https://github.com/uber-go/zap) + [lumberjack](https://github.com/natefinch/lumberjack) — console (dev) / JSON (prod), file rotation, per-domain log files |
| Scheduler      | In-process intervals + [robfig/cron/v3](https://github.com/robfig/cron) cron specs, with panic recovery |
| Worker pool    | In-process bounded goroutine pool for fire-and-forget async work |
| Persistent queue | [hibiken/asynq](https://github.com/hibiken/asynq) — Redis-backed, retries, delayed tasks, priority queues, dashboard |
| Config         | [joho/godotenv](https://github.com/joho/godotenv) + typed struct |
| Mail           | stdlib `net/smtp` + `html/template` |
| Docs           | [swaggo/swag](https://github.com/swaggo/swag) — annotation-driven Swagger UI |
| Rate limiting  | Bucket4j-style token bucket, in-memory |
| Idempotency    | DB-backed; replays cached response on retry |
| Hot reload     | [air](https://github.com/air-verse/air) |

---

## Quick Start

### 1. Prerequisites
- Go 1.24+
- Postgres 13+ (or MySQL 8 — flip `DB_DRIVER` in `.env`)
- (optional) [air](https://github.com/air-verse/air) for hot reload, [goose](https://github.com/pressly/goose) for manual migrations

### 2. Configure
```bash
cp .env.example .env
# edit .env if your DB creds differ
```
`.env` is gitignored. `.env.example` is committed and documents every supported variable.

### 3. Install + run

The `docs/` folder is gitignored — it's regenerated locally. On a **fresh clone you MUST generate it before the first `go run`**, otherwise the build fails on the blank import in `main.go`.

```bash
# One-time: install the swag CLI (writes to $GOPATH/bin — make sure that's on PATH)
go install github.com/swaggo/swag/cmd/swag@latest

# Generate the Swagger spec into docs/ — required before the first run
swag init --parseDependency --parseInternal

# Sync Go dependencies
go mod tidy

# Run
go run .
```

Boot order: `.env` loads → DB connects → goose applies migrations → bootstrap admin is created if missing → HTTP server listens on `:8080`.

**On subsequent runs** you only need `swag init --parseDependency --parseInternal` again if you've changed `// @...` annotations on a handler. Plain `go run .` is enough otherwise.

### 4. Try it
- Health: <http://localhost:8080/health>
- Login (default admin):
  ```bash
  curl -X POST http://localhost:8080/api/v1/auth/login \
    -H 'Content-Type: application/json' \
    -d '{"usernameOrEmail":"admin","password":"admin123"}'
  ```
- Use `Authorization: Bearer <accessToken>` on protected endpoints.

### 5. (optional) Hot reload
```bash
go install github.com/air-verse/air@latest
air
```

### 6. Swagger / OpenAPI

The generated spec lives at [docs/](docs/) and is served at `/swagger/*` by the running app.

**Install the CLI once** (writes to `$GOPATH/bin`, make sure that's on `PATH`):
```bash
make swag-install
# or directly:
go install github.com/swaggo/swag/cmd/swag@latest
swag --version
```

**Generate the spec** — bare `swag init` works because `main.go` is at the project root:
```bash
swag init --parseDependency --parseInternal
# or via the Makefile:
make swag
# or via go generate (uses the directive in main.go):
go generate ./...
```

Re-run whenever you add or change `// @...` annotations on a handler, then restart the app — the spec is generated at build time, not at runtime.

**Run the app and open**:
- **Swagger UI** → http://localhost:8080/swagger/index.html
- Spec JSON → http://localhost:8080/swagger/doc.json

To call protected endpoints from the UI: click **Authorize** (top right), paste only the `accessToken` value — the `Bearer ` prefix is added automatically.

See [Swagger / OpenAPI Annotations](#swagger--openapi-annotations) below for the annotation format on individual handlers.

---

## Common commands cheatsheet

Everything you'll do day-to-day. Pick the form that matches your shell (`make` works in Git Bash on Windows or any Unix shell; the bare commands work in PowerShell).

### Setup (fresh clone)

```powershell
# 1. Sync Go dependencies
go mod tidy

# 2. Install the swag CLI (REQUIRED — docs/ is gitignored)
make swag-install
# or: go install github.com/swaggo/swag/cmd/swag@latest

# 3. Generate docs/ before the first build, or main.go fails on the blank import
swag init --parseDependency --parseInternal
# or: make swag

# 4. Optional tooling
go install github.com/pressly/goose/v3/cmd/goose@latest   # manual migrations
go install github.com/air-verse/air@latest                # hot reload
```

### Run

```powershell
# Plain run (migrations auto-apply, admin auto-seeds, server on :8080)
go run .
# or: make run

# Hot reload (rebuilds + restarts on .go save; uses .air.toml)
air
# or: make dev
```

### Migrations (goose)

Migrations run automatically on boot (`MIGRATIONS_AUTO_RUN=true`). To manage them by hand:

```powershell
# PowerShell — Postgres example
$env:DB_DRIVER = "postgres"
$env:DB_DSN    = "postgres://postgres:postgres@localhost:5432/gin_template?sslmode=disable"

goose -dir migrations $env:DB_DRIVER "$env:DB_DSN" up         # apply all pending
goose -dir migrations $env:DB_DRIVER "$env:DB_DSN" down       # roll back the latest
goose -dir migrations $env:DB_DRIVER "$env:DB_DSN" status     # show what's applied

goose -dir migrations create add_orders sql                   # create a new file
```

The Makefile wraps these as `make migrate-up`, `make migrate-down`, `make migrate-status`, `make migrate-create name=...`.

### Swagger

| What | Command |
|---|---|
| Install CLI | `make swag-install` |
| Regenerate spec | `make swag` (or `go generate ./...` or `swag init --parseDependency --parseInternal`) |
| View UI | http://localhost:8080/swagger/index.html (after the app is running) |

Whenever you add or change `// @...` annotations on a handler, re-run `make swag` and restart the app — the spec is generated at build time, not at runtime.

### Build

```powershell
# Local OS
go build -o bin\gin-template.exe .
# or: make build

# Cross-compile a static Linux binary from Windows
$env:GOOS="linux"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"
go build -ldflags="-s -w" -o bin\gin-template .
Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED
```

### Tests & code quality

```powershell
go test -race ./...      # or: make test
go vet ./...             # or: make vet
go fmt ./...             # or: make fmt
```

---

## Default Credentials

| Field    | Value             |
|----------|-------------------|
| Username | `admin`           |
| Email    | `admin@gmail.com` |
| Password | `admin123`        |
| Role     | `ADMIN` (every permission incl. `AUDIT_READ`, `USER_IMPERSONATE`, `SESSION_*`, and the four `ADMIN_*` overrides) |

**Change these before any non-local deployment.** Override via `.env`:
```
BOOTSTRAP_ADMIN_USERNAME=admin
BOOTSTRAP_ADMIN_EMAIL=admin@yourdomain.com
BOOTSTRAP_ADMIN_PASSWORD=strong-pass
```

---

## Project Structure

```
gin-template/
├── main.go                            ← entry point (composition)
├── docs/                              ← generated Swagger spec — gitignored, recreated by `swag init`
├── internal/
│   ├── config/                        ← typed env loading (godotenv + struct)
│   ├── logger/                        ← zap setup + context-scoped fields
│   ├── database/                      ← GORM connection + goose runner
│   ├── bootstrap/                     ← wire deps, register callbacks, seed admin
│   ├── router/                        ← gin.Engine + middleware order
│   ├── common/                        ← shared infrastructure (depend on, don't duplicate)
│   │   ├── audit/                     BaseModel · GORM created_by/updated_by callbacks
│   │   ├── dto/                       ApiResponse · PageResponse · BaseResponse
│   │   ├── errs/                      Typed AppError → HTTP status mapping
│   │   ├── web/                       request_id · logging · recovery · cors · handler wrapper
│   │   ├── security/                  JwtService · JWTAuth · HasRole/HasPermission · password
│   │   ├── ratelimit/                 Token bucket + middleware
│   │   ├── idempotency/               Replay-safe store + middleware
│   │   ├── mail/                      SMTP service + HTML template renderer
│   │   ├── applog/                    Per-domain rotating log files with actor enrichment
│   │   ├── scheduler/                 Recurring jobs (Every / Cron / Once) with panic recovery
│   │   ├── worker/                    In-process bounded goroutine pool
│   │   ├── jobs/                      Global accessor for scheduler + worker pool
│   │   └── queue/                     Persistent task queue (asynq + Redis)
│   └── modules/                       ← every feature lives here, identical shape
│       ├── permission/                names · model · dto · repo · service · handler
│       ├── role/                      same shape, owns role-permission M:N
│       ├── user/                      same shape + activate/deactivate/force-logout/assign-roles
│       ├── session/                   per-device sessions + cleanup job + admin search
│       ├── auth/                      register · login · refresh · logout · sessions · impersonate · password reset
│       ├── audit/                     model · service (buffered async writer) · middleware · admin read API
│       ├── file/                      multipart upload + public serve
│       └── product/                   ← reference CRUD module (mirror this when adding new modules)
├── migrations/                        ← goose SQL migrations
├── templates/email/                   ← HTML email templates
├── .env.example                       ← every supported variable, documented
├── .air.toml                          ← hot-reload config
├── Makefile                           ← common dev commands
├── go.mod
└── README.md
```

---

## Adding a New API Module

The shape is identical for every module. The reference module is [internal/modules/product](internal/modules/product) — copy and rename.

### Recipe — adding `Order`

#### 1. Scaffold
```
internal/modules/order/
├── model.go            (Order entity, embeds audit.BaseModel)
├── dto.go              (Request, Response, Filter, ToResponse, ToResponses)
├── repository.go       (interface + gormRepo)
├── service.go          (interface + service)
└── handler.go          (Handler + Register, route gating with HasPermission)
```

#### 2. Entity
```go
type Order struct {
    audit.BaseModel
    Reference string  `gorm:"size:100;not null;uniqueIndex" json:"reference"`
    Total     float64 `gorm:"not null" json:"total"`
}
func (Order) TableName() string { return "orders" }
```

#### 3. Permission constants — add to [internal/modules/permission/names.go](internal/modules/permission/names.go):
```go
const (
    OrderRead   = "ORDER_READ"
    OrderWrite  = "ORDER_WRITE"
    OrderDelete = "ORDER_DELETE"
)
// add to permission.All so bootstrap admin picks them up
```

#### 4. Handler routes gated by permission
```go
g := r.Group("/orders")
g.GET("", security.HasPermission(permission.OrderRead), web.Handler(h.list))
g.POST("", security.HasPermission(permission.OrderWrite), web.Handler(h.create))
// ...
```

#### 5. Wire it in [internal/bootstrap/bootstrap.go](internal/bootstrap/bootstrap.go)
Add the repo, service, handler to `Deps` and the corresponding `New*` calls.

#### 6. Register routes in [internal/router/router.go](internal/router/router.go)
```go
deps.OrderHandler.Register(protected)
```

#### 7. Migration
```bash
goose -dir migrations create add_orders sql
```

Restart, log in as ADMIN, hit `/api/v1/orders`. Adding a new field later is a 3-edit job: entity + DTO + migration.

---

## Cross-Cutting Concerns

### Authentication (JWT)
- Bearer access tokens validated by [internal/common/security/jwt_middleware.go](internal/common/security/jwt_middleware.go).
- Claims carry `uid`, `sid` (session id), `imp` (impersonator id if any), and the union of `ROLE_<name>` + permission names as `authorities`.
- Issued by `POST /api/v1/auth/login`. Refresh via `POST /api/v1/auth/refresh`. TTLs: env-driven (default 60 min access / 14 day refresh).
- **Stateful revocation** — every login inserts a row in `user_sessions` with its id baked into the JWT as `sid`. The auth middleware validates the session is still active on every request, so revoking the row instantly invalidates every token that carried `sid`.

### Authorization — `HasPermission` / `HasRole`
Middleware factories enforcing per-route gates. No SpEL, no annotations to parse — Gin-native.

```go
g.POST("", security.HasPermission(permission.ProductWrite), web.Handler(h.create))
g.DELETE("/:id", security.HasRole(role.NameAdmin), web.Handler(h.delete))
```

Returns 401 if no principal, 403 if principal lacks the authority.

### Admin Override Permissions
Four coarse permissions in [names.go](internal/modules/permission/names.go) (`ADMIN_READ` / `ADMIN_WRITE` / `ADMIN_EDIT` / `ADMIN_DELETE`) let an admin bypass per-record ownership checks at the service layer:

```go
if security.MustPrincipal(c).HasAnyAuthority(permission.AdminAny...) {
    return repo.AllForAnyone(ctx)
}
return repo.OnlyMine(ctx, currentUserID)
```

### Sessions & Multi-Device Logout
Every login creates a row in `user_sessions`. Revoke endpoints:

| Method   | Path                            | Effect |
|----------|---------------------------------|--------|
| `POST`   | `/api/v1/auth/logout`           | Revoke this device |
| `POST`   | `/api/v1/auth/logout-all`       | Revoke every active session for me |
| `GET`    | `/api/v1/auth/sessions`         | List my active sessions |
| `DELETE` | `/api/v1/auth/sessions/{id}`    | Revoke one session (must be mine, unless I have an `ADMIN_*` permission) |
| `GET`    | `/api/v1/admin/sessions`        | (admin) cross-user search — gated by `SESSION_READ` |
| `DELETE` | `/api/v1/admin/sessions/{id}`   | (admin) revoke any session — gated by `SESSION_REVOKE` |

The session cleanup job (interval + retention configurable) purges expired/revoked rows after the retention window.

### User Lifecycle
- `POST /api/v1/users/{id}/activate` (`USER_WRITE`)
- `POST /api/v1/users/{id}/deactivate` (`USER_WRITE`) — also revokes every session
- `POST /api/v1/users/{id}/force-logout` (`USER_WRITE`) — revoke without touching `enabled`
- `PUT  /api/v1/users/{id}/roles` (`USER_WRITE`) — replace role set; revokes sessions for safety

Login + refresh both reject `enabled=false` accounts with **"Account is deactivated"**.

### Admin Impersonation
`POST /api/v1/auth/impersonate/{userId}` (`USER_IMPERSONATE`) issues tokens **scoped to the target user** but with the admin's id stored on the session row as `impersonator_id` and in the JWT as `imp`. Audit trail attributes actions correctly.

### Current User Context
```go
p, err := security.CurrentPrincipal(c)   // *Principal or error
p := security.MustPrincipal(c)           // aborts with 401 if absent
p, ok := security.FromContext(ctx)       // for service-layer code with context.Context only

p.UserID
p.Username
p.SessionID
p.ImpersonatorID
p.HasAuthority("PRODUCT_READ")
p.HasAnyAuthority(permission.AdminAny...)
p.HasRole("ADMIN")
```

### ApiResponse Envelope
Handlers return `*dto.ApiResponse` from a `web.Handler`-wrapped function — never `c.JSON` directly:

```go
g.POST("", web.Handler(func(c *gin.Context) (*dto.ApiResponse, error) {
    var req CreateOrderRequest
    if err := web.BindJSON(c, &req); err != nil { return nil, err }
    res, err := svc.Create(c.Request.Context(), req)
    if err != nil { return nil, err }
    return dto.Created(res), nil
}))
```

Standard JSON shape:
```json
{
  "success": true,
  "message": "OK",
  "data": { ... },
  "errors": null,
  "timestamp": "2026-04-29T..."
}
```

Constructors:
- `dto.OK(data)` — 200
- `dto.OKWithMessage(data, msg)` — 200 + message
- `dto.Created(data)` — 201
- `dto.Message("Deleted")` — 200, no data
- `dto.NoContent()` — 204
- `dto.Error(status, msg, errors)` — error envelope

Or throw a typed `*errs.AppError` from service code and the wrapper maps it to the right status + message:
- `errs.NotFound("Order", id)` → 404
- `errs.BadRequest("...")` → 400
- `errs.Validation(map[string]string{"email": "..."})` → 400
- `errs.Unauthorized("...")` → 401
- `errs.Forbidden("...")` → 403
- `errs.Conflict("...")` / `errs.Duplicate("...")` → 409
- `errs.TooManyRequests("...")` → 429
- `errs.Internal("...")` → 500

### Pagination — automatic
```go
page := web.BindPage(c)   // reads ?page=&size=&sort=, clamps to [0, 200]
return dto.NewPage(rows, page, total), nil
```
Returns:
```json
"data": {
  "content": [...],
  "page": 0, "size": 20,
  "totalElements": 137, "totalPages": 7,
  "first": true, "last": false, "empty": false
}
```

### Auditing — automatic
Every entity embedding `audit.BaseModel` gets `created_at`, `updated_at`, `created_by`, `updated_by`, `deleted_at`, `deleted_by`, `version`. The user fields are filled by GORM callbacks registered in [internal/common/audit/callbacks.go](internal/common/audit/callbacks.go) — pulled from the request-scoped username on `context.Context` (set by the JWT middleware).

Every response DTO embedding `dto.BaseResponse` exposes the audit/lifecycle fields without redeclaration.

### Soft delete — automatic
`audit.BaseModel.DeletedAt` is a `gorm.DeletedAt`, so GORM appends `deleted_at IS NULL` to every query. To delete, just call `repo.Delete(ctx, id)`; to restore, `entity.Restore()` + save.

### Audit Logging (DB-backed)
Every API call under `/api/` is captured into `audit_logs` asynchronously (1k buffer, single writer goroutine):

- `request_id`, `timestamp`, `duration_ms`
- `user_id`, `username`
- `method`, `path`, `query_string`, `status_code`
- `action`, `resource_type`, `resource_id` — from `audit.Action(c, ...)`
- `client_ip`, `user_agent`
- `request_body`, `response_body` — masked (`password`, `token`, `secret`, etc. → `***`) + truncated at `AUDIT_MAX_BODY_LENGTH`
- `error_message` if the request errored

Read API: `GET /api/v1/audit-logs` — paged, filterable by username, userId, method, path, action, resourceType, resourceId, statusCode, requestId, from, to. Gated by `AUDIT_READ` permission.

Annotate handlers to label the action:
```go
audit.Action(c, "ORDER_CREATE", "Order", "")            // create
audit.Action(c, "ORDER_DELETE", "Order", idString)      // delete with id
audit.Skip(c)                                           // opt out
```

Toggles (env): `AUDIT_ENABLED`, `AUDIT_EXPOSE_API`, `AUDIT_CAPTURE_REQUEST_BODY`, `AUDIT_CAPTURE_RESPONSE_BODY`, `AUDIT_MAX_BODY_LENGTH`.

### Rate Limiting
Token bucket per identity (username if authenticated, else IP). Two buckets:
- default — applied to `/api/**`
- auth — stricter, applied to `/api/v1/auth/*`

Response headers when allowed: `X-RateLimit-Limit`, `X-RateLimit-Remaining`. When blocked: 429 + `Retry-After`.

Toggles (env): `RATE_LIMIT_ENABLED`, `RATE_LIMIT_CAPACITY`, `RATE_LIMIT_REFILL_TOKENS`, `RATE_LIMIT_REFILL_PERIOD`, plus `_AUTH_` variants for the stricter bucket.

> **Cluster note**: the in-memory map is fine for single-instance. For multi-instance, swap `ratelimit.Service` for a Redis-backed implementation — the middleware signature stays the same.

### Idempotency
Apply per-route. Replays cached response if the same `Idempotency-Key` header is sent within `IDEMPOTENCY_TTL`. Mismatched payload → 409.

```go
mutators := v1.Group("")
mutators.Use(idempotency.Middleware(deps.Idem, cfg.Idem))
```

### File Uploads
`POST /api/v1/files/upload` (authenticated, multipart). Subfolder sanitized; filenames become `<uuid>.<ext>` so uploads can't collide. `GET /api/v1/files/{subfolder}/{name}` is public so `<img>` works without a token.

### Mail
SMTP via `net/smtp`. Off by default (`MAIL_ENABLED=false`) so misconfigured local runs don't silently swallow notifications. Templates in `templates/email/*.html` use stdlib `html/template`.

### Logging — zap + rotation + per-domain files

The template uses **zap** for all logging with a tee:
- **stdout** — always on, console-encoded in dev (colored) or JSON in prod (`LOG_FORMAT`).
- **file sink** — `LOG_DIR/LOG_FILE` (default `logs/app.log`), JSON-encoded, rotated by [lumberjack](https://github.com/natefinch/lumberjack).

Rotation knobs (env):
```
LOG_MAX_SIZE_MB=100   # rotate when active file hits this size
LOG_MAX_BACKUPS=7     # keep at most N rotated files
LOG_MAX_AGE_DAYS=2    # delete rotated files older than N days
LOG_COMPRESS=true     # gzip rotated files
```

After rotation, `logs/` looks like:
```
logs/
├── app.log                                       # active, growing
├── app-2026-05-17T14-55-51.452.log.gz           # rotated, gzipped
└── app-2026-05-16T09-12-03.117.log.gz
```

Anything older than `LOG_MAX_AGE_DAYS` or beyond `LOG_MAX_BACKUPS` is auto-deleted. Set `LOG_FILE=` (empty) to disable file output and keep only stdout.

#### Writing logs

| What you need | Use this |
|---|---|
| Plain logger | `logger.L().Info("...", zap.Int(...))` |
| Request-scoped (carries `request_id`, `username`) | `logger.FromContext(ctx).Info(...)` |
| Custom file (no request context) | `logger.File("background.log").Info(...)` |
| Custom file + request fields | `logger.FromContextFile(ctx, "audit.log").Info(...)` |

#### Per-domain log files (recommended pattern)

For business events that deserve their own file, use the **applog** package — it routes the line to a domain file and automatically attaches the acting user (`actorId` + `actor`) from the security principal.

```go
import "github.com/AkashAhmed66/gin-template/internal/common/applog"

// Once at the package level — bind the filename
var plog = applog.Channel("products.log")

// At call sites — just pass ctx
plog(ctx).Info("product created",
    zap.Uint64("productId", p.ID),
    zap.String("sku", p.SKU),
)
```

The line lands in `logs/products.log` with stdout still receiving a copy. Caching means repeated `plog(ctx)` calls are cheap.

See [internal/modules/product/service.go](internal/modules/product/service.go) for the reference implementation.

> **Important**: `ctx` is unavoidable — it carries `requestId`, `username`, and the security principal. Without it the line in the file has no correlation back to the originating request. For truly stateless background logs use `logger.File("name.log")` instead.

### Background Scheduler — `jobs.Every` / `jobs.Cron` / `jobs.Once`

Recurring and one-off jobs running **in-process**. Survives the running process only — use the queue (next section) for work that must survive restarts.

```go
import "github.com/AkashAhmed66/gin-template/internal/common/jobs"

// Periodic — every N duration
jobs.Every("audit-flush", 30*time.Second, func(ctx context.Context) error {
    return s.flushPending(ctx)
})

// Cron — 5-field spec, evaluated in JOBS_TIMEZONE
jobs.Cron("nightly-report", "0 3 * * *", func(ctx context.Context) error {
    return s.generateReport(ctx)
})

// One-off — fires after delay
jobs.Once("retry-webhook", 5*time.Minute, func(ctx context.Context) error {
    return s.retryWebhook(ctx, id)
})
```

**What you get for free:** panic recovery (a panic in one job doesn't kill the others), startup jitter so jobs don't all fire at t=0, graceful shutdown via the lifecycle hook.

**Built-in jobs** registered by bootstrap:
- `session-cleanup` — purges expired/revoked sessions on `SESSIONS_CLEANUP_INTERVAL`
- `idempotency-cleanup` — drops expired idempotency records on `IDEMPOTENCY_CLEANUP_INTERVAL`
- `password-reset-cleanup` — drops consumed/expired reset tokens on `MAIL_PASSWORD_RESET_CLEANUP_INTERVAL`

**Cron quick reference:**
| Spec | Meaning |
|---|---|
| `*/15 * * * *` | every 15 minutes |
| `0 * * * *` | top of every hour |
| `0 3 * * *` | daily at 03:00 |
| `0 9 * * 1-5` | weekday mornings at 09:00 |
| `0 0 1 * *` | first day of every month at midnight |

### In-process Worker Pool — `jobs.Submit`

Bounded goroutine pool for fire-and-forget async work where loss-on-crash is acceptable.

```go
// Non-blocking submit. Returns false if queue is full.
ok := jobs.Submit(func(ctx context.Context) error {
    return s.cacheWarm(ctx, key)
})

// Blocking submit (backpressure-aware).
err := jobs.SubmitBlocking(ctx, func(ctx context.Context) error {
    return s.heavyTask(ctx)
})
```

**Capacity tuning:**
- `JOBS_WORKER_POOL_SIZE` (default 8) — concurrent workers
- `JOBS_QUEUE_SIZE` (default 1024) — buffered queue depth before `Submit` returns false

Use this for **ephemeral** work. For anything that **must run even if the server crashes** (emails, payments, webhooks), use the persistent queue instead.

### Persistent Task Queue — `queue.Enqueue` (Redis-backed, asynq)

For work that needs **persistence, retries, delays, and multi-instance distribution**. Backed by Redis via [hibiken/asynq](https://github.com/hibiken/asynq).

#### When to use which

| Use case | Use |
|---|---|
| Recurring cleanups, periodic syncs | `jobs.Every` / `jobs.Cron` |
| Fire-and-forget, OK to lose on crash (cache warm, log flush) | `jobs.Submit` |
| Email, push notification, webhook, payment, anything that **must** complete | `queue.Enqueue` |
| Scheduled work ("send in 24 hours", "bill on Jan 1") | `queue.EnqueueIn` / `queue.EnqueueAt` |

#### Setup

1. **Run Redis**:
   ```bash
   docker run -d --name redis -p 6379:6379 redis:7-alpine
   ```
2. **Enable in [.env](.env)**:
   ```
   QUEUE_ENABLED=true
   REDIS_ADDR=localhost:6379
   REDIS_PASSWORD=
   REDIS_DB=0
   QUEUE_CONCURRENCY=10
   QUEUE_PRIORITIES=critical=6,default=3,low=1
   ```
3. **Restart** — you'll see `redis queue enabled` in the logs.

If `QUEUE_ENABLED=false`, every `queue.Enqueue` returns `ErrNotInstalled` so callers can fall back to a synchronous path. See `ForgotPassword` in [internal/modules/auth/service.go](internal/modules/auth/service.go) for the pattern.

#### Defining a task

Two parts: **payload + handler** (registered once), and **enqueue calls** (anywhere).

```go
// 1. internal/modules/auth/queue.go — declare task type + payload + handler
const TaskPasswordResetEmail = "auth:password-reset-email"

type PasswordResetEmailPayload struct {
    Email    string `json:"email"`
    Username string `json:"username"`
    ResetURL string `json:"resetUrl"`
    TTL      string `json:"ttl"`
}

func RegisterQueueHandlers(q *queue.Manager, mailer mail.Service, log *zap.Logger) {
    q.Handle(TaskPasswordResetEmail, func(ctx context.Context, body []byte) error {
        var p PasswordResetEmailPayload
        if err := json.Unmarshal(body, &p); err != nil { return err }
        return sendEmail(ctx, mailer, p)
    })
}

// 2. internal/bootstrap/bootstrap.go — wire it once
if d.Queue != nil {
    authmod.RegisterQueueHandlers(d.Queue, d.Mail, d.Log)
}

// 3. Anywhere in your code — enqueue
queue.Enqueue(ctx, auth.TaskPasswordResetEmail, auth.PasswordResetEmailPayload{
    Email: u.Email, Username: u.Username, ResetURL: url, TTL: ttl,
})
```

#### Enqueue options

```go
queue.Enqueue(ctx, "report:daily", payload,
    queue.Queue("low"),                  // route to "low" priority lane
    queue.MaxRetry(3),                   // override default retries
    queue.Timeout(2*time.Minute),        // cancel handler after 2 min
    queue.Unique(1*time.Hour),           // dedup within 1h window
)

queue.EnqueueIn(ctx, "order:reminder", payload, 24*time.Hour)
queue.EnqueueAt(ctx, "subscription:bill", payload, billDate)
```

#### Built-in retry schedule

Failed tasks retry at: **1m → 5m → 30m → 1h → 6h → 24h**. After exhausting retries, tasks go to the **archived** state for inspection (and optional replay) — they're not silently lost.

#### Closures vs typed payloads — the one trade-off

The in-process worker pool (`jobs.Submit`) accepts Go closures that capture variables. The persistent queue **does not** — tasks must be JSON-serialisable payloads, because they're stored in Redis and a worker on a different instance can't dereference closure-captured variables. This is the standard model for any persistent queue (Sidekiq, Celery, Bull).

#### Observability — Asynqmon web UI

asynq ships with a free dashboard showing queue depth, active tasks, retries, archived failures (with payloads + error traces). The repo includes a [docker-compose.yml](docker-compose.yml) that starts both Redis and Asynqmon:

```bash
docker compose up -d
```

- Asynqmon UI → http://localhost:8081
- Redis → `localhost:6379`

Or run the binary directly:

```bash
go install github.com/hibiken/asynqmon/cmd/asynqmon@latest
asynqmon --redis-addr=localhost:6379 --port=8081
```

From the UI you can inspect any task's payload + error trace, re-enqueue archived failures (e.g. after fixing a bug), pause/unpause queues, and watch live throughput.

> **Production note**: Asynqmon ships with no auth. Bind it to localhost and tunnel over SSH, or put it behind a reverse proxy with auth. Don't expose it publicly.

#### Multi-instance behaviour

When multiple instances connect to the same Redis:
- **Enqueued tasks** are distributed across workers on any instance automatically.
- **Crashed instance** → in-flight tasks return to the queue for retry by another instance.
- **No duplicate processing** — exactly one worker picks up each task.

This is what unlocks horizontal scaling for async work.

### Validation & Exceptions
Gin's bound binding (`binding:"required,email,max=255"`) is enforced by validator.v10. Failures produce:
```json
{
  "success": false,
  "message": "Validation failed",
  "errors": { "email": "must be a well-formed email address" }
}
```

---

## Swagger / OpenAPI Annotations

Every handler in the template carries [swaggo/swag](https://github.com/swaggo/swag) annotations directly above the function. Running `swag init` (or `go generate ./...` or `make swag`) walks the source tree, parses these comments, and writes `docs/docs.go` + `docs/swagger.json` + `docs/swagger.yaml`. The blank import in [main.go](main.go) registers the spec at boot, and [internal/router/router.go](internal/router/router.go) mounts `gin-swagger` at `/swagger/*any`.

### The annotation contract

A complete annotation block looks like this:

```go
// methodName godoc
// @Summary      One-line summary shown in the UI list
// @Description  Longer description shown when the endpoint is expanded.
// @Tags         tag-name                         ← groups endpoints in the UI
// @Accept       json                              ← request Content-Type (json, multipart/form-data, ...)
// @Produce      json                              ← response Content-Type
// @Security     BearerAuth                        ← omit on public endpoints
// @Param        id     path     int     true  "user id"
// @Param        q      query    string  false "free-text query"
// @Param        body   body     LoginRequest  true "payload"
// @Param        file   formData file    true  "uploaded file"
// @Success      200  {object}  dto.ApiResponse{data=AuthResponse}
// @Failure      401  {object}  dto.ApiResponse
// @Router       /api/v1/auth/login [post]
func (h *Handler) login(c *gin.Context) (*dto.ApiResponse, error) { ... }
```

### Field-by-field

| Field | Required | Notes |
|---|---|---|
| `@Summary` | yes | One short sentence. Shown in the collapsed UI list. |
| `@Description` | no | Multi-line OK — continue on the next `// @Description` line. |
| `@Tags` | yes | Use the resource name (`users`, `roles`, `products`). Group admin endpoints under `admin-<resource>`. |
| `@Accept` | only for endpoints with a body | `json`, `multipart/form-data`, `application/x-www-form-urlencoded`, `xml` |
| `@Produce` | yes | Almost always `json`. Use `octet-stream` for file downloads. |
| `@Security` | only for protected endpoints | `BearerAuth` — defined in [main.go](main.go) via `@securityDefinitions.apikey BearerAuth`. |
| `@Param` | one per param | Format: `name location type required "description"`. Locations: `path`, `query`, `header`, `body`, `formData`. |
| `@Success` / `@Failure` | at least one `@Success` | Format: `status {schema-type} type-ref`. `{object}` for structs, `{array}` for slices, `{file}` for binaries, `{string}` for plain text. |
| `@Router` | yes | The route path **as the client sees it**, then `[method]`. Must match what the handler is registered at. |

### Response schema patterns

| What you want to show | Annotation |
|---|---|
| Plain envelope, no inner data | `@Success 200 {object} dto.ApiResponse` |
| Envelope wrapping one DTO | `@Success 200 {object} dto.ApiResponse{data=AuthResponse}` |
| Envelope wrapping a paged list | `@Success 200 {object} dto.ApiResponse{data=dto.PageResponse[user.Response]}` |
| File download | `@Success 200 {file} binary` |
| Just a status, no body | `@Success 204` |

The generic `dto.ApiResponse{data=...}` syntax is what makes the UI render the actual response shape — without it the client only sees the envelope.

### Cross-package types — the one gotcha

swag resolves type references based on the **current file's imports**. If your handler in package `auth` references `session.Response`, that package must be imported by the handler file. Two workarounds when it isn't:

1. **Drop the inner type** — leave the annotation as `dto.ApiResponse` with no `{data=...}`. The UI loses the inline schema for that one endpoint; everything else still works. (This is what [internal/modules/auth/handler.go](internal/modules/auth/handler.go)'s `mySessions` does.)
2. **Add a local alias** to the file just for the annotation: `import sessionmod "...session"` then write `data=[]sessionmod.Response`.

### Daily workflow

```powershell
# 1. Add or edit annotations on your handler
# 2. Regenerate
make swag                # or: go generate ./...   or: swag init --parseDependency --parseInternal
# 3. Restart the app
go run .
# 4. Refresh http://localhost:8080/swagger/index.html
```

If the UI shows **"Failed to load API definition — 500 doc.json"**, you forgot step 2 — the spec is still the old one (or the placeholder).

### Where everything lives

| File | Role |
|---|---|
| [main.go](main.go) | `@title` / `@version` / `@BasePath` / `@securityDefinitions` block. Blank-imports `docs/`. Carries the `go:generate` directive. |
| `docs/docs.go` | **Generated and gitignored.** `swag init` writes it. Required for the build — must exist before the first `go run .`. |
| `docs/swagger.json`, `docs/swagger.yaml` | **Generated and gitignored.** Same trigger as `docs.go`. |
| [internal/router/router.go](internal/router/router.go) | Mounts `gin-swagger` at `/swagger/*any`. |
| [internal/modules/*/handler.go](internal/modules/) | All endpoint-level annotations. Every existing endpoint already has one — copy the shape when adding new ones. |

### Troubleshooting

| Symptom | Fix |
|---|---|
| `Failed to load API definition — Internal Server Error doc.json` | Spec hasn't been generated. Run `go generate ./...` then restart. |
| `unknown field LeftDelim in Spec` build error | swag CLI is newer than the library — `go get github.com/swaggo/swag@latest && go mod tidy`. |
| `cannot find type definition: foo.Bar` while running `swag init` | The handler file doesn't import that package. Use a local alias or drop the `data=` portion. |
| `cannot find package "github.com/.../docs"` build error after a fresh clone | `docs/` is gitignored — generate it before the first build: `swag init --parseDependency --parseInternal` (or `make swag`). |
| `cannot find file: main.go` from `swag init` | You ran swag from a subdirectory. Run it from the project root: `cd c:\projects\go\gin-template; swag init --parseDependency --parseInternal`. |
| Empty `paths: {}` in the generated spec | swag couldn't find your handlers. Make sure you're running from the project root and pass `--parseDependency --parseInternal`. |
| New endpoint missing from the UI | Forgot to re-run `swag init` after adding annotations, or your `@Router` path doesn't match what's actually registered. |
| UI shows the spec but `Authorize` button doesn't work | `@Security BearerAuth` is missing on the endpoint, or the `@securityDefinitions.apikey BearerAuth` block in `main.go` was deleted. |

---

## Configuration Reference

Lookup order (highest priority first): OS env → `.env` → in-code default.

See [.env.example](.env.example) — every variable is documented inline. Highlights:

### Application
| Variable | Default | Purpose |
|---|---|---|
| `APP_NAME` | `gin-template` | App name |
| `APP_ENV` | `dev` | `dev` or `prod` (changes gin mode + log defaults) |
| `SERVER_PORT` | `8080` | HTTP port |
| `SERVER_SHUTDOWN_TIMEOUT` | `30s` | Graceful shutdown deadline |

### Database
| Variable | Default | Purpose |
|---|---|---|
| `DB_DRIVER` | `postgres` | `postgres` or `mysql` |
| `DB_HOST` / `DB_PORT` / `DB_USER` / `DB_PASSWORD` / `DB_NAME` | localhost stack | Connection |
| `DB_SSLMODE` | `disable` | Postgres only |
| `DB_POOL_MAX_OPEN` | `20` | Pool size |

### JWT / Sessions
| Variable | Default | Purpose |
|---|---|---|
| `JWT_SECRET` | placeholder | **Override in prod**: `openssl rand -base64 32` |
| `JWT_ACCESS_TTL` | `60m` | Access token lifetime |
| `JWT_REFRESH_TTL` | `336h` (14d) | Refresh token + session row lifetime |
| `SESSIONS_CLEANUP_INTERVAL` | `1h` | Cleanup job cadence |
| `SESSIONS_CLEANUP_RETENTION` | `168h` (7d) | Retention past expiry for forensics |

### Logging
| Variable | Default | Purpose |
|---|---|---|
| `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |
| `LOG_FORMAT` | `console` | `console` (colored, for terminals) or `json` (for log aggregators) |
| `LOG_DIR` | `logs` | Directory for log files (empty disables file output) |
| `LOG_FILE` | `app.log` | Filename inside `LOG_DIR` (empty disables file output) |
| `LOG_MAX_SIZE_MB` | `100` | Rotate when active log file reaches this size |
| `LOG_MAX_BACKUPS` | `7` | Keep at most N rotated files (0 = unlimited) |
| `LOG_MAX_AGE_DAYS` | `2` | Delete rotated files older than N days (0 = no age limit) |
| `LOG_COMPRESS` | `true` | gzip rotated files |

### Background jobs (in-process scheduler + worker pool)
| Variable | Default | Purpose |
|---|---|---|
| `JOBS_ENABLED` | `true` | Master switch for the in-process scheduler + worker pool |
| `JOBS_WORKER_POOL_SIZE` | `8` | Concurrent workers consuming the queue |
| `JOBS_QUEUE_SIZE` | `1024` | Buffered queue depth before `Submit` returns false |
| `JOBS_TIMEZONE` | `UTC` | IANA tz for cron specs (e.g. `Asia/Dhaka`) |

### Redis (for the persistent queue)
| Variable | Default | Purpose |
|---|---|---|
| `REDIS_ADDR` | `localhost:6379` | Redis host:port |
| `REDIS_PASSWORD` | _(empty)_ | Auth password if Redis requires one |
| `REDIS_DB` | `0` | Logical DB number (0–15) |

### Persistent task queue (asynq)
| Variable | Default | Purpose |
|---|---|---|
| `QUEUE_ENABLED` | `false` | Turn on the asynq queue (requires reachable Redis) |
| `QUEUE_CONCURRENCY` | `10` | Total workers on this instance across all queues |
| `QUEUE_PRIORITIES` | `critical=6,default=3,low=1` | `name=weight,...` for weighted round-robin |

Full reference in [.env.example](.env.example).

---

## Built-in Endpoints

| Method   | Path                                        | Auth | Permission        | Purpose |
|----------|---------------------------------------------|------|-------------------|---------|
| `GET`    | `/`                                         | —    | —                 | App info |
| `GET`    | `/health`                                   | —    | —                 | Liveness |
| `POST`   | `/api/v1/auth/register`                     | —    | —                 | Self-service register |
| `POST`   | `/api/v1/auth/login`                        | —    | —                 | Login |
| `POST`   | `/api/v1/auth/refresh`                      | —    | —                 | Rotate tokens |
| `POST`   | `/api/v1/auth/forgot-password`              | —    | —                 | Request reset email |
| `POST`   | `/api/v1/auth/reset-password`               | —    | —                 | Consume reset token |
| `POST`   | `/api/v1/auth/logout`                       | ✓    | —                 | Revoke this device |
| `POST`   | `/api/v1/auth/logout-all`                   | ✓    | —                 | Revoke all my sessions |
| `GET`    | `/api/v1/auth/sessions`                     | ✓    | —                 | List my sessions |
| `DELETE` | `/api/v1/auth/sessions/{id}`                | ✓    | —                 | Revoke one of my sessions |
| `POST`   | `/api/v1/auth/impersonate/{userId}`         | ✓    | `USER_IMPERSONATE`| Admin impersonation |
| `GET`    | `/api/v1/users/me`                          | ✓    | —                 | Current profile |
| `GET`    | `/api/v1/users`                             | ✓    | `USER_READ`       | List users (paged + filter) |
| `GET`    | `/api/v1/users/{id}`                        | ✓    | `USER_READ`       | Fetch user |
| `PUT`    | `/api/v1/users/{id}`                        | ✓    | `USER_WRITE`      | Update user |
| `POST`   | `/api/v1/users/{id}/activate`               | ✓    | `USER_WRITE`      | Enable user |
| `POST`   | `/api/v1/users/{id}/deactivate`             | ✓    | `USER_WRITE`      | Disable user + revoke sessions |
| `POST`   | `/api/v1/users/{id}/force-logout`           | ✓    | `USER_WRITE`      | Revoke user's sessions |
| `PUT`    | `/api/v1/users/{id}/roles`                  | ✓    | `USER_WRITE`      | Replace user's role set |
| `DELETE` | `/api/v1/users/{id}`                        | ✓    | `USER_DELETE`     | Soft-delete user |
| `GET`/`POST`/`PUT`/`DELETE` | `/api/v1/roles[/{id}[/permissions]]` | ✓ | `ROLE_*` | Role CRUD + assign permissions |
| `GET`/`POST`/`PUT`/`DELETE` | `/api/v1/permissions[/{id}]` | ✓ | `PERMISSION_*` | Permission CRUD |
| `GET`    | `/api/v1/admin/sessions`                    | ✓    | `SESSION_READ`    | Cross-user session search |
| `GET`    | `/api/v1/admin/sessions/{id}`               | ✓    | `SESSION_READ`    | Inspect any session |
| `DELETE` | `/api/v1/admin/sessions/{id}`               | ✓    | `SESSION_REVOKE`  | Revoke any session |
| `GET`    | `/api/v1/audit-logs`                        | ✓    | `AUDIT_READ`      | Paged audit query |
| `POST`   | `/api/v1/files/upload`                      | ✓    | —                 | Upload a file |
| `GET`    | `/api/v1/files/{subfolder}/{name}`          | —    | —                 | Serve a file |
| `GET`/`POST`/`PUT`/`DELETE` | `/api/v1/products[/{id}[/image]]` | ✓ | `PRODUCT_*` | Reference module |

---

## Database Migrations

Migrations live in [migrations/](migrations/), written for goose. On boot they run automatically when `MIGRATIONS_AUTO_RUN=true` (default).

Manual control:
```bash
# install goose once
go install github.com/pressly/goose/v3/cmd/goose@latest

# create a new SQL migration
goose -dir migrations create add_orders sql

# apply / roll back
goose -dir migrations postgres "postgres://user:pass@localhost/db?sslmode=disable" up
goose -dir migrations postgres "postgres://user:pass@localhost/db?sslmode=disable" down
```

---

## Build, Test, Run

```bash
# Sync deps
go mod tidy

# Generate Swagger docs/ (gitignored — REQUIRED before the first build/run)
swag init --parseDependency --parseInternal

# Run
go run .

# Hot reload (requires air)
air

# Tests
go test -race ./...

# Build
go build -o bin/gin-template .
./bin/gin-template
```

See [Makefile](Makefile) for the full set of dev commands.

---

## What's Intentionally NOT in the Box

- **Caching layer** — Spring's `@Cacheable` is replaceable in idiomatic Go with a service-layer interface + Redis adapter. Add when you need it; don't pre-build.
- **Distributed rate limiting** — single-instance buckets are enough until you scale horizontally. Swap to Redis (`bucket4j-redis` equivalent: Go has `github.com/redis/go-redis/v9`-backed implementations) without touching the middleware.
- **Multi-tenancy** — out of scope; add at the model layer if needed (every entity gets a `tenant_id` column + the JWT carries `tid`).
- **Tests / CI / Dockerfile** — see the audit in the docs for the suggested ordering.
