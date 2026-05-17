package ratelimit

import (
	"sync"
	"time"

	"github.com/AkashAhmed66/gin-template/internal/config"
)

// Service holds two pools of buckets keyed by identity (username or IP):
//   - default: applied to all /api/** paths.
//   - auth:    stricter bucket applied to /api/v1/auth/* to mitigate stuffing.
//
// Concurrency: a per-pool sync.Map is fine for the volumes we expect from a
// single-instance deployment. Switch to Redis-backed buckets when running
// multi-instance.
type Service struct {
	cfg     config.RateLimitConfig
	defaul  sync.Map // map[string]*Bucket
	auth    sync.Map // map[string]*Bucket
}

// NewService builds a Service from typed config.
func NewService(cfg config.RateLimitConfig) *Service {
	return &Service{cfg: cfg}
}

// Enabled returns whether the master switch is on.
func (s *Service) Enabled() bool { return s.cfg.Enabled }

// IsAuthPath reports whether path falls inside one of the auth prefixes.
func (s *Service) IsAuthPath(path string) bool {
	for _, p := range s.cfg.AuthPathPrefixes {
		if hasPrefix(path, p) {
			return true
		}
	}
	return false
}

// IsIncluded reports whether path is one of the include prefixes (every other
// path skips rate limiting entirely).
func (s *Service) IsIncluded(path string) bool {
	if len(s.cfg.IncludePrefixes) == 0 {
		return true
	}
	for _, p := range s.cfg.IncludePrefixes {
		if hasPrefix(path, p) {
			return true
		}
	}
	return false
}

// TakeDefault consumes 1 token from the default bucket for the given identity.
func (s *Service) TakeDefault(identity string) (bool, int, int, time.Duration) {
	b := s.bucketFor(&s.defaul, identity, s.cfg.Capacity, s.cfg.RefillTokens, s.cfg.RefillPeriod)
	allowed, remaining, retry := b.Take()
	return allowed, remaining, b.Capacity(), retry
}

// TakeAuth consumes 1 token from the auth bucket for the given identity.
func (s *Service) TakeAuth(identity string) (bool, int, int, time.Duration) {
	b := s.bucketFor(&s.auth, identity, s.cfg.AuthCapacity, s.cfg.AuthRefillTokens, s.cfg.AuthRefillPeriod)
	allowed, remaining, retry := b.Take()
	return allowed, remaining, b.Capacity(), retry
}

func (s *Service) bucketFor(store *sync.Map, identity string, cap, refill int, period time.Duration) *Bucket {
	if v, ok := store.Load(identity); ok {
		return v.(*Bucket)
	}
	nb := NewBucket(cap, refill, period)
	actual, _ := store.LoadOrStore(identity, nb)
	return actual.(*Bucket)
}

func hasPrefix(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	return s[:len(prefix)] == prefix
}
