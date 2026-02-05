package adapter

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"shared/pkg/logger"
)

// LogAggregator aggregates and processes log entries
type LogAggregator struct {
	mu             sync.RWMutex
	buffer         []AggregatedLogEntry
	maxBufferSize  int
	flushInterval  time.Duration
	ticker         *time.Ticker
	done           chan bool
	processors     []LogProcessor
	hooks          []AggregationHook
	stats          *AggregationStats
	metricsTracker *MetricsTracker
}

// AggregatedLogEntry represents an aggregated log entry
type AggregatedLogEntry struct {
	Timestamp   time.Time
	Level       string
	Message     string
	Fields      map[string]interface{}
	Source      string
	Hostname    string
	ServiceName string
	Count       int
	FirstSeen   time.Time
	LastSeen    time.Time
	Fingerprint string
}

// LogProcessor processes log entries
type LogProcessor interface {
	Process(entry *AggregatedLogEntry) error
	Name() string
}

// AggregationHook is called during aggregation lifecycle
type AggregationHook func(phase string, entry *AggregatedLogEntry)

// AggregationStats tracks aggregation statistics
type AggregationStats struct {
	mu                sync.RWMutex
	totalProcessed    int64
	totalDropped      int64
	totalAggregated   int64
	totalFlushed      int64
	lastFlushTime     time.Time
	processingErrors  int64
	averageFlushTime  time.Duration
	bufferUtilization float64
}

// AggregationStatsSnapshot is a safe, lock-free copy of AggregationStats data
type AggregationStatsSnapshot struct {
	TotalProcessed    int64
	TotalDropped      int64
	TotalAggregated   int64
	TotalFlushed      int64
	LastFlushTime     time.Time
	ProcessingErrors  int64
	AverageFlushTime  time.Duration
	BufferUtilization float64
}

// NewLogAggregator creates a new log aggregator
func NewLogAggregator(maxBufferSize int, flushInterval time.Duration) *LogAggregator {
	la := &LogAggregator{
		buffer:         make([]AggregatedLogEntry, 0, maxBufferSize),
		maxBufferSize:  maxBufferSize,
		flushInterval:  flushInterval,
		ticker:         time.NewTicker(flushInterval),
		done:           make(chan bool),
		processors:     make([]LogProcessor, 0),
		hooks:          make([]AggregationHook, 0),
		stats:          &AggregationStats{},
		metricsTracker: NewMetricsTracker(),
	}

	go la.autoFlush()
	return la
}

// AddProcessor adds a log processor
func (la *LogAggregator) AddProcessor(processor LogProcessor) {
	la.mu.Lock()
	defer la.mu.Unlock()
	la.processors = append(la.processors, processor)
}

// AddHook adds an aggregation hook
func (la *LogAggregator) AddHook(hook AggregationHook) {
	la.mu.Lock()
	defer la.mu.Unlock()
	la.hooks = append(la.hooks, hook)
}

// Aggregate aggregates a log entry
func (la *LogAggregator) Aggregate(level, message string, fields map[string]interface{}) error {
	entry := AggregatedLogEntry{
		Timestamp:   time.Now(),
		Level:       level,
		Message:     message,
		Fields:      fields,
		FirstSeen:   time.Now(),
		LastSeen:    time.Now(),
		Count:       1,
		Fingerprint: la.generateFingerprint(level, message, fields),
	}

	// Extract metadata
	if source, ok := fields["source"].(string); ok {
		entry.Source = source
	}
	if hostname, ok := fields["hostname"].(string); ok {
		entry.Hostname = hostname
	}
	if service, ok := fields["service"].(string); ok {
		entry.ServiceName = service
	}

	la.mu.Lock()
	defer la.mu.Unlock()

	// Check if we can merge with existing entry
	merged := false
	for i := range la.buffer {
		if la.buffer[i].Fingerprint == entry.Fingerprint {
			la.buffer[i].Count++
			la.buffer[i].LastSeen = time.Now()
			merged = true
			la.stats.totalAggregated++
			break
		}
	}

	if !merged {
		if len(la.buffer) >= la.maxBufferSize {
			// Buffer is full, flush
			la.flushLocked()
		}
		la.buffer = append(la.buffer, entry)
	}

	la.stats.totalProcessed++
	la.callHooks("aggregate", &entry)

	return nil
}

// Flush flushes the buffer
func (la *LogAggregator) Flush() {
	la.mu.Lock()
	defer la.mu.Unlock()
	la.flushLocked()
}

