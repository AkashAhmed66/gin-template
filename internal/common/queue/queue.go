// Package queue is the application's persistent, Redis-backed task queue. It
// wraps hibiken/asynq with a small, opinionated API.
//
// Use it for any work you want to run async with persistence + retries:
//   - Sending emails / push notifications
//   - Calling slow external APIs
//   - Generating reports / PDFs
//   - Webhook delivery
//
// Two-phase usage:
//
//  1. Register handlers once at startup (typically from a service constructor
//     or from bootstrap):
//
//     queue.Handle("auth:password-reset-email", emailHandler)
//
//  2. Enqueue tasks from anywhere:
//
//     queue.Enqueue(ctx, "auth:password-reset-email", PasswordResetPayload{
//         Email: u.Email, Token: token,
//     })
//
// Tasks are JSON-serialised, persisted in Redis, and processed by workers
// running on any instance. Failed tasks are retried with exponential backoff
// and ultimately archived for inspection.
package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/AkashAhmed66/gin-template/internal/config"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

// Handler is the signature for a task processor. The payload arrives as raw
// JSON; decode it into the type your enqueue-side used.
type Handler func(ctx context.Context, payload []byte) error

// Manager bundles the asynq client (enqueue side) + server (process side) +
// mux (handler registry). One instance per process; install it via Set.
type Manager struct {
	client *asynq.Client
	server *asynq.Server
	mux    *asynq.ServeMux
	log    *zap.Logger
}

// ErrNotInstalled is returned by Enqueue / Handle when no Manager has been
// installed via Set. Usually means the queue is disabled in config.
var ErrNotInstalled = errors.New("queue: no manager installed (QUEUE_ENABLED=false?)")

// New builds a Manager from typed config. Returns nil + nil when the queue is
// disabled, so callers can use the result with Set() unconditionally.
func New(cfg config.QueueConfig, redis config.RedisConfig, log *zap.Logger) (*Manager, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	if log == nil {
		log = zap.NewNop()
	}
	redisOpt := asynq.RedisClientOpt{
		Addr:     redis.Addr,
		Password: redis.Password,
		DB:       redis.DB,
	}

	queues := cfg.Queues
	if len(queues) == 0 {
		queues = map[string]int{"default": 1}
	}

	srvCfg := asynq.Config{
		Concurrency: cfg.Concurrency,
		Queues:      queues,
		RetryDelayFunc: func(n int, _ error, _ *asynq.Task) time.Duration {
			// 1m, 5m, 30m, 1h, 6h ... capped at 24h.
			base := []time.Duration{
				time.Minute, 5 * time.Minute, 30 * time.Minute,
				time.Hour, 6 * time.Hour, 24 * time.Hour,
			}
			if n >= len(base) {
				return base[len(base)-1]
			}
			return base[n]
		},
		Logger: &asynqLogger{log: log},
	}

	return &Manager{
		client: asynq.NewClient(redisOpt),
		server: asynq.NewServer(redisOpt, srvCfg),
		mux:    asynq.NewServeMux(),
		log:    log,
	}, nil
}

// Handle registers a handler for a task type. Call before Start. Idempotent
// per task type — the last registration wins.
func (m *Manager) Handle(taskType string, h Handler) {
	if m == nil {
		return
	}
	m.mux.HandleFunc(taskType, func(ctx context.Context, t *asynq.Task) error {
		return h(ctx, t.Payload())
	})
}

// Start begins processing tasks. Blocks the goroutine; run from a separate
// goroutine in bootstrap.
func (m *Manager) Start() error {
	if m == nil {
		return nil
	}
	return m.server.Start(m.mux)
}

// Stop drains in-flight tasks (up to ctx deadline) and closes the client.
func (m *Manager) Stop(ctx context.Context) {
	if m == nil {
		return
	}
	m.server.Shutdown()
	_ = m.client.Close()
}

// Enqueue submits a task for immediate processing. Returns ErrNotInstalled
// if the queue is disabled.
func (m *Manager) Enqueue(ctx context.Context, taskType string, payload any, opts ...Option) error {
	if m == nil {
		return ErrNotInstalled
	}
	body, err := encode(payload)
	if err != nil {
		return fmt.Errorf("queue: marshal payload for %q: %w", taskType, err)
	}
	asynqOpts := applyOptions(opts)
	_, err = m.client.EnqueueContext(ctx, asynq.NewTask(taskType, body), asynqOpts...)
	return err
}

// EnqueueIn submits a task to be processed after delay.
func (m *Manager) EnqueueIn(ctx context.Context, taskType string, payload any, delay time.Duration, opts ...Option) error {
	return m.Enqueue(ctx, taskType, payload, append(opts, ProcessIn(delay))...)
}

