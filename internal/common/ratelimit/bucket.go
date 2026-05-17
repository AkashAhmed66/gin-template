// Package ratelimit is an in-memory per-identity token-bucket limiter. The
// public surface mirrors Bucket4j's TokenBucket from the spring-boot template:
// each identity gets its own bucket lazily, refill happens at a fixed period,
// and the middleware emits the standard X-RateLimit-* response headers.
//
// For multi-instance deployments, swap the in-memory store for Redis without
// changing the middleware signature.
package ratelimit

import (
	"math"
	"sync"
	"time"
)

// Bucket is a leaky/token bucket. Safe for concurrent use.
type Bucket struct {
	mu               sync.Mutex
	capacity         int
	tokens           float64
	refillTokens     int
	refillPeriod     time.Duration
	lastRefill       time.Time
}

// NewBucket returns a full bucket sized to capacity.
func NewBucket(capacity, refillTokens int, refillPeriod time.Duration) *Bucket {
	return &Bucket{
		capacity:     capacity,
		tokens:       float64(capacity),
		refillTokens: refillTokens,
		refillPeriod: refillPeriod,
		lastRefill:   time.Now(),
	}
}

// Take attempts to consume 1 token. Returns (allowed, remaining, retryAfter).
// retryAfter is non-zero only when allowed=false — the duration the caller
// must wait for the next refill that would let one token through.
func (b *Bucket) Take() (bool, int, time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.refill()
	if b.tokens >= 1 {
		b.tokens--
		return true, int(b.tokens), 0
	}
	missing := 1 - b.tokens
	tokensPerSec := float64(b.refillTokens) / b.refillPeriod.Seconds()
	if tokensPerSec <= 0 {
		return false, 0, b.refillPeriod
	}
	waitSec := missing / tokensPerSec
	return false, 0, time.Duration(math.Ceil(waitSec)) * time.Second
}

// Capacity returns the bucket size.
func (b *Bucket) Capacity() int { return b.capacity }

func (b *Bucket) refill() {
	now := time.Now()
	elapsed := now.Sub(b.lastRefill)
	if elapsed <= 0 {
		return
	}
	tokensPerSec := float64(b.refillTokens) / b.refillPeriod.Seconds()
	b.tokens = math.Min(float64(b.capacity), b.tokens+elapsed.Seconds()*tokensPerSec)
	b.lastRefill = now
}