// flushLocked flushes the buffer (must be called with lock held)
func (la *LogAggregator) flushLocked() {
	if len(la.buffer) == 0 {
		return
	}

	startTime := time.Now()

	// Process entries
	for i := range la.buffer {
		entry := &la.buffer[i]
		la.callHooks("flush", entry)

		for _, processor := range la.processors {
			if err := processor.Process(entry); err != nil {
				la.stats.processingErrors++
			}
		}
	}

	flushDuration := time.Since(startTime)

	// Update stats
	la.stats.totalFlushed += int64(len(la.buffer))
	la.stats.lastFlushTime = time.Now()
	la.stats.averageFlushTime = (la.stats.averageFlushTime + flushDuration) / 2
	la.stats.bufferUtilization = float64(len(la.buffer)) / float64(la.maxBufferSize) * 100

	// Clear buffer
	la.buffer = la.buffer[:0]
}

// autoFlush periodically flushes the buffer
func (la *LogAggregator) autoFlush() {
	for {
		select {
		case <-la.ticker.C:
			la.Flush()
		case <-la.done:
			la.ticker.Stop()
			la.Flush()
			return
		}
	}
}

// Close closes the aggregator
func (la *LogAggregator) Close() {
	la.done <- true
}

// GetStats returns aggregation statistics

// GetStats returns aggregation statistics snapshot
func (la *LogAggregator) GetStats() AggregationStatsSnapshot {
	la.stats.mu.RLock()
	defer la.stats.mu.RUnlock()
	return AggregationStatsSnapshot{
		TotalProcessed:    la.stats.totalProcessed,
		TotalDropped:      la.stats.totalDropped,
		TotalAggregated:   la.stats.totalAggregated,
		TotalFlushed:      la.stats.totalFlushed,
		LastFlushTime:     la.stats.lastFlushTime,
		ProcessingErrors:  la.stats.processingErrors,
		AverageFlushTime:  la.stats.averageFlushTime,
		BufferUtilization: la.stats.bufferUtilization,
	}
}

// generateFingerprint generates a fingerprint for deduplication
func (la *LogAggregator) generateFingerprint(level, message string, fields map[string]interface{}) string {
	// Simple fingerprint based on level and message
	return fmt.Sprintf("%s:%s", level, message)
}

// callHooks calls all hooks
func (la *LogAggregator) callHooks(phase string, entry *AggregatedLogEntry) {
	for _, hook := range la.hooks {
		hook(phase, entry)
	}
}

// ConsoleProcessor writes logs to console
type ConsoleProcessor struct {
	formatter *AdvancedFormatter
}

// NewConsoleProcessor creates a new console processor
func NewConsoleProcessor() *ConsoleProcessor {
	return &ConsoleProcessor{
		formatter: NewAdvancedFormatter(DefaultFormatterConfig()),
	}
}

// Process processes a log entry
func (cp *ConsoleProcessor) Process(entry *AggregatedLogEntry) error {
	formatted := cp.formatter.Format(entry)
	fmt.Println(formatted)
	return nil
}

// Name returns the processor name
func (cp *ConsoleProcessor) Name() string {
	return "console"
}

// FileProcessor writes logs to a file
type FileProcessor struct {
	filename string
	mu       sync.Mutex
}

// NewFileProcessor creates a new file processor
func NewFileProcessor(filename string) *FileProcessor {
	return &FileProcessor{
		filename: filename,
	}
}

// Process processes a log entry
func (fp *FileProcessor) Process(entry *AggregatedLogEntry) error {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	// In a real implementation, this would write to a file
	// For now, just a placeholder
	return nil
}

// Name returns the processor name
func (fp *FileProcessor) Name() string {
	return "file"
}

// MetricsTracker tracks metrics for log aggregation
type MetricsTracker struct {
	mu      sync.RWMutex
	metrics map[string]*Metric
}

// Metric represents a metric
type Metric struct {
	Name       string
	Value      float64
	Count      int64
	Min        float64
	Max        float64
	Sum        float64
	LastUpdate time.Time
	Tags       map[string]string
}

// NewMetricsTracker creates a new metrics tracker
func NewMetricsTracker() *MetricsTracker {
	return &MetricsTracker{
		metrics: make(map[string]*Metric),
	}
}

// RecordMetric records a metric
func (mt *MetricsTracker) RecordMetric(name string, value float64, tags map[string]string) {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	key := mt.generateKey(name, tags)

	metric, exists := mt.metrics[key]
	if !exists {
		metric = &Metric{
			Name:       name,
			Value:      value,
			Count:      1,
			Min:        value,
			Max:        value,
			Sum:        value,
			LastUpdate: time.Now(),
			Tags:       tags,
		}
		mt.metrics[key] = metric
	} else {
		metric.Value = value
		metric.Count++
		metric.Sum += value

		if value < metric.Min {
			metric.Min = value
		}
		if value > metric.Max {
			metric.Max = value
		}

		metric.LastUpdate = time.Now()
	}
}

