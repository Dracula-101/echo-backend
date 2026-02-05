package middleware

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type RateLimiterConfig struct {
	RequestsPerSecond float64
	BurstSize         int
	CleanupInterval   time.Duration
}

func DefaultRateLimiterConfig() *RateLimiterConfig {
	return &RateLimiterConfig{
		RequestsPerSecond: 20,
		BurstSize:         50,
		CleanupInterval:   5 * time.Minute,
	}
}

type RateLimiter struct {
	limiters map[string]*rateLimiterEntry
	config   *RateLimiterConfig
	mu       sync.RWMutex
	stopCh   chan struct{}
}

type rateLimiterEntry struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

func NewRateLimiter(cfg *RateLimiterConfig) *RateLimiter {
	if cfg == nil {
		cfg = DefaultRateLimiterConfig()
	}

	rl := &RateLimiter{
		limiters: make(map[string]*rateLimiterEntry),
		config:   cfg,
		stopCh:   make(chan struct{}),
	}

	go rl.runCleanup()
	return rl
}

func (rl *RateLimiter) Allow(key string) bool {
	limiter := rl.getLimiter(key)
	return limiter.Allow()
}

func (rl *RateLimiter) AllowN(key string, n int) bool {
	limiter := rl.getLimiter(key)
	return limiter.AllowN(time.Now(), n)
}

func (rl *RateLimiter) Wait(key string) {
	limiter := rl.getLimiter(key)
	_ = limiter.Wait(context.TODO())
}

func (rl *RateLimiter) Reserve(key string) *rate.Reservation {
	limiter := rl.getLimiter(key)
	return limiter.Reserve()
}

func (rl *RateLimiter) Tokens(key string) float64 {
	limiter := rl.getLimiter(key)
	return limiter.Tokens()
}

func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.RLock()
	entry, exists := rl.limiters[key]
	rl.mu.RUnlock()

	if exists {
		rl.mu.Lock()
		entry.lastAccess = time.Now()
		rl.mu.Unlock()
		return entry.limiter
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	if entry, exists = rl.limiters[key]; exists {
		entry.lastAccess = time.Now()
		return entry.limiter
	}

	limiter := rate.NewLimiter(rate.Limit(rl.config.RequestsPerSecond), rl.config.BurstSize)
	rl.limiters[key] = &rateLimiterEntry{
		limiter:    limiter,
		lastAccess: time.Now(),
	}
	return limiter
}

func (rl *RateLimiter) Remove(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.limiters, key)
}

func (rl *RateLimiter) Clear() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.limiters = make(map[string]*rateLimiterEntry)
}

func (rl *RateLimiter) runCleanup() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.cleanup()
		case <-rl.stopCh:
			return
		}
	}
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	threshold := time.Now().Add(-rl.config.CleanupInterval)

	for key, entry := range rl.limiters {
		if entry.lastAccess.Before(threshold) {
			delete(rl.limiters, key)
		}
	}
}

func (rl *RateLimiter) Stop() {
	close(rl.stopCh)
}

func (rl *RateLimiter) GetStats() RateLimiterStats {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	return RateLimiterStats{
		ActiveLimiters:    len(rl.limiters),
		RequestsPerSecond: rl.config.RequestsPerSecond,
		BurstSize:         rl.config.BurstSize,
	}
}

type RateLimiterStats struct {
	ActiveLimiters    int     `json:"active_limiters"`
	RequestsPerSecond float64 `json:"requests_per_second"`
	BurstSize         int     `json:"burst_size"`
}
