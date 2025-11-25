package websocket

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// TokenBucketRateLimiter implements rate limiting using token bucket algorithm
type TokenBucketRateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
}

// NewTokenBucketRateLimiter creates a new token bucket rate limiter
func NewTokenBucketRateLimiter(ratePerSecond int, burst int) *TokenBucketRateLimiter {
	return &TokenBucketRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rate.Limit(ratePerSecond),
		burst:    burst,
	}
}

// Allow checks if the client is allowed to send a message
func (rl *TokenBucketRateLimiter) Allow(clientID string) bool {
	rl.mu.RLock()
	limiter, exists := rl.limiters[clientID]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		// Double-check after acquiring write lock
		limiter, exists = rl.limiters[clientID]
		if !exists {
			limiter = rate.NewLimiter(rl.rate, rl.burst)
			rl.limiters[clientID] = limiter
		}
		rl.mu.Unlock()
	}

	return limiter.Allow()
}

// Reset removes the rate limiter for a client
func (rl *TokenBucketRateLimiter) Reset(clientID string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.limiters, clientID)
}

// Cleanup removes rate limiters for inactive clients
func (rl *TokenBucketRateLimiter) Cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// In a production environment, you'd track last access time
	// and remove stale entries
	// For now, we just clear the map
	rl.limiters = make(map[string]*rate.Limiter)
}

// SlidingWindowRateLimiter implements rate limiting using sliding window algorithm
type SlidingWindowRateLimiter struct {
	windows     map[string]*window
	mu          sync.RWMutex
	maxRequests int
	windowSize  time.Duration
}

type window struct {
	timestamps []time.Time
	mu         sync.Mutex
}

// NewSlidingWindowRateLimiter creates a new sliding window rate limiter
func NewSlidingWindowRateLimiter(maxRequests int, windowSize time.Duration) *SlidingWindowRateLimiter {
	return &SlidingWindowRateLimiter{
		windows:     make(map[string]*window),
		maxRequests: maxRequests,
		windowSize:  windowSize,
	}
}

// Allow checks if the client is allowed to send a message
func (rl *SlidingWindowRateLimiter) Allow(clientID string) bool {
	rl.mu.RLock()
	w, exists := rl.windows[clientID]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		// Double-check after acquiring write lock
		w, exists = rl.windows[clientID]
		if !exists {
			w = &window{
				timestamps: make([]time.Time, 0, rl.maxRequests),
			}
			rl.windows[clientID] = w
		}
		rl.mu.Unlock()
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.windowSize)

	// Remove timestamps outside the window
	validTimestamps := make([]time.Time, 0, len(w.timestamps))
	for _, ts := range w.timestamps {
		if ts.After(cutoff) {
			validTimestamps = append(validTimestamps, ts)
		}
	}
	w.timestamps = validTimestamps

	// Check if limit is exceeded
	if len(w.timestamps) >= rl.maxRequests {
		return false
	}

	// Add current timestamp
	w.timestamps = append(w.timestamps, now)
	return true
}

// Reset removes the rate limiter for a client
func (rl *SlidingWindowRateLimiter) Reset(clientID string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.windows, clientID)
}

// Cleanup removes rate limiters for inactive clients
func (rl *SlidingWindowRateLimiter) Cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.windowSize * 2) // Keep for 2x window size

	for clientID, w := range rl.windows {
		w.mu.Lock()
		if len(w.timestamps) == 0 || w.timestamps[len(w.timestamps)-1].Before(cutoff) {
			delete(rl.windows, clientID)
		}
		w.mu.Unlock()
	}
}

// NoOpRateLimiter is a rate limiter that allows everything
type NoOpRateLimiter struct{}

// NewNoOpRateLimiter creates a new no-op rate limiter
func NewNoOpRateLimiter() *NoOpRateLimiter {
	return &NoOpRateLimiter{}
}

// Allow always returns true
func (rl *NoOpRateLimiter) Allow(clientID string) bool {
	return true
}

// Reset does nothing
func (rl *NoOpRateLimiter) Reset(clientID string) {
	// No-op
}

// CompositeRateLimiter combines multiple rate limiters
type CompositeRateLimiter struct {
	limiters []RateLimiter
}

// NewCompositeRateLimiter creates a new composite rate limiter
func NewCompositeRateLimiter(limiters ...RateLimiter) *CompositeRateLimiter {
	return &CompositeRateLimiter{
		limiters: limiters,
	}
}

// Allow checks if all rate limiters allow the request
func (rl *CompositeRateLimiter) Allow(clientID string) bool {
	for _, limiter := range rl.limiters {
		if !limiter.Allow(clientID) {
			return false
		}
	}
	return true
}

// Reset resets all rate limiters for a client
func (rl *CompositeRateLimiter) Reset(clientID string) {
	for _, limiter := range rl.limiters {
		limiter.Reset(clientID)
	}
}

// IPBasedRateLimiter implements rate limiting based on IP address
type IPBasedRateLimiter struct {
	limiter RateLimiter
	clients map[string]string // clientID -> IP
	mu      sync.RWMutex
}

// NewIPBasedRateLimiter creates a new IP-based rate limiter
func NewIPBasedRateLimiter(ratePerSecond int, burst int) *IPBasedRateLimiter {
	return &IPBasedRateLimiter{
		limiter: NewTokenBucketRateLimiter(ratePerSecond, burst),
		clients: make(map[string]string),
	}
}

// RegisterClient registers a client with their IP
func (rl *IPBasedRateLimiter) RegisterClient(clientID, ipAddress string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.clients[clientID] = ipAddress
}

// Allow checks if the client's IP is allowed
func (rl *IPBasedRateLimiter) Allow(clientID string) bool {
	rl.mu.RLock()
	ip, exists := rl.clients[clientID]
	rl.mu.RUnlock()

	if !exists {
		// If client not registered, allow by default
		return true
	}

	return rl.limiter.Allow(ip)
}

// Reset removes the client and resets their IP limit
func (rl *IPBasedRateLimiter) Reset(clientID string) {
	rl.mu.Lock()
	ip, exists := rl.clients[clientID]
	if exists {
		delete(rl.clients, clientID)
		rl.limiter.Reset(ip)
	}
	rl.mu.Unlock()
}
