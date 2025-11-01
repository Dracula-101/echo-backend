package health

import "time"

type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

type Response struct {
	Status    Status                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Service   string                 `json:"service"`
	Version   string                 `json:"version"`
	Uptime    string                 `json:"uptime"`
	Checks    map[string]CheckResult `json:"checks,omitempty"`
	Services  map[string]CheckResult `json:"services,omitempty"`
	System    *SystemMetrics         `json:"system,omitempty"`
}

type CheckResult struct {
	Status       Status  `json:"status"`
	Message      string  `json:"message,omitempty"`
	ResponseTime float64 `json:"response_time_ms,omitempty"`
	LastChecked  string  `json:"last_checked,omitempty"`
	Error        string  `json:"error,omitempty"`
}

type SystemMetrics struct {
	MemoryUsedMB   float64 `json:"memory_used_mb"`
	GoroutineCount int     `json:"goroutine_count"`
	CPUCores       int     `json:"cpu_cores"`
}
