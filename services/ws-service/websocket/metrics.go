package websocket

import (
	"sync"
	"sync/atomic"
	"time"
)

type Metrics struct {
	prefix string
	mu     sync.RWMutex

	totalConnections      atomic.Int64
	activeConnections     atomic.Int64
	totalMessages         atomic.Int64
	messagesPerSecond     atomic.Int64
	totalErrors           atomic.Int64
	rateLimitedRequests   atomic.Int64
	totalSubscriptions    atomic.Int64
	activeSubscriptions   atomic.Int64
	totalBroadcasts       atomic.Int64
	broadcastsPerSecond   atomic.Int64
	averageLatencyNs      atomic.Int64
	peakConnections       atomic.Int64
	totalBytesReceived    atomic.Int64
	totalBytesSent        atomic.Int64
	reconnections         atomic.Int64
	authFailures          atomic.Int64
	messagesByType        sync.Map
	errorsByCode          sync.Map
	connectionsByPlatform sync.Map

	startTime        time.Time
	lastResetTime    time.Time
	windowMessages   atomic.Int64
	windowBroadcasts atomic.Int64
}

func NewMetrics(prefix string) *Metrics {
	m := &Metrics{
		prefix:        prefix,
		startTime:     time.Now(),
		lastResetTime: time.Now(),
	}

	go m.runWindowReset()
	return m
}

func (m *Metrics) runWindowReset() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		m.messagesPerSecond.Store(m.windowMessages.Swap(0))
		m.broadcastsPerSecond.Store(m.windowBroadcasts.Swap(0))
	}
}

func (m *Metrics) OnConnect(platform string) {
	m.totalConnections.Add(1)
	current := m.activeConnections.Add(1)

	for {
		peak := m.peakConnections.Load()
		if current <= peak {
			break
		}
		if m.peakConnections.CompareAndSwap(peak, current) {
			break
		}
	}

	if platform != "" {
		m.incrementMapCounter(&m.connectionsByPlatform, platform)
	}
}

func (m *Metrics) OnDisconnect(platform string) {
	m.activeConnections.Add(-1)

	if platform != "" {
		m.decrementMapCounter(&m.connectionsByPlatform, platform)
	}
}

func (m *Metrics) OnReconnect() {
	m.reconnections.Add(1)
}

func (m *Metrics) OnMessage(msgType string, sizeBytes int64) {
	m.totalMessages.Add(1)
	m.windowMessages.Add(1)
	m.totalBytesReceived.Add(sizeBytes)

	if msgType != "" {
		m.incrementMapCounter(&m.messagesByType, msgType)
	}
}

func (m *Metrics) OnMessageSent(sizeBytes int64) {
	m.totalBytesSent.Add(sizeBytes)
}

func (m *Metrics) OnError(errorCode string) {
	m.totalErrors.Add(1)

	if errorCode != "" {
		m.incrementMapCounter(&m.errorsByCode, errorCode)
	}
}

func (m *Metrics) OnRateLimited() {
	m.rateLimitedRequests.Add(1)
}

func (m *Metrics) OnAuthFailure() {
	m.authFailures.Add(1)
}

func (m *Metrics) OnSubscribe() {
	m.totalSubscriptions.Add(1)
	m.activeSubscriptions.Add(1)
}

func (m *Metrics) OnUnsubscribe() {
	m.activeSubscriptions.Add(-1)
}

func (m *Metrics) OnBroadcast() {
	m.totalBroadcasts.Add(1)
	m.windowBroadcasts.Add(1)
}

func (m *Metrics) RecordLatency(latencyNs int64) {
	for {
		current := m.averageLatencyNs.Load()
		total := m.totalMessages.Load()
		if total == 0 {
			if m.averageLatencyNs.CompareAndSwap(current, latencyNs) {
				break
			}
			continue
		}
		newAvg := (current*(total-1) + latencyNs) / total
		if m.averageLatencyNs.CompareAndSwap(current, newAvg) {
			break
		}
	}
}