// GetMetric gets a metric
func (mt *MetricsTracker) GetMetric(name string, tags map[string]string) *Metric {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	key := mt.generateKey(name, tags)
	return mt.metrics[key]
}

// GetAllMetrics gets all metrics
func (mt *MetricsTracker) GetAllMetrics() []*Metric {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	metrics := make([]*Metric, 0, len(mt.metrics))
	for _, metric := range mt.metrics {
		metrics = append(metrics, metric)
	}
	return metrics
}

// generateKey generates a key for a metric
func (mt *MetricsTracker) generateKey(name string, tags map[string]string) string {
	key := name
	if tags != nil {
		data, _ := json.Marshal(tags)
		key = fmt.Sprintf("%s:%s", name, string(data))
	}
	return key
}

// LogBuffer provides a circular buffer for log entries
type LogBuffer struct {
	mu       sync.RWMutex
	entries  []logger.Field
	size     int
	head     int
	tail     int
	count    int
	overflow int64
}

// NewLogBuffer creates a new log buffer
func NewLogBuffer(size int) *LogBuffer {
	return &LogBuffer{
		entries: make([]logger.Field, size),
		size:    size,
	}
}

// Add adds an entry to the buffer
func (lb *LogBuffer) Add(entry logger.Field) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	lb.entries[lb.tail] = entry
	lb.tail = (lb.tail + 1) % lb.size

	if lb.count < lb.size {
		lb.count++
	} else {
		lb.head = (lb.head + 1) % lb.size
		lb.overflow++
	}
}

// Get gets all entries from the buffer
func (lb *LogBuffer) Get() []logger.Field {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	result := make([]logger.Field, lb.count)

	if lb.count == 0 {
		return result
	}

	idx := lb.head
	for i := 0; i < lb.count; i++ {
		result[i] = lb.entries[idx]
		idx = (idx + 1) % lb.size
	}

	return result
}

// Clear clears the buffer
func (lb *LogBuffer) Clear() {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	lb.head = 0
	lb.tail = 0
	lb.count = 0
	lb.overflow = 0
}

// Count returns the number of entries in the buffer
func (lb *LogBuffer) Count() int {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	return lb.count
}

// Overflow returns the number of overflowed entries
func (lb *LogBuffer) Overflow() int64 {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	return lb.overflow
}

// RateLimiter limits the rate of log entries
type RateLimiter struct {
	mu         sync.Mutex
	rate       int
	burst      int
	tokens     int
	lastRefill time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate, burst int) *RateLimiter {
	return &RateLimiter{
		rate:       rate,
		burst:      burst,
		tokens:     burst,
		lastRefill: time.Now(),
	}
}

// Allow checks if a log entry is allowed
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)

	// Refill tokens based on elapsed time
	tokensToAdd := int(elapsed.Seconds()) * rl.rate
	if tokensToAdd > 0 {
		rl.tokens += tokensToAdd
		if rl.tokens > rl.burst {
			rl.tokens = rl.burst
		}
		rl.lastRefill = now
	}

	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}

// SetRate sets the rate
func (rl *RateLimiter) SetRate(rate int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.rate = rate
}

// SetBurst sets the burst size
func (rl *RateLimiter) SetBurst(burst int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.burst = burst
	if rl.tokens > burst {
		rl.tokens = burst
	}
}

// GetTokens returns the current number of tokens
func (rl *RateLimiter) GetTokens() int {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.tokens
}

// AsyncLogger provides asynchronous logging with buffering
type AsyncLogger struct {
	baseLogger *zapLogger
	buffer     chan logEntry
	workerPool *WorkerPool
	wg         sync.WaitGroup
	done       chan bool
}

type logEntry struct {
	level  string
	msg    string
	fields []logger.Field
}

// NewAsyncLogger creates a new async logger
func NewAsyncLogger(baseLogger *zapLogger, bufferSize int, workers int) *AsyncLogger {
	al := &AsyncLogger{
		baseLogger: baseLogger,
		buffer:     make(chan logEntry, bufferSize),
		workerPool: NewWorkerPool(workers),
		done:       make(chan bool),
	}

	al.start()
	return al
}

// start starts the async logger workers
func (al *AsyncLogger) start() {
	for i := 0; i < al.workerPool.Size(); i++ {
		al.wg.Add(1)
		go al.worker()
	}
}

