package health

import (
	"context"
	"sync"
	"time"
)

type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
)

type Check struct {
	Name     string        `json:"name"`
	Status   Status        `json:"status"`
	Message  string        `json:"message,omitempty"`
	Duration time.Duration `json:"duration"`
}

type Response struct {
	Service string           `json:"service"`
	Version string           `json:"version"`
	Status  Status           `json:"status"`
	Checks  map[string]Check `json:"checks,omitempty"`
}

type Checker interface {
	Name() string
	Check(ctx context.Context) (Status, string)
}

type Manager struct {
	serviceName string
	version     string
	checkers    []Checker
	mu          sync.RWMutex
}

func NewManager(serviceName, version string) *Manager {
	return &Manager{
		serviceName: serviceName,
		version:     version,
		checkers:    make([]Checker, 0),
	}
}

func (m *Manager) RegisterChecker(checker Checker) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checkers = append(m.checkers, checker)
}

func (m *Manager) Check(ctx context.Context) Response {
	m.mu.RLock()
	checkers := m.checkers
	m.mu.RUnlock()

	checks := make(map[string]Check)
	overallStatus := StatusHealthy

	for _, checker := range checkers {
		start := time.Now()
		status, message := checker.Check(ctx)
		duration := time.Since(start)

		checks[checker.Name()] = Check{
			Name:     checker.Name(),
			Status:   status,
			Message:  message,
			Duration: duration,
		}

		if status == StatusUnhealthy {
			overallStatus = StatusUnhealthy
		} else if status == StatusDegraded && overallStatus != StatusUnhealthy {
			overallStatus = StatusDegraded
		}
	}

	return Response{
		Service: m.serviceName,
		Version: m.version,
		Status:  overallStatus,
		Checks:  checks,
	}
}

func (m *Manager) Liveness() Response {
	return Response{
		Service: m.serviceName,
		Version: m.version,
		Status:  StatusHealthy,
	}
}

func (m *Manager) Readiness(ctx context.Context) Response {
	return m.Check(ctx)
}
