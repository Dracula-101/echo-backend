package health

import (
	"context"
	"net/http"
	"runtime"
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

func (m *Manager) HealthWithServices(ctx context.Context, services map[string]CheckResult) Response {
	resp := m.Health(ctx, true)
	resp.Services = services

	for _, svc := range services {
		if svc.Status == StatusUnhealthy {
			resp.Status = StatusDegraded
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
	if resp.Status == StatusUnhealthy {
		resp.Status = StatusUnhealthy
	}
	return resp
}

func (m *Manager) Detailed(ctx context.Context) Response {
	resp := m.Health(ctx, true)

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	resp.System = &SystemMetrics{
		MemoryUsedMB:   float64(mem.Alloc) / 1024 / 1024,
		GoroutineCount: runtime.NumGoroutine(),
		CPUCores:       runtime.NumCPU(),
	}

	return resp
}

func (m *Manager) runChecks(ctx context.Context) map[string]CheckResult {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make(map[string]CheckResult)
	now := time.Now()

	for name, checker := range m.checkers {
		if cached, ok := m.cache[name]; ok && now.Sub(cached.timestamp) < m.cacheTTL {
			results[name] = cached.result
			continue
		}

		checkCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		result := checker.Check(checkCtx)
		cancel()

		m.cache[name] = cachedResult{
			result:    result,
			timestamp: now,
		}
		results[name] = result
	}

	return results
}

func (m *Manager) HTTPStatus(status Status) int {
	switch status {
	case StatusHealthy:
		return http.StatusOK
	case StatusDegraded:
		return http.StatusOK
	case StatusUnhealthy:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
