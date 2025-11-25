package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

// Collector collects WebSocket metrics
type Collector struct {
	// Connection metrics
	totalConnections     atomic.Int64
	activeConnections    atomic.Int64
	totalDisconnections  atomic.Int64
	failedConnections    atomic.Int64

	// Message metrics
	messagesSent        atomic.Int64
	messagesReceived    atomic.Int64
	messagesSentFailed  atomic.Int64
	messagesDropped     atomic.Int64

	// Byte metrics
	bytesSent           atomic.Int64
	bytesReceived       atomic.Int64

	// Error metrics
	totalErrors         atomic.Int64
	errorsByType        map[string]*atomic.Int64
	errorsMu            sync.RWMutex

	// Latency tracking
	latencies           []time.Duration
	latenciesMu         sync.RWMutex
	maxLatencySamples   int

	// Start time
	startTime           time.Time
}

// NewCollector creates a new metrics collector
func NewCollector(maxLatencySamples int) *Collector {
	return &Collector{
		errorsByType:      make(map[string]*atomic.Int64),
		latencies:         make([]time.Duration, 0, maxLatencySamples),
		maxLatencySamples: maxLatencySamples,
		startTime:         time.Now(),
	}
}

// IncrementConnections increments connection counter
func (c *Collector) IncrementConnections() {
	c.totalConnections.Add(1)
	c.activeConnections.Add(1)
}

// DecrementConnections decrements active connections
func (c *Collector) DecrementConnections() {
	c.activeConnections.Add(-1)
	c.totalDisconnections.Add(1)
}

// IncrementFailedConnections increments failed connection counter
func (c *Collector) IncrementFailedConnections() {
	c.failedConnections.Add(1)
}

// IncrementMessagesSent increments sent messages counter
func (c *Collector) IncrementMessagesSent() {
	c.messagesSent.Add(1)
}

// IncrementMessagesReceived increments received messages counter
func (c *Collector) IncrementMessagesReceived() {
	c.messagesReceived.Add(1)
}

// IncrementMessagesSentFailed increments failed sent messages counter
func (c *Collector) IncrementMessagesSentFailed() {
	c.messagesSentFailed.Add(1)
}

// IncrementMessagesDropped increments dropped messages counter
func (c *Collector) IncrementMessagesDropped() {
	c.messagesDropped.Add(1)
}

// AddBytesSent adds to bytes sent counter
func (c *Collector) AddBytesSent(bytes int64) {
	c.bytesSent.Add(bytes)
}

// AddBytesReceived adds to bytes received counter
func (c *Collector) AddBytesReceived(bytes int64) {
	c.bytesReceived.Add(bytes)
}

// IncrementErrors increments error counter
func (c *Collector) IncrementErrors(errorType string) {
	c.totalErrors.Add(1)

	c.errorsMu.Lock()
	defer c.errorsMu.Unlock()

	if counter, exists := c.errorsByType[errorType]; exists {
		counter.Add(1)
	} else {
		counter := &atomic.Int64{}
		counter.Store(1)
		c.errorsByType[errorType] = counter
	}
}

// RecordLatency records message latency
func (c *Collector) RecordLatency(latency time.Duration) {
	c.latenciesMu.Lock()
	defer c.latenciesMu.Unlock()

	if len(c.latencies) >= c.maxLatencySamples {
		// Remove oldest
		c.latencies = c.latencies[1:]
	}

	c.latencies = append(c.latencies, latency)
}

// GetSnapshot returns a metrics snapshot
func (c *Collector) GetSnapshot() *Snapshot {
	c.errorsMu.RLock()
	errorsByType := make(map[string]int64)
	for errType, counter := range c.errorsByType {
		errorsByType[errType] = counter.Load()
	}
	c.errorsMu.RUnlock()

	c.latenciesMu.RLock()
	avgLatency, p50, p95, p99 := c.calculateLatencyStats()
	c.latenciesMu.RUnlock()

	return &Snapshot{
		TotalConnections:     c.totalConnections.Load(),
		ActiveConnections:    c.activeConnections.Load(),
		TotalDisconnections:  c.totalDisconnections.Load(),
		FailedConnections:    c.failedConnections.Load(),
		MessagesSent:         c.messagesSent.Load(),
		MessagesReceived:     c.messagesReceived.Load(),
		MessagesSentFailed:   c.messagesSentFailed.Load(),
		MessagesDropped:      c.messagesDropped.Load(),
		BytesSent:            c.bytesSent.Load(),
		BytesReceived:        c.bytesReceived.Load(),
		TotalErrors:          c.totalErrors.Load(),
		ErrorsByType:         errorsByType,
		AvgLatency:           avgLatency,
		P50Latency:           p50,
		P95Latency:           p95,
		P99Latency:           p99,
		Uptime:               time.Since(c.startTime),
	}
}

// calculateLatencyStats calculates latency statistics
func (c *Collector) calculateLatencyStats() (avg, p50, p95, p99 time.Duration) {
	if len(c.latencies) == 0 {
		return
	}

	// Calculate average
	var sum time.Duration
	for _, lat := range c.latencies {
		sum += lat
	}
	avg = sum / time.Duration(len(c.latencies))

	// Calculate percentiles (simplified)
	sorted := make([]time.Duration, len(c.latencies))
	copy(sorted, c.latencies)

	if len(sorted) > 0 {
		p50 = sorted[len(sorted)*50/100]
		p95 = sorted[len(sorted)*95/100]
		p99 = sorted[len(sorted)*99/100]
	}

	return
}

// Reset resets all metrics
func (c *Collector) Reset() {
	c.totalConnections.Store(0)
	c.activeConnections.Store(0)
	c.totalDisconnections.Store(0)
	c.failedConnections.Store(0)
	c.messagesSent.Store(0)
	c.messagesReceived.Store(0)
	c.messagesSentFailed.Store(0)
	c.messagesDropped.Store(0)
	c.bytesSent.Store(0)
	c.bytesReceived.Store(0)
	c.totalErrors.Store(0)

	c.errorsMu.Lock()
	c.errorsByType = make(map[string]*atomic.Int64)
	c.errorsMu.Unlock()

	c.latenciesMu.Lock()
	c.latencies = make([]time.Duration, 0, c.maxLatencySamples)
	c.latenciesMu.Unlock()

	c.startTime = time.Now()
}
