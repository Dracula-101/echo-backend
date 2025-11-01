package health

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Status represents the health status
type Status string

const (
	// StatusUp indicates the component is healthy
	StatusUp Status = "up"

	// StatusDown indicates the component is unhealthy
	StatusDown Status = "down"

	// StatusDegraded indicates the component is partially healthy
	StatusDegraded Status = "degraded"
)

// CheckResult represents the result of a health check
type CheckResult struct {
	Status    Status                 `json:"status"`
	Message   string                 `json:"message,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// CheckFunc is a function that performs a health check
type CheckFunc func(context.Context) CheckResult

// Checker represents a health checker
type Checker struct {
	name      string
	checkFunc CheckFunc
	timeout   time.Duration
	critical  bool
}

// Check performs the health check
func (c *Checker) Check(ctx context.Context) CheckResult {
	if c.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	resultChan := make(chan CheckResult, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				resultChan <- CheckResult{
					Status:    StatusDown,
					Message:   fmt.Sprintf("panic during health check: %v", r),
					Timestamp: time.Now(),
				}
			}
		}()
		resultChan <- c.checkFunc(ctx)
	}()

	select {
	case result := <-resultChan:
		return result
	case <-ctx.Done():
		return CheckResult{
			Status:    StatusDown,
			Message:   "health check timeout",
			Timestamp: time.Now(),
		}
	}
}

// Name returns the checker name
func (c *Checker) Name() string {
	return c.name
}

// IsCritical returns whether this check is critical
func (c *Checker) IsCritical() bool {
	return c.critical
}

// Health manages multiple health checks
type Health struct {
	checkers map[string]*Checker
	mu       sync.RWMutex
}

// New creates a new Health instance
func New() *Health {
	return &Health{
		checkers: make(map[string]*Checker),
	}
}

// RegisterCheck registers a new health check
func (h *Health) RegisterCheck(name string, checkFunc CheckFunc, opts ...Option) {
	h.mu.Lock()
	defer h.mu.Unlock()

	checker := &Checker{
		name:      name,
		checkFunc: checkFunc,
		timeout:   5 * time.Second,
		critical:  false,
	}

	for _, opt := range opts {
		opt(checker)
	}

	h.checkers[name] = checker
}

// UnregisterCheck removes a health check
func (h *Health) UnregisterCheck(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.checkers, name)
}

// Check performs all health checks
func (h *Health) Check(ctx context.Context) HealthReport {
	h.mu.RLock()
	checkers := make([]*Checker, 0, len(h.checkers))
	for _, checker := range h.checkers {
		checkers = append(checkers, checker)
	}
	h.mu.RUnlock()

	results := make(map[string]CheckResult)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, checker := range checkers {
		wg.Add(1)
		go func(c *Checker) {
			defer wg.Done()
			result := c.Check(ctx)

			mu.Lock()
			results[c.Name()] = result
			mu.Unlock()
		}(checker)
	}

	wg.Wait()

	return h.buildReport(results)
}

// CheckOne performs a single health check by name
func (h *Health) CheckOne(ctx context.Context, name string) (CheckResult, bool) {
	h.mu.RLock()
	checker, exists := h.checkers[name]
	h.mu.RUnlock()

	if !exists {
		return CheckResult{}, false
	}

	return checker.Check(ctx), true
}

func (h *Health) buildReport(results map[string]CheckResult) HealthReport {
	h.mu.RLock()
	defer h.mu.RUnlock()

	report := HealthReport{
		Status:    StatusUp,
		Checks:    results,
		Timestamp: time.Now(),
	}

	// Determine overall status
	hasDown := false
	hasDegraded := false

	for name, result := range results {
		checker := h.checkers[name]

		switch result.Status {
		case StatusDown:
			if checker != nil && checker.critical {
				report.Status = StatusDown
				return report
			}
			hasDown = true
		case StatusDegraded:
			hasDegraded = true
		}
	}

	if hasDown {
		report.Status = StatusDown
	} else if hasDegraded {
		report.Status = StatusDegraded
	}

	return report
}

// HealthReport represents the overall health status
type HealthReport struct {
	Status    Status                 `json:"status"`
	Checks    map[string]CheckResult `json:"checks"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// IsHealthy returns true if the overall status is up
func (r HealthReport) IsHealthy() bool {
	return r.Status == StatusUp
}

// Option is a functional option for configuring a Checker
type Option func(*Checker)

// WithTimeout sets the timeout for a health check
func WithTimeout(timeout time.Duration) Option {
	return func(c *Checker) {
		c.timeout = timeout
	}
}

// WithCritical marks a health check as critical
// If a critical check fails, the overall status is down
func WithCritical() Option {
	return func(c *Checker) {
		c.critical = true
	}
}

// Common health check implementations

// AlwaysHealthy returns a check that always succeeds
func AlwaysHealthy() CheckFunc {
	return func(ctx context.Context) CheckResult {
		return CheckResult{
			Status:    StatusUp,
			Timestamp: time.Now(),
		}
	}
}

// DatabaseCheck creates a health check for a database connection
func DatabaseCheck(pingFunc func(context.Context) error) CheckFunc {
	return func(ctx context.Context) CheckResult {
		err := pingFunc(ctx)
		if err != nil {
			return CheckResult{
				Status:    StatusDown,
				Message:   fmt.Sprintf("database ping failed: %v", err),
				Timestamp: time.Now(),
			}
		}

		return CheckResult{
			Status:    StatusUp,
			Message:   "database is healthy",
			Timestamp: time.Now(),
		}
	}
}

// RedisCheck creates a health check for a Redis connection
func RedisCheck(pingFunc func(context.Context) error) CheckFunc {
	return func(ctx context.Context) CheckResult {
		err := pingFunc(ctx)
		if err != nil {
			return CheckResult{
				Status:    StatusDown,
				Message:   fmt.Sprintf("redis ping failed: %v", err),
				Timestamp: time.Now(),
			}
		}

		return CheckResult{
			Status:    StatusUp,
			Message:   "redis is healthy",
			Timestamp: time.Now(),
		}
	}
}

// HTTPCheck creates a health check for an HTTP endpoint
func HTTPCheck(url string, httpClient interface {
	Get(string) (interface {
		StatusCode() int
		Close() error
	}, error)
}) CheckFunc {
	return func(ctx context.Context) CheckResult {
		resp, err := httpClient.Get(url)
		if err != nil {
			return CheckResult{
				Status:    StatusDown,
				Message:   fmt.Sprintf("http check failed: %v", err),
				Timestamp: time.Now(),
			}
		}
		defer resp.Close()

		if resp.StatusCode() >= 200 && resp.StatusCode() < 300 {
			return CheckResult{
				Status:    StatusUp,
				Message:   "http endpoint is healthy",
				Timestamp: time.Now(),
				Metadata: map[string]interface{}{
					"status_code": resp.StatusCode(),
				},
			}
		}

		return CheckResult{
			Status:    StatusDown,
			Message:   fmt.Sprintf("http check returned status %d", resp.StatusCode()),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"status_code": resp.StatusCode(),
			},
		}
	}
}

