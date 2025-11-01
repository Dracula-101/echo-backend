package checkers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"echo-backend/services/api-gateway/internal/health"
)

type ServiceChecker struct {
	name       string
	healthURL  string
	httpClient *http.Client
	timeout    time.Duration
}

func NewServiceChecker(name, healthURL string, timeout time.Duration) *ServiceChecker {
	return &ServiceChecker{
		name:      name,
		healthURL: healthURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

func (c *ServiceChecker) Name() string {
	return c.name
}

func (c *ServiceChecker) Check(ctx context.Context) health.CheckResult {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.healthURL, nil)
	if err != nil {
		return health.CheckResult{
			Status:       health.StatusUnhealthy,
			Message:      fmt.Sprintf("Failed to create request for %s", c.name),
			Error:        err.Error(),
			ResponseTime: time.Since(start).Seconds() * 1000,
			LastChecked:  time.Now().Format(time.RFC3339),
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return health.CheckResult{
			Status:       health.StatusUnhealthy,
			Message:      fmt.Sprintf("Service %s unreachable", c.name),
			Error:        err.Error(),
			ResponseTime: time.Since(start).Seconds() * 1000,
			LastChecked:  time.Now().Format(time.RFC3339),
		}
	}
	defer resp.Body.Close()

	responseTime := time.Since(start).Seconds() * 1000

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return health.CheckResult{
			Status:       health.StatusHealthy,
			Message:      fmt.Sprintf("Service %s is healthy", c.name),
			ResponseTime: responseTime,
			LastChecked:  time.Now().Format(time.RFC3339),
		}
	} else if resp.StatusCode >= 500 {
		return health.CheckResult{
			Status:       health.StatusUnhealthy,
			Message:      fmt.Sprintf("Service %s returned %d", c.name, resp.StatusCode),
			ResponseTime: responseTime,
			LastChecked:  time.Now().Format(time.RFC3339),
		}
	}

	return health.CheckResult{
		Status:       health.StatusDegraded,
		Message:      fmt.Sprintf("Service %s returned %d", c.name, resp.StatusCode),
		ResponseTime: responseTime,
		LastChecked:  time.Now().Format(time.RFC3339),
	}
}
