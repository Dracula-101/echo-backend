package health

import (
	"context"
	"sync"
)

type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
)

type CheckResult struct {
	Name    string                 `json:"name"`
	Status  Status                 `json:"status"`
	Message string                 `json:"message,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

type Checker interface {
	Name() string
	Check(ctx context.Context) CheckResult
}

type Manager struct {
	serviceName    string
	serviceVersion string
	checkers       []Checker
	mu             sync.RWMutex
}

func NewManager(serviceName, serviceVersion string) *Manager {
	return &Manager{
		serviceName:    serviceName,
		serviceVersion: serviceVersion,
		checkers:       make([]Checker, 0),
	}
}

func (m *Manager) RegisterChecker(checker Checker) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checkers = append(m.checkers, checker)
}

func (m *Manager) Check(ctx context.Context) map[string]interface{} {
	m.mu.RLock()
	checkers := make([]Checker, len(m.checkers))
	copy(checkers, m.checkers)
	m.mu.RUnlock()

	results := make([]CheckResult, 0, len(checkers))
	overallStatus := StatusHealthy

	for _, checker := range checkers {
		result := checker.Check(ctx)
		results = append(results, result)

		if result.Status == StatusUnhealthy {
			overallStatus = StatusUnhealthy
		} else if result.Status == StatusDegraded && overallStatus == StatusHealthy {
			overallStatus = StatusDegraded
		}
	}

	return map[string]interface{}{
		"service": m.serviceName,
		"version": m.serviceVersion,
		"status":  overallStatus,
		"checks":  results,
	}
}

func (m *Manager) Liveness(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"service": m.serviceName,
		"version": m.serviceVersion,
		"status":  StatusHealthy,
	}
}

func (m *Manager) Readiness(ctx context.Context) map[string]interface{} {
	return m.Check(ctx)
}
