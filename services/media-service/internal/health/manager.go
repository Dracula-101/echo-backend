package health

import (
	"context"
	"sync"
	"time"
)

// Status represents health check status
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

// CheckResult represents the result of a health check
type CheckResult struct {
	Status    Status                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// Checker interface for health checks
type Checker interface {
	Name() string
	Check(ctx context.Context) CheckResult
}

// Manager manages health checks
type Manager struct {
	serviceName string
	version     string
	checkers    []Checker
	mu          sync.RWMutex
}

// NewManager creates a new health check manager
func NewManager(serviceName, version string) *Manager {
	return &Manager{
		serviceName: serviceName,
		version:     version,
		checkers:    make([]Checker, 0),
	}
}

// RegisterChecker registers a health checker
func (m *Manager) RegisterChecker(checker Checker) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checkers = append(m.checkers, checker)
}

// CheckHealth runs all health checks
func (m *Manager) CheckHealth(ctx context.Context) HealthResponse {
	m.mu.RLock()
	checkers := make([]Checker, len(m.checkers))
	copy(checkers, m.checkers)
	m.mu.RUnlock()

	results := make(map[string]CheckResult)
	overallStatus := StatusHealthy

	for _, checker := range checkers {
		result := checker.Check(ctx)
		results[checker.Name()] = result

		// Determine overall status
		if result.Status == StatusUnhealthy {
			overallStatus = StatusUnhealthy
		} else if result.Status == StatusDegraded && overallStatus != StatusUnhealthy {
			overallStatus = StatusDegraded
		}
	}

	return HealthResponse{
		Status:    string(overallStatus),
		Service:   m.serviceName,
		Version:   m.version,
		Timestamp: time.Now(),
		Checks:    results,
	}
}

// HealthResponse represents the overall health response
type HealthResponse struct {
	Status    string                 `json:"status"`
	Service   string                 `json:"service"`
	Version   string                 `json:"version"`
	Timestamp time.Time              `json:"timestamp"`
	Checks    map[string]CheckResult `json:"checks"`
}
