package health

import (
	"context"
	"net/http"
	"sync"
	"time"
)

type Checker interface {
	Name() string
	Check(ctx context.Context) CheckResult
}

type Manager struct {
	serviceName string
	version     string
	startTime   time.Time
	checkers    map[string]Checker
	mu          sync.RWMutex
	cache       map[string]cachedResult
	cacheTTL    time.Duration
}

type cachedResult struct {
	result    CheckResult
	timestamp time.Time
}

func NewManager(serviceName, version string) *Manager {
	return &Manager{
		serviceName: serviceName,
		version:     version,
		startTime:   time.Now(),
		checkers:    make(map[string]Checker),
		cache:       make(map[string]cachedResult),
		cacheTTL:    5 * time.Second,
	}
}

func (m *Manager) RegisterChecker(checker Checker) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checkers[checker.Name()] = checker
}

func (m *Manager) Health(ctx context.Context, includeChecks bool) Response {
	resp := Response{
		Status:    StatusHealthy,
		Timestamp: time.Now(),
		Service:   m.serviceName,
		Version:   m.version,
		Uptime:    time.Since(m.startTime).String(),
	}

	if includeChecks {
		resp.Checks = m.runChecks(ctx)
		for _, check := range resp.Checks {
			if check.Status == StatusUnhealthy {
				resp.Status = StatusUnhealthy
				break
			} else if check.Status == StatusDegraded && resp.Status == StatusHealthy {
				resp.Status = StatusDegraded
			}
		}
	}

	return resp
}

func (m *Manager) Liveness(ctx context.Context) Response {
	return Response{
		Status:    StatusHealthy,
		Timestamp: time.Now(),
		Service:   m.serviceName,
		Version:   m.version,
		Uptime:    time.Since(m.startTime).String(),
	}
}

func (m *Manager) Readiness(ctx context.Context) Response {
	resp := m.Health(ctx, true)
	// Service is not ready if any critical check fails
	if resp.Status == StatusUnhealthy {
		resp.Status = StatusUnhealthy
	}
	return resp
}

func (m *Manager) Detailed(ctx context.Context) Response {
	return m.Health(ctx, true)
}

func (m *Manager) runChecks(ctx context.Context) map[string]CheckResult {
	m.mu.RLock()
	checkers := make(map[string]Checker, len(m.checkers))
	for name, checker := range m.checkers {
		checkers[name] = checker
	}
	m.mu.RUnlock()

	results := make(map[string]CheckResult)
	now := time.Now()

	for name, checker := range checkers {
		// Check cache first
		m.mu.RLock()
		if cached, ok := m.cache[name]; ok && now.Sub(cached.timestamp) < m.cacheTTL {
			results[name] = cached.result
			m.mu.RUnlock()
			continue
		}
		m.mu.RUnlock()

		// Run check with timeout
		checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		result := checker.Check(checkCtx)
		cancel()

		// Cache the result
		m.mu.Lock()
		m.cache[name] = cachedResult{
			result:    result,
			timestamp: now,
		}
		m.mu.Unlock()

		results[name] = result
	}

	return results
}

func (m *Manager) HTTPStatus(status Status) int {
	switch status {
	case StatusHealthy:
		return http.StatusOK
	case StatusDegraded:
		return http.StatusOK // Still accepting traffic but with issues
	case StatusUnhealthy:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
