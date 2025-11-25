package ratelimit

import (
	"sync"
	"time"
)

// SlidingWindowLimiter uses sliding window algorithm
type SlidingWindowLimiter struct {
	windows map[string]*window
	mu      sync.RWMutex

	limit      int
	windowSize time.Duration
}

type window struct {
	timestamps []time.Time
	mu         sync.Mutex
}

// NewSlidingWindowLimiter creates a new sliding window limiter
func NewSlidingWindowLimiter(limit int, windowSize time.Duration) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		windows:    make(map[string]*window),
		limit:      limit,
		windowSize: windowSize,
	}
}

// Allow checks if an action is allowed
func (l *SlidingWindowLimiter) Allow(key string) bool {
	w := l.getWindow(key)
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.windowSize)

	// Remove old timestamps
	newTimestamps := make([]time.Time, 0)
	for _, ts := range w.timestamps {
		if ts.After(cutoff) {
			newTimestamps = append(newTimestamps, ts)
		}
	}
	w.timestamps = newTimestamps

	// Check limit
	if len(w.timestamps) >= l.limit {
		return false
	}

	w.timestamps = append(w.timestamps, now)
	return true
}

// Wait is not supported for sliding window
func (l *SlidingWindowLimiter) Wait(key string) error {
	return ErrWaitNotSupported
}

// Reset resets the window for a key
func (l *SlidingWindowLimiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.windows, key)
}

// getWindow gets or creates a window for a key
func (l *SlidingWindowLimiter) getWindow(key string) *window {
	l.mu.Lock()
	defer l.mu.Unlock()

	w, exists := l.windows[key]
	if !exists {
		w = &window{
			timestamps: make([]time.Time, 0),
		}
		l.windows[key] = w
	}

	return w
}