func (m *Metrics) GetSnapshot() MetricsSnapshot {
	uptime := time.Since(m.startTime)

	return MetricsSnapshot{
		Prefix:                m.prefix,
		Uptime:                uptime,
		UptimeSeconds:         int64(uptime.Seconds()),
		TotalConnections:      m.totalConnections.Load(),
		ActiveConnections:     m.activeConnections.Load(),
		PeakConnections:       m.peakConnections.Load(),
		TotalMessages:         m.totalMessages.Load(),
		MessagesPerSecond:     m.messagesPerSecond.Load(),
		TotalErrors:           m.totalErrors.Load(),
		RateLimitedRequests:   m.rateLimitedRequests.Load(),
		TotalSubscriptions:    m.totalSubscriptions.Load(),
		ActiveSubscriptions:   m.activeSubscriptions.Load(),
		TotalBroadcasts:       m.totalBroadcasts.Load(),
		BroadcastsPerSecond:   m.broadcastsPerSecond.Load(),
		AverageLatencyMs:      float64(m.averageLatencyNs.Load()) / float64(time.Millisecond),
		TotalBytesReceived:    m.totalBytesReceived.Load(),
		TotalBytesSent:        m.totalBytesSent.Load(),
		Reconnections:         m.reconnections.Load(),
		AuthFailures:          m.authFailures.Load(),
		MessagesByType:        m.getMapSnapshot(&m.messagesByType),
		ErrorsByCode:          m.getMapSnapshot(&m.errorsByCode),
		ConnectionsByPlatform: m.getMapSnapshot(&m.connectionsByPlatform),
	}
}

func (m *Metrics) Reset() {
	m.totalConnections.Store(0)
	m.totalMessages.Store(0)
	m.totalErrors.Store(0)
	m.rateLimitedRequests.Store(0)
	m.totalSubscriptions.Store(0)
	m.totalBroadcasts.Store(0)
	m.totalBytesReceived.Store(0)
	m.totalBytesSent.Store(0)
	m.reconnections.Store(0)
	m.authFailures.Store(0)
	m.averageLatencyNs.Store(0)

	m.messagesByType = sync.Map{}
	m.errorsByCode = sync.Map{}

	m.lastResetTime = time.Now()
}

func (m *Metrics) incrementMapCounter(mp *sync.Map, key string) {
	for {
		val, _ := mp.LoadOrStore(key, new(atomic.Int64))
		counter := val.(*atomic.Int64)
		counter.Add(1)
		return
	}
}

func (m *Metrics) decrementMapCounter(mp *sync.Map, key string) {
	if val, ok := mp.Load(key); ok {
		counter := val.(*atomic.Int64)
		counter.Add(-1)
	}
}

func (m *Metrics) getMapSnapshot(mp *sync.Map) map[string]int64 {
	result := make(map[string]int64)
	mp.Range(func(key, value any) bool {
		k := key.(string)
		v := value.(*atomic.Int64)
		result[k] = v.Load()
		return true
	})
	return result
}

type MetricsSnapshot struct {
	Prefix                string           `json:"prefix"`
	Uptime                time.Duration    `json:"-"`
	UptimeSeconds         int64            `json:"uptime_seconds"`
	TotalConnections      int64            `json:"total_connections"`
	ActiveConnections     int64            `json:"active_connections"`
	PeakConnections       int64            `json:"peak_connections"`
	TotalMessages         int64            `json:"total_messages"`
	MessagesPerSecond     int64            `json:"messages_per_second"`
	TotalErrors           int64            `json:"total_errors"`
	RateLimitedRequests   int64            `json:"rate_limited_requests"`
	TotalSubscriptions    int64            `json:"total_subscriptions"`
	ActiveSubscriptions   int64            `json:"active_subscriptions"`
	TotalBroadcasts       int64            `json:"total_broadcasts"`
	BroadcastsPerSecond   int64            `json:"broadcasts_per_second"`
	AverageLatencyMs      float64          `json:"average_latency_ms"`
	TotalBytesReceived    int64            `json:"total_bytes_received"`
	TotalBytesSent        int64            `json:"total_bytes_sent"`
	Reconnections         int64            `json:"reconnections"`
	AuthFailures          int64            `json:"auth_failures"`
	MessagesByType        map[string]int64 `json:"messages_by_type"`
	ErrorsByCode          map[string]int64 `json:"errors_by_code"`
	ConnectionsByPlatform map[string]int64 `json:"connections_by_platform"`
}
