// Package dedup provides idempotency tracking for messages flowing through
// ws-service. The Kafka consumer is at-least-once: rebalances or partition
// re-fetch after a handler error can re-deliver the same record. Without dedup
// the client would receive duplicate frames.
//
// This implementation is in-memory per pod with periodic eviction. When Redis
// Cluster lands (P3) the SeenWithin call can be swapped for a SETNX EX without
// changing the call sites.
package dedup

import (
	"sync"
	"time"
)

// Tracker remembers seen IDs for up to TTL. SeenWithin is the only required op:
// returns true if this is the first time the ID has been seen within TTL,
// false if it's a duplicate.
type Tracker struct {
	ttl     time.Duration
	mu      sync.Mutex
	entries map[string]time.Time
	stop    chan struct{}
}

func NewTracker(ttl, evictInterval time.Duration) *Tracker {
	t := &Tracker{
		ttl:     ttl,
		entries: make(map[string]time.Time),
		stop:    make(chan struct{}),
	}
	go t.runEviction(evictInterval)
	return t
}

// SeenWithin returns true if this is the first observation of id within the
// TTL window. A return of false means the caller should skip processing.
func (t *Tracker) SeenWithin(id string) bool {
	now := time.Now()
	t.mu.Lock()
	defer t.mu.Unlock()
	if expiry, ok := t.entries[id]; ok && expiry.After(now) {
		return false
	}
	t.entries[id] = now.Add(t.ttl)
	return true
}

func (t *Tracker) Stop() {
	close(t.stop)
}

func (t *Tracker) runEviction(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-t.stop:
			return
		case <-ticker.C:
			t.evict()
		}
	}
}

func (t *Tracker) evict() {
	now := time.Now()
	t.mu.Lock()
	defer t.mu.Unlock()
	for id, expiry := range t.entries {
		if expiry.Before(now) {
			delete(t.entries, id)
		}
	}
}

// Size returns the current entry count. Useful for metrics.
func (t *Tracker) Size() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.entries)
}
