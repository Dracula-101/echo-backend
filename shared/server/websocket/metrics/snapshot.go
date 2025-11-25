package metrics

import "time"

// Snapshot represents a metrics snapshot
type Snapshot struct {
	// Connection metrics
	TotalConnections     int64 `json:"total_connections"`
	ActiveConnections    int64 `json:"active_connections"`
	TotalDisconnections  int64 `json:"total_disconnections"`
	FailedConnections    int64 `json:"failed_connections"`

	// Message metrics
	MessagesSent        int64 `json:"messages_sent"`
	MessagesReceived    int64 `json:"messages_received"`
	MessagesSentFailed  int64 `json:"messages_sent_failed"`
	MessagesDropped     int64 `json:"messages_dropped"`

	// Byte metrics
	BytesSent           int64 `json:"bytes_sent"`
	BytesReceived       int64 `json:"bytes_received"`

	// Error metrics
	TotalErrors         int64            `json:"total_errors"`
	ErrorsByType        map[string]int64 `json:"errors_by_type"`

	// Latency metrics
	AvgLatency          time.Duration `json:"avg_latency"`
	P50Latency          time.Duration `json:"p50_latency"`
	P95Latency          time.Duration `json:"p95_latency"`
	P99Latency          time.Duration `json:"p99_latency"`

	// System metrics
	Uptime              time.Duration `json:"uptime"`
}

// MessagesPerSecond returns messages per second rate
func (s *Snapshot) MessagesPerSecond() float64 {
	if s.Uptime.Seconds() == 0 {
		return 0
	}
	return float64(s.MessagesSent+s.MessagesReceived) / s.Uptime.Seconds()
}

// BytesPerSecond returns bytes per second rate
func (s *Snapshot) BytesPerSecond() float64 {
	if s.Uptime.Seconds() == 0 {
		return 0
	}
	return float64(s.BytesSent+s.BytesReceived) / s.Uptime.Seconds()
}

// ErrorRate returns error rate percentage
func (s *Snapshot) ErrorRate() float64 {
	total := s.MessagesSent + s.MessagesReceived
	if total == 0 {
		return 0
	}
	return float64(s.TotalErrors) / float64(total) * 100
}
