package health

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Status represents health status
type Status int

const (
	// StatusHealthy indicates healthy state
	StatusHealthy Status = iota
	// StatusDegraded indicates degraded state
	StatusDegraded
	// StatusUnhealthy indicates unhealthy state
	StatusUnhealthy
)

// String returns string representation of status
func (s Status) String() string {
	switch s {
	case StatusHealthy:
		return "healthy"
	case StatusDegraded:
		return "degraded"
	case StatusUnhealthy:
		return "unhealthy"
	default:
		return "unknown"
	}
}

// Check is a health check function
type Check func(ctx context.Context) error

// Result represents health check result
type Result struct {
	Name      string
	Status    Status
	Timestamp time.Time
	Error     error
	Duration  time.Duration
}

// Checker performs health checks
type Checker struct {
	checks   map[string]Check
	results  map[string]*Result
	mu       sync.RWMutex

	status atomic.Value // Status

	interval time.Duration
	stopCh   chan struct{}
}

// NewChecker creates a new health checker
func NewChecker(interval time.Duration) *Checker {
	c := &Checker{
		checks:   make(map[string]Check),
		results:  make(map[string]*Result),
		interval: interval,
		stopCh:   make(chan struct{}),
	}
	c.status.Store(StatusHealthy)
	return c
}

// RegisterCheck registers a health check
func (c *Checker) RegisterCheck(name string, check Check) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.checks[name] = check
}

// RunCheck runs a specific health check
func (c *Checker) RunCheck(ctx context.Context, name string) *Result {
	c.mu.RLock()
	check, exists := c.checks[name]
	c.mu.RUnlock()

	if !exists {
		return &Result{
			Name:      name,
			Status:    StatusUnhealthy,
			Timestamp: time.Now(),
			Error:     ErrCheckNotFound,
		}
	}

	start := time.Now()
	err := check(ctx)
	duration := time.Since(start)

	status := StatusHealthy
	if err != nil {
		status = StatusUnhealthy
	}

	result := &Result{
		Name:      name,
		Status:    status,
		Timestamp: time.Now(),
		Error:     err,
		Duration:  duration,
	}

	c.mu.Lock()
	c.results[name] = result
	c.mu.Unlock()

	return result
}

// RunAllChecks runs all registered health checks
func (c *Checker) RunAllChecks(ctx context.Context) map[string]*Result {
	c.mu.RLock()
	checkNames := make([]string, 0, len(c.checks))
	for name := range c.checks {
		checkNames = append(checkNames, name)
	}
	c.mu.RUnlock()

	results := make(map[string]*Result)
	for _, name := range checkNames {
		results[name] = c.RunCheck(ctx, name)
	}

	// Update overall status
	c.updateStatus(results)

	return results
}

// GetStatus returns the current health status
func (c *Checker) GetStatus() Status {
	return c.status.Load().(Status)
}

// updateStatus updates overall health status
func (c *Checker) updateStatus(results map[string]*Result) {
	hasUnhealthy := false
	hasDegraded := false

	for _, result := range results {
		if result.Status == StatusUnhealthy {
			hasUnhealthy = true
			break
		}
		if result.Status == StatusDegraded {
			hasDegraded = true
		}
	}

	if hasUnhealthy {
		c.status.Store(StatusUnhealthy)
	} else if hasDegraded {
		c.status.Store(StatusDegraded)
	} else {
		c.status.Store(StatusHealthy)
	}
}

// Start starts periodic health checks
func (c *Checker) Start(ctx context.Context) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.RunAllChecks(ctx)
		}
	}
}

// Stop stops periodic health checks
func (c *Checker) Stop() {
	close(c.stopCh)
}
