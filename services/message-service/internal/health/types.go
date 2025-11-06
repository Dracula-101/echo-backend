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
}

type CheckResult struct {
	Status       Status                 `json:"status"`
	Message      string                 `json:"message,omitempty"`
	ResponseTime float64                `json:"response_time_ms,omitempty"`
	LastChecked  string                 `json:"last_checked,omitempty"`
	Error        string                 `json:"error,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"`
}

type DatabaseDetails struct {
	Connected       bool   `json:"connected"`
	OpenConnections int    `json:"open_connections"`
	IdleConnections int    `json:"idle_connections"`
	MaxConnections  int    `json:"max_connections"`
	WaitCount       int64  `json:"wait_count"`
	WaitDuration    string `json:"wait_duration"`
	MaxIdleTime     string `json:"max_idle_time"`
	MaxLifetime     string `json:"max_lifetime"`
}

type CacheDetails struct {
	Connected        bool   `json:"connected"`
	PoolSize         int    `json:"pool_size"`
	IdleConns        int    `json:"idle_conns"`
	StaleConns       int    `json:"stale_conns"`
	TotalConns       int    `json:"total_conns"`
	HitRate          string `json:"hit_rate,omitempty"`
	UsedMemory       string `json:"used_memory,omitempty"`
	MaxMemory        string `json:"max_memory,omitempty"`
	EvictedKeys      string `json:"evicted_keys,omitempty"`
	ExpiredKeys      string `json:"expired_keys,omitempty"`
	ConnectedClients string `json:"connected_clients,omitempty"`
}

type KafkaDetails struct {
	Connected bool   `json:"connected"`
	Brokers   int    `json:"brokers"`
	Topics    int    `json:"topics,omitempty"`
	Message   string `json:"message,omitempty"`
}

type WebSocketDetails struct {
	ActiveConnections int    `json:"active_connections"`
	MaxConnections    int    `json:"max_connections"`
	TotalClients      int    `json:"total_clients"`
	Message           string `json:"message,omitempty"`
}
