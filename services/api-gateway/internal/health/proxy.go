package health

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"echo-backend/services/api-gateway/internal/config"
)

type ProxyHealthChecker struct {
	services   map[string]config.ServiceConfig
	httpClient *http.Client
	timeout    time.Duration
}

func NewProxyHealthChecker(services map[string]config.ServiceConfig, timeout time.Duration) *ProxyHealthChecker {
	return &ProxyHealthChecker{
		services: services,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

func (p *ProxyHealthChecker) CheckServices(ctx context.Context) map[string]CheckResult {
	results := make(map[string]CheckResult)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for name, svc := range p.services {
		wg.Add(1)
		go func(serviceName string, service config.ServiceConfig) {
			defer wg.Done()

			result := p.checkService(ctx, serviceName, service)
			mu.Lock()
			results[serviceName] = result
			mu.Unlock()
		}(name, svc)
	}

	wg.Wait()
	return results
}

func (p *ProxyHealthChecker) checkService(ctx context.Context, name string, svc config.ServiceConfig) CheckResult {
	if len(svc.Addresses) == 0 {
		return CheckResult{
			Status:      StatusUnhealthy,
			Message:     "No addresses configured",
			LastChecked: time.Now().Format(time.RFC3339),
		}
	}

	healthURL := fmt.Sprintf("%s/health", svc.Addresses[0])
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return CheckResult{
			Status:       StatusUnhealthy,
			Message:      "Failed to create health check request",
			Error:        err.Error(),
			ResponseTime: time.Since(start).Seconds() * 1000,
			LastChecked:  time.Now().Format(time.RFC3339),
		}
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return CheckResult{
			Status:       StatusUnhealthy,
			Message:      "Service unreachable",
			Error:        err.Error(),
			ResponseTime: time.Since(start).Seconds() * 1000,
			LastChecked:  time.Now().Format(time.RFC3339),
		}
	}
	defer resp.Body.Close()

	responseTime := time.Since(start).Seconds() * 1000

	if resp.StatusCode == http.StatusOK {
		return CheckResult{
			Status:       StatusHealthy,
			Message:      "Service is healthy",
			ResponseTime: responseTime,
			LastChecked:  time.Now().Format(time.RFC3339),
		}
	}

	return CheckResult{
		Status:       StatusUnhealthy,
		Message:      fmt.Sprintf("Service returned status %d", resp.StatusCode),
		ResponseTime: responseTime,
		LastChecked:  time.Now().Format(time.RFC3339),
	}
}
