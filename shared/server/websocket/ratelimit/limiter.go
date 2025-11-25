package ratelimit

import (
	"context"
	"sync"

	"golang.org/x/time/rate"
)

// Limiter is a rate limiter interface
type Limiter interface {
	Allow(key string) bool
	Wait(key string) error
	Reset(key string)
}

// TokenBucketLimiter uses token bucket algorithm
type TokenBucketLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex

	rate  rate.Limit
	burst int
}

// NewTokenBucketLimiter creates a new token bucket limiter
func NewTokenBucketLimiter(ratePerSec int, burst int) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rate.Limit(ratePerSec),
		burst:    burst,
	}
}

// Allow checks if an action is allowed
func (l *TokenBucketLimiter) Allow(key string) bool {
	return l.getLimiter(key).Allow()
}

// Wait waits until action is allowed
func (l *TokenBucketLimiter) Wait(key string) error {
	return l.getLimiter(key).Wait(context.Background())
}

// Reset resets the limiter for a key
func (l *TokenBucketLimiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.limiters, key)
}

// getLimiter gets or creates a limiter for a key
func (l *TokenBucketLimiter) getLimiter(key string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	limiter, exists := l.limiters[key]
	if !exists {
		limiter = rate.NewLimiter(l.rate, l.burst)
		l.limiters[key] = limiter
	}

	return limiter
}