// worker processes log entries from the buffer
func (al *AsyncLogger) worker() {
	defer al.wg.Done()

	for {
		select {
		case entry := <-al.buffer:
			al.processEntry(entry)
		case <-al.done:
			// Process remaining entries
			for len(al.buffer) > 0 {
				entry := <-al.buffer
				al.processEntry(entry)
			}
			return
		}
	}
}

// processEntry processes a log entry
func (al *AsyncLogger) processEntry(entry logEntry) {
	switch entry.level {
	case "DEBUG":
		al.baseLogger.Debug(entry.msg, entry.fields...)
	case "INFO":
		al.baseLogger.Info(entry.msg, entry.fields...)
	case "WARN":
		al.baseLogger.Warn(entry.msg, entry.fields...)
	case "ERROR":
		al.baseLogger.Error(entry.msg, entry.fields...)
	case "FATAL":
		al.baseLogger.Fatal(entry.msg, entry.fields...)
	}
}

// Log logs a message asynchronously
func (al *AsyncLogger) Log(level, msg string, fields ...logger.Field) {
	entry := logEntry{
		level:  level,
		msg:    msg,
		fields: fields,
	}

	select {
	case al.buffer <- entry:
	default:
		// Buffer is full, log synchronously as fallback
		al.processEntry(entry)
	}
}

// Close closes the async logger
func (al *AsyncLogger) Close() {
	close(al.done)
	al.wg.Wait()
	close(al.buffer)
}

// WorkerPool manages a pool of workers
type WorkerPool struct {
	size int
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(size int) *WorkerPool {
	return &WorkerPool{
		size: size,
	}
}

// Size returns the pool size
func (wp *WorkerPool) Size() int {
	return wp.size
}

// LogRotator handles log rotation
type LogRotator struct {
	mu            sync.Mutex
	filename      string
	maxSize       int64
	maxBackups    int
	currentSize   int64
	rotationCount int
}

// NewLogRotator creates a new log rotator
func NewLogRotator(filename string, maxSize int64, maxBackups int) *LogRotator {
	return &LogRotator{
		filename:   filename,
		maxSize:    maxSize,
		maxBackups: maxBackups,
	}
}

// ShouldRotate checks if rotation is needed
func (lr *LogRotator) ShouldRotate(size int64) bool {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	lr.currentSize += size
	return lr.currentSize >= lr.maxSize
}

// Rotate performs log rotation
func (lr *LogRotator) Rotate() error {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	lr.rotationCount++
	lr.currentSize = 0

	// In a real implementation, this would rename files
	// and manage backups
	return nil
}

// GetRotationCount returns the rotation count
func (lr *LogRotator) GetRotationCount() int {
	lr.mu.Lock()
	defer lr.mu.Unlock()
	return lr.rotationCount
}

// CompressionManager manages log compression
type CompressionManager struct {
	mu               sync.Mutex
	compressionType  string
	compressionLevel int
	enabled          bool
}

// NewCompressionManager creates a new compression manager
func NewCompressionManager(compressionType string, level int) *CompressionManager {
	return &CompressionManager{
		compressionType:  compressionType,
		compressionLevel: level,
		enabled:          true,
	}
}

// Compress compresses data
func (cm *CompressionManager) Compress(data []byte) ([]byte, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !cm.enabled {
		return data, nil
	}

	// In a real implementation, this would compress the data
	// using the specified algorithm
	return data, nil
}

// Decompress decompresses data
func (cm *CompressionManager) Decompress(data []byte) ([]byte, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// In a real implementation, this would decompress the data
	return data, nil
}

// SetEnabled enables or disables compression
func (cm *CompressionManager) SetEnabled(enabled bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.enabled = enabled
}

// IsEnabled returns whether compression is enabled
func (cm *CompressionManager) IsEnabled() bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.enabled
}

// MarshalJSON custom JSON marshaling for AggregatedLogEntry
func (ale AggregatedLogEntry) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"timestamp":    ale.Timestamp,
		"level":        ale.Level,
		"message":      ale.Message,
		"fields":       ale.Fields,
		"source":       ale.Source,
		"hostname":     ale.Hostname,
		"service_name": ale.ServiceName,
		"count":        ale.Count,
		"first_seen":   ale.FirstSeen,
		"last_seen":    ale.LastSeen,
		"fingerprint":  ale.Fingerprint,
	})
}

// MarshalJSON custom JSON marshaling for Metric
func (m Metric) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"name":        m.Name,
		"value":       m.Value,
		"count":       m.Count,
		"min":         m.Min,
		"max":         m.Max,
		"average":     m.Sum / float64(m.Count),
		"sum":         m.Sum,
		"last_update": m.LastUpdate,
		"tags":        m.Tags,
	})
}
