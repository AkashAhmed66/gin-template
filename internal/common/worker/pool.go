// Package worker provides a bounded pool of goroutines that drain Tasks from
// a queue. Use it for fire-and-forget async work (sending emails, generating
// reports, calling slow external APIs).
//
// Typical lifecycle:
//
//	pool := worker.NewPool(8, 1024, log)
//	pool.Start()
//	// ... anywhere ...
//	pool.Submit(func(ctx context.Context) error { sendEmail(...) ; return nil })
//	// ... shutdown ...
//	pool.Stop(shutdownCtx)
package worker

import (
	"context"
	"errors"
	"runtime/debug"
	"sync"

	"go.uber.org/zap"
)

// Task is the signature for a unit of background work. The ctx passed in is
// cancelled when Stop is called.
type Task func(ctx context.Context) error

// ErrPoolStopped is returned by SubmitBlocking when the pool has been stopped.
var ErrPoolStopped = errors.New("worker pool stopped")

// Pool runs Task functions through N workers drawing from a buffered queue.
type Pool struct {
	size  int
	queue chan Task
	log   *zap.Logger

	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	startOnce sync.Once
	stopOnce  sync.Once
}

// NewPool builds an idle pool with `size` workers and a buffered queue of
// `queueSize`. queueSize=0 yields an unbuffered queue (Submit always returns
// false when no worker is free). Call Start to spin workers up.
func NewPool(size, queueSize int, log *zap.Logger) *Pool {
	if size <= 0 {
		size = 4
	}
	if queueSize < 0 {
		queueSize = 0
	}
	if log == nil {
		log = zap.NewNop()
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Pool{
		size:   size,
		queue:  make(chan Task, queueSize),
		log:    log,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start spawns worker goroutines. Idempotent.
func (p *Pool) Start() {
	p.startOnce.Do(func() {
		for i := 0; i < p.size; i++ {
			p.wg.Add(1)
			go p.worker(i)
		}
		p.log.Info("worker pool started",
			zap.Int("size", p.size),
			zap.Int("queueCap", cap(p.queue)),
		)
	})
}

// Submit attempts to enqueue t without blocking. Returns false if the queue
// is full or the pool has been stopped.
func (p *Pool) Submit(t Task) bool {
	select {
	case <-p.ctx.Done():
		return false
	default:
	}
	select {
	case p.queue <- t:
		return true
	default:
		return false
	}
}

// SubmitBlocking enqueues t, blocking until queued, ctx times out, or the
// pool is stopped.
func (p *Pool) SubmitBlocking(ctx context.Context, t Task) error {
	select {
	case p.queue <- t:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-p.ctx.Done():
		return ErrPoolStopped
	}
}

// Stop cancels worker contexts and waits for them to drain or for ctx to expire.
func (p *Pool) Stop(ctx context.Context) {
	p.stopOnce.Do(func() {
		p.cancel()
		done := make(chan struct{})
		go func() {
			p.wg.Wait()
			close(done)
		}()
		select {
		case <-done:
			p.log.Info("worker pool stopped")
		case <-ctx.Done():
			p.log.Warn("worker pool stop timed out")
		}
	})
}

func (p *Pool) worker(id int) {
	defer p.wg.Done()
	for {
		select {
		case <-p.ctx.Done():
			return
		case t, ok := <-p.queue:
			if !ok {
				return
			}
			p.run(t)
		}
	}
}

func (p *Pool) run(t Task) {
	defer func() {
		if r := recover(); r != nil {
			p.log.Error("worker task panicked",
				zap.Any("panic", r),
				zap.ByteString("stack", debug.Stack()),
			)
		}
	}()
	if err := t(p.ctx); err != nil {
		p.log.Warn("worker task failed", zap.Error(err))
	}
}
