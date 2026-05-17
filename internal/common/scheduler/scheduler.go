// Package scheduler runs recurring (interval and cron) and one-off background
// jobs with panic recovery, jitter on startup, and graceful shutdown.
//
// Typical lifecycle:
//
//	sched := scheduler.New(log, time.UTC)
//	sched.Every("audit-flush", 30*time.Second, func(ctx context.Context) error { ... })
//	sched.Cron("nightly-report", "0 3 * * *", func(ctx context.Context) error { ... })
//	sched.Start()
//	// ... app runs ...
//	sched.Stop(shutdownCtx)
package scheduler

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// JobFunc is the signature for a scheduled task. The ctx passed in is the
// scheduler's lifetime context — it cancels when Stop is called, allowing
// long-running jobs to bail out cooperatively.
type JobFunc func(ctx context.Context) error

// Scheduler manages recurring and one-off jobs.
type Scheduler struct {
	mu        sync.Mutex
	cron      *cron.Cron
	intervals []intervalJob
	log       *zap.Logger

	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool
}

type intervalJob struct {
	name     string
	interval time.Duration
	fn       JobFunc
}

// New returns an idle Scheduler. Call Start to activate.
// Pass nil location to default to UTC.
func New(log *zap.Logger, tz *time.Location) *Scheduler {
	if log == nil {
		log = zap.NewNop()
	}
	if tz == nil {
		tz = time.UTC
	}
	cl := cronLogger{log: log}
	return &Scheduler{
		cron: cron.New(
			cron.WithLocation(tz),
			cron.WithLogger(cl),
			cron.WithChain(cron.Recover(cl)),
		),
		log: log,
	}
}

// Every registers fn to run on a fixed interval. Safe to call before or after Start.
func (s *Scheduler) Every(name string, interval time.Duration, fn JobFunc) error {
	if interval <= 0 {
		return fmt.Errorf("scheduler.Every(%q): interval must be > 0", name)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	job := intervalJob{name: name, interval: interval, fn: fn}
	s.intervals = append(s.intervals, job)
	if s.started {
		s.spawnInterval(job)
	}
	return nil
}

// Cron registers fn to run on a 5-field cron expression. Standard syntax:
//
//	"0 3 * * *"     daily at 03:00 in the scheduler's timezone
//	"*/15 * * * *"  every 15 minutes
//	"0 0 * * 0"     weekly on Sunday at midnight
func (s *Scheduler) Cron(name, spec string, fn JobFunc) error {
	wrapped := func() {
		s.run(name, fn)
	}
	if _, err := s.cron.AddFunc(spec, wrapped); err != nil {
		return fmt.Errorf("scheduler.Cron(%q): invalid spec %q: %w", name, spec, err)
	}
	return nil
}

// Once schedules fn to run exactly once after delay. Cancelled if Stop fires
// before delay elapses.
func (s *Scheduler) Once(name string, delay time.Duration, fn JobFunc) {
	s.mu.Lock()
	if !s.started {
		// Defer: spawn at Start time.
		s.intervals = append(s.intervals, intervalJob{
			name:     "__once:" + name,
			interval: delay,
			fn:       fn,
		})
		s.mu.Unlock()
		return
	}
	ctx := s.ctx
	s.mu.Unlock()
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		t := time.NewTimer(delay)
		defer t.Stop()
		select {
		case <-ctx.Done():
		case <-t.C:
			s.run(name, fn)
		}
	}()
}

// Start activates the scheduler. Safe to call once.
func (s *Scheduler) Start() {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.started = true
	intervals := append([]intervalJob(nil), s.intervals...)
	s.mu.Unlock()

	for _, j := range intervals {
		s.spawnInterval(j)
	}
	s.cron.Start()
	s.log.Info("scheduler started",
		zap.Int("intervalJobs", len(intervals)),
		zap.Int("cronJobs", len(s.cron.Entries())),
	)
}

// Stop signals all running jobs to cancel, then waits for them to finish or
// until ctx expires.
func (s *Scheduler) Stop(ctx context.Context) {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return
	}
	s.started = false
	s.cancel()
	s.mu.Unlock()

	cronDone := s.cron.Stop()

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		<-cronDone.Done()
		close(done)
	}()

	select {
	case <-done:
		s.log.Info("scheduler stopped")
	case <-ctx.Done():
		s.log.Warn("scheduler stop timed out — some jobs may still be running")
	}
}

func (s *Scheduler) spawnInterval(j intervalJob) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		// Stagger startup: wait a small jittered delay so jobs don't all fire at t=0.
		jitter := j.interval / 10
		if jitter > 5*time.Second {
			jitter = 5 * time.Second
		}
		if jitter > 0 {
			t := time.NewTimer(jitter)
			select {
			case <-s.ctx.Done():
				t.Stop()
				return
			case <-t.C:
			}
		}

		ticker := time.NewTicker(j.interval)
		defer ticker.Stop()
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-ticker.C:
				s.run(j.name, j.fn)
			}
		}
	}()
}

// run invokes fn with the scheduler ctx, recovering panics and logging errors.
func (s *Scheduler) run(name string, fn JobFunc) {
	defer func() {
		if r := recover(); r != nil {
			s.log.Error("scheduled job panicked",
				zap.String("job", name),
				zap.Any("panic", r),
				zap.ByteString("stack", debug.Stack()),
			)
		}
	}()
	if err := fn(s.ctx); err != nil {
		s.log.Warn("scheduled job failed", zap.String("job", name), zap.Error(err))
	}
}

// cronLogger adapts zap to cron's logger interface.
type cronLogger struct{ log *zap.Logger }

func (c cronLogger) Info(msg string, keysAndValues ...any) {
	c.log.Debug("cron: "+msg, zap.Any("ctx", keysAndValues))
}

func (c cronLogger) Error(err error, msg string, keysAndValues ...any) {
	c.log.Warn("cron: "+msg, zap.Error(err), zap.Any("ctx", keysAndValues))
}