// EnqueueAt submits a task to be processed at a specific time.
func (m *Manager) EnqueueAt(ctx context.Context, taskType string, payload any, at time.Time, opts ...Option) error {
	return m.Enqueue(ctx, taskType, payload, append(opts, ProcessAt(at))...)
}

// --- package-level convenience API -------------------------------------------

var current atomic.Pointer[Manager]

// Set installs the application-wide Manager. Call once from bootstrap.
func Set(m *Manager) { current.Store(m) }

// Get returns the installed Manager, or nil.
func Get() *Manager { return current.Load() }

// Handle registers a handler on the global Manager.
func Handle(taskType string, h Handler) { current.Load().Handle(taskType, h) }

// Enqueue submits a task on the global Manager.
func Enqueue(ctx context.Context, taskType string, payload any, opts ...Option) error {
	return current.Load().Enqueue(ctx, taskType, payload, opts...)
}

// EnqueueIn submits a delayed task on the global Manager.
func EnqueueIn(ctx context.Context, taskType string, payload any, delay time.Duration, opts ...Option) error {
	return current.Load().EnqueueIn(ctx, taskType, payload, delay, opts...)
}

// EnqueueAt submits a task scheduled at `at` on the global Manager.
func EnqueueAt(ctx context.Context, taskType string, payload any, at time.Time, opts ...Option) error {
	return current.Load().EnqueueAt(ctx, taskType, payload, at, opts...)
}

// --- options -----------------------------------------------------------------

// Option configures a single Enqueue call.
type Option func(*optionSet)

type optionSet struct {
	queue      string
	maxRetry   int
	taskID     string
	timeout    time.Duration
	processIn  time.Duration
	processAt  time.Time
	uniqueFor  time.Duration
}

// Queue overrides the destination queue (e.g. "critical", "low").
func Queue(name string) Option { return func(o *optionSet) { o.queue = name } }

// MaxRetry overrides the retry attempts for this task.
func MaxRetry(n int) Option { return func(o *optionSet) { o.maxRetry = n } }

// TaskID assigns a deterministic ID — used for dedup with Unique.
func TaskID(id string) Option { return func(o *optionSet) { o.taskID = id } }

// Timeout sets a per-task timeout (handler ctx is cancelled after).
func Timeout(d time.Duration) Option { return func(o *optionSet) { o.timeout = d } }

// ProcessIn delays processing by d.
func ProcessIn(d time.Duration) Option { return func(o *optionSet) { o.processIn = d } }

// ProcessAt schedules processing for a specific time.
func ProcessAt(t time.Time) Option { return func(o *optionSet) { o.processAt = t } }

// Unique enforces "at most one in flight" semantics for this task ID within
// the given window. Submitting a duplicate within the window is a no-op.
func Unique(d time.Duration) Option { return func(o *optionSet) { o.uniqueFor = d } }

func applyOptions(opts []Option) []asynq.Option {
	var o optionSet
	for _, fn := range opts {
		fn(&o)
	}
	out := make([]asynq.Option, 0, 4)
	if o.queue != "" {
		out = append(out, asynq.Queue(o.queue))
	}
	if o.maxRetry > 0 {
		out = append(out, asynq.MaxRetry(o.maxRetry))
	}
	if o.taskID != "" {
		out = append(out, asynq.TaskID(o.taskID))
	}
	if o.timeout > 0 {
		out = append(out, asynq.Timeout(o.timeout))
	}
	if o.processIn > 0 {
		out = append(out, asynq.ProcessIn(o.processIn))
	}
	if !o.processAt.IsZero() {
		out = append(out, asynq.ProcessAt(o.processAt))
	}
	if o.uniqueFor > 0 {
		out = append(out, asynq.Unique(o.uniqueFor))
	}
	return out
}

func encode(payload any) ([]byte, error) {
	if payload == nil {
		return []byte("{}"), nil
	}
	if b, ok := payload.([]byte); ok {
		return b, nil
	}
	return json.Marshal(payload)
}

// asynqLogger adapts zap to asynq's logger interface.
type asynqLogger struct{ log *zap.Logger }

func (l *asynqLogger) Debug(args ...any) { l.log.Sugar().Debug(args...) }
func (l *asynqLogger) Info(args ...any)  { l.log.Sugar().Info(args...) }
func (l *asynqLogger) Warn(args ...any)  { l.log.Sugar().Warn(args...) }
func (l *asynqLogger) Error(args ...any) { l.log.Sugar().Error(args...) }
func (l *asynqLogger) Fatal(args ...any) { l.log.Sugar().Fatal(args...) }