// MemoryCheck creates a health check that monitors memory usage
func MemoryCheck(getMemStats func() (alloc, totalAlloc, sys uint64)) CheckFunc {
	return func(ctx context.Context) CheckResult {
		alloc, totalAlloc, sys := getMemStats()

		// Simple heuristic: if allocated memory is more than 90% of system memory, degrade
		threshold := float64(sys) * 0.9
		status := StatusUp
		message := "memory usage is healthy"

		if float64(alloc) > threshold {
			status = StatusDegraded
			message = "memory usage is high"
		}

		return CheckResult{
			Status:    status,
			Message:   message,
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"alloc_mb":       alloc / 1024 / 1024,
				"total_alloc_mb": totalAlloc / 1024 / 1024,
				"sys_mb":         sys / 1024 / 1024,
			},
		}
	}
}

// GoroutineCheck creates a health check that monitors goroutine count
func GoroutineCheck(getGoroutineCount func() int, threshold int) CheckFunc {
	return func(ctx context.Context) CheckResult {
		count := getGoroutineCount()
		status := StatusUp
		message := "goroutine count is healthy"

		if count > threshold {
			status = StatusDegraded
			message = fmt.Sprintf("high goroutine count: %d (threshold: %d)", count, threshold)
		}

		return CheckResult{
			Status:    status,
			Message:   message,
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"goroutine_count": count,
				"threshold":       threshold,
			},
		}
	}
}

// DiskSpaceCheck creates a health check for disk space
func DiskSpaceCheck(getSpaceFunc func() (used, total uint64), thresholdPercent float64) CheckFunc {
	return func(ctx context.Context) CheckResult {
		used, total := getSpaceFunc()
		usedPercent := float64(used) / float64(total) * 100

		status := StatusUp
		message := "disk space is healthy"

		if usedPercent > thresholdPercent {
			status = StatusDegraded
			message = fmt.Sprintf("disk space usage is high: %.2f%%", usedPercent)
		}

		if usedPercent > 95 {
			status = StatusDown
			message = fmt.Sprintf("disk space critically low: %.2f%%", usedPercent)
		}

		return CheckResult{
			Status:    status,
			Message:   message,
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"used_gb":      used / 1024 / 1024 / 1024,
				"total_gb":     total / 1024 / 1024 / 1024,
				"used_percent": usedPercent,
			},
		}
	}
}

// CompositeCheck combines multiple checks into one
func CompositeCheck(checks map[string]CheckFunc) CheckFunc {
	return func(ctx context.Context) CheckResult {
		results := make(map[string]CheckResult)
		var wg sync.WaitGroup
		var mu sync.Mutex

		for name, checkFunc := range checks {
			wg.Add(1)
			go func(n string, fn CheckFunc) {
				defer wg.Done()
				result := fn(ctx)
				mu.Lock()
				results[n] = result
				mu.Unlock()
			}(name, checkFunc)
		}

		wg.Wait()

		// Determine overall status
		status := StatusUp
		messages := make([]string, 0)

		for _, result := range results {
			if result.Status == StatusDown {
				status = StatusDown
			} else if result.Status == StatusDegraded && status != StatusDown {
				status = StatusDegraded
			}

			if result.Message != "" {
				messages = append(messages, result.Message)
			}
		}

		message := ""
		if len(messages) > 0 {
			message = fmt.Sprintf("%d checks: %v", len(checks), messages)
		}

		return CheckResult{
			Status:    status,
			Message:   message,
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"checks": results,
			},
		}
	}
}
