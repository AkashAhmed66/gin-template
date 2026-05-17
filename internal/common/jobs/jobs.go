// Package jobs is the application-wide entry point for background scheduling
// and async work. It wraps a scheduler.Scheduler and a worker.Pool so any
// service can submit tasks or register recurring jobs without dependency
// injection plumbing.
//
// Bootstrap installs the Manager exactly once via Set. After that:
//
//	jobs.Every("audit-flush", 30*time.Second, func(ctx context.Context) error { ... })
//	jobs.Cron("nightly-report", "0 3 * * *", func(ctx context.Context) error { ... })
//	jobs.Submit(func(ctx context.Context) error { sendEmail(...); return nil })
//	jobs.SubmitBlocking(ctx, taskFn)
//
// If a call is made before Set, the recurring/scheduling helpers return
// ErrNotInstalled and Submit returns false.
package jobs

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/AkashAhmed66/gin-template/internal/common/scheduler"
	"github.com/AkashAhmed66/gin-template/internal/common/worker"
)

// Manager bundles the scheduler + worker pool. Constructed in bootstrap and
// installed via Set; access from anywhere via the package-level helpers.
type Manager struct {
	Scheduler *scheduler.Scheduler
	Pool      *worker.Pool
}

// ErrNotInstalled is returned by helpers when no Manager has been installed.
var ErrNotInstalled = errors.New("jobs: no manager installed (call jobs.Set during bootstrap)")

var current atomic.Pointer[Manager]

// Set installs the application-wide Manager. Replaces any previous one.
func Set(m *Manager) { current.Store(m) }

// Get returns the installed Manager, or nil.
func Get() *Manager { return current.Load() }

// Submit enqueues a non-blocking task on the global worker pool.
// Returns false if no pool is installed or the queue is full.
func Submit(t worker.Task) bool {
	m := current.Load()
	if m == nil || m.Pool == nil {
		return false
	}
	return m.Pool.Submit(t)
}

// SubmitBlocking enqueues a task, blocking until queued or ctx expires.
func SubmitBlocking(ctx context.Context, t worker.Task) error {
	m := current.Load()
	if m == nil || m.Pool == nil {
		return ErrNotInstalled
	}
	return m.Pool.SubmitBlocking(ctx, t)
}

// Every registers an interval-based recurring job on the global scheduler.
func Every(name string, interval time.Duration, fn scheduler.JobFunc) error {
	m := current.Load()
	if m == nil || m.Scheduler == nil {
		return ErrNotInstalled
	}
	return m.Scheduler.Every(name, interval, fn)
}

// Cron registers a cron-expression recurring job on the global scheduler.
func Cron(name, spec string, fn scheduler.JobFunc) error {
	m := current.Load()
	if m == nil || m.Scheduler == nil {
		return ErrNotInstalled
	}
	return m.Scheduler.Cron(name, spec, fn)
}

// Once schedules a one-off job after delay.
func Once(name string, delay time.Duration, fn scheduler.JobFunc) error {
	m := current.Load()
	if m == nil || m.Scheduler == nil {
		return ErrNotInstalled
	}
	m.Scheduler.Once(name, delay, fn)
	return nil
}
