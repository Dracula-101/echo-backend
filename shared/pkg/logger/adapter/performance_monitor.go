package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"runtime/pprof"
	"sync"
	"sync/atomic"
	"time"
)

// PerformanceMonitor monitors logging performance
type PerformanceMonitor struct {
	mu                 sync.RWMutex
	metrics            *PerformanceMetrics
	samplers           map[string]*PerformanceSampler
	profilers          map[string]*Profiler
	optimizers         []PerformanceOptimizer
	alertThresholds    map[string]float64
	enabled            bool
	samplingRate       float64
	reportingInterval  time.Duration
	ticker             *time.Ticker
	done               chan bool
	callbackRegistry   *CallbackRegistry
	resourceTracker    *ResourceTracker
	latencyHistogram   *LatencyHistogram
	throughputMeter    *ThroughputMeter
	memoryAnalyzer     *MemoryAnalyzer
	cpuProfiler        *CPUProfiler
	blockProfiler      *BlockProfiler
	goroutineAnalyzer  *GoroutineAnalyzer
	allocator          *AllocationTracker
	cachingMetrics     *CachingMetrics
	ioMetrics          *IOMetrics
	concurrencyMetrics *ConcurrencyMetrics
}

// PerformanceMetrics holds performance metrics
type PerformanceMetrics struct {
	mu                    sync.RWMutex
	TotalLogs             int64
	LogsPerSecond         float64
	AverageLatency        time.Duration
	MaxLatency            time.Duration
	MinLatency            time.Duration
	P50Latency            time.Duration
	P95Latency            time.Duration
	P99Latency            time.Duration
	DroppedLogs           int64
	BufferUtilization     float64
	MemoryUsage           uint64
	CPUUsage              float64
	GoroutineCount        int
	AllocationsPerSecond  float64
	GCPauseTime           time.Duration
	LastReportTime        time.Time
	WindowStart           time.Time
	SampleCount           int64
	ErrorRate             float64
	SuccessRate           float64
	AverageProcessingTime time.Duration
	TotalProcessingTime   time.Duration
	CacheHitRate          float64
	CacheMissRate         float64
	QueueDepth            int
	BackpressureEvents    int64
	CircuitBreakerState   string
	HealthScore           float64
}

// PerformanceSampler samples performance data
type PerformanceSampler struct {
	mu           sync.RWMutex
	name         string
	samples      []PerformanceSample
	maxSamples   int
	windowSize   time.Duration
	aggregations map[string]float64
}

// PerformanceSample represents a performance sample
type PerformanceSample struct {
	Timestamp      time.Time
	Latency        time.Duration
	MemoryUsage    uint64
	CPUUsage       float64
	GoroutineCount int
	Success        bool
	ErrorCode      string
	Metadata       map[string]interface{}
}

// PerformanceOptimizer optimizes performance
type PerformanceOptimizer interface {
	Optimize(metrics *PerformanceMetrics) []OptimizationRecommendation
	Name() string
	Priority() int
}

// OptimizationRecommendation represents an optimization recommendation
type OptimizationRecommendation struct {
	Category        string
	Severity        string
	Description     string
	Action          string
	EstimatedImpact float64
	Confidence      float64
	Metadata        map[string]interface{}
}

// Profiler profiles code execution
type Profiler struct {
	mu         sync.RWMutex
	name       string
	startTime  time.Time
	endTime    time.Time
	duration   time.Duration
	samples    []ProfileSample
	callStacks map[string]int
	hotspots   []Hotspot
	enabled    bool
}

// ProfileSample represents a profiling sample
type ProfileSample struct {
	Timestamp  time.Time
	Function   string
	Duration   time.Duration
	MemAlloc   uint64
	LineNumber int
	Filename   string
}

// Hotspot represents a performance hotspot
type Hotspot struct {
	Function     string
	TotalTime    time.Duration
	CallCount    int
	AverageTime  time.Duration
	MemoryImpact uint64
	Severity     string
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor(reportingInterval time.Duration) *PerformanceMonitor {
	pm := &PerformanceMonitor{
		metrics:            NewPerformanceMetrics(),
		samplers:           make(map[string]*PerformanceSampler),
		profilers:          make(map[string]*Profiler),
		optimizers:         make([]PerformanceOptimizer, 0),
		alertThresholds:    make(map[string]float64),
		enabled:            true,
		samplingRate:       1.0,
		reportingInterval:  reportingInterval,
		ticker:             time.NewTicker(reportingInterval),
		done:               make(chan bool),
		callbackRegistry:   NewCallbackRegistry(),
		resourceTracker:    NewResourceTracker(),
		latencyHistogram:   NewLatencyHistogram(),
		throughputMeter:    NewThroughputMeter(),
		memoryAnalyzer:     NewMemoryAnalyzer(),
		cpuProfiler:        NewCPUProfiler(),
		blockProfiler:      NewBlockProfiler(),
		goroutineAnalyzer:  NewGoroutineAnalyzer(),
		allocator:          NewAllocationTracker(),
		cachingMetrics:     NewCachingMetrics(),
		ioMetrics:          NewIOMetrics(),
		concurrencyMetrics: NewConcurrencyMetrics(),
	}

	pm.setDefaultThresholds()
	go pm.reportLoop()
	return pm
}

// NewPerformanceMetrics creates new performance metrics
func NewPerformanceMetrics() *PerformanceMetrics {
	return &PerformanceMetrics{
		WindowStart:    time.Now(),
		LastReportTime: time.Now(),
		HealthScore:    100.0,
	}
}

// setDefaultThresholds sets default alert thresholds
func (pm *PerformanceMonitor) setDefaultThresholds() {
	pm.alertThresholds = map[string]float64{
		"latency_ms":        1000.0,
		"error_rate":        5.0,
		"memory_usage_mb":   1024.0,
		"cpu_usage_percent": 80.0,
		"goroutine_count":   10000.0,
		"dropped_logs":      100.0,
	}
}

// RecordLog records a log event
func (pm *PerformanceMonitor) RecordLog(latency time.Duration, success bool, errorCode string) {
	if !pm.enabled {
		return
	}

	atomic.AddInt64(&pm.metrics.TotalLogs, 1)
	atomic.AddInt64(&pm.metrics.SampleCount, 1)

	if !success {
		atomic.AddInt64(&pm.metrics.DroppedLogs, 1)
	}

	// Update latency metrics
	pm.metrics.mu.Lock()
	if latency > pm.metrics.MaxLatency {
		pm.metrics.MaxLatency = latency
	}
	if pm.metrics.MinLatency == 0 || latency < pm.metrics.MinLatency {
		pm.metrics.MinLatency = latency
	}

	// Update average latency
	totalSamples := atomic.LoadInt64(&pm.metrics.SampleCount)
	if totalSamples > 0 {
		avgNanos := pm.metrics.AverageLatency.Nanoseconds()
		newAvg := (avgNanos*totalSamples + latency.Nanoseconds()) / (totalSamples + 1)
		pm.metrics.AverageLatency = time.Duration(newAvg)
	}
	pm.metrics.mu.Unlock()

	// Record sample
	sample := PerformanceSample{
		Timestamp: time.Now(),
		Latency:   latency,
		Success:   success,
		ErrorCode: errorCode,
		Metadata:  make(map[string]interface{}),
	}

	pm.recordSample("default", sample)
	pm.latencyHistogram.Record(latency)
	pm.throughputMeter.Record()
}

// recordSample records a performance sample
func (pm *PerformanceMonitor) recordSample(samplerName string, sample PerformanceSample) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	sampler, exists := pm.samplers[samplerName]
	if !exists {
		sampler = NewPerformanceSampler(samplerName, 1000, time.Minute*5)
		pm.samplers[samplerName] = sampler
	}

	sampler.AddSample(sample)
}

// UpdateSystemMetrics updates system-level metrics
func (pm *PerformanceMonitor) UpdateSystemMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	pm.metrics.mu.Lock()
	pm.metrics.MemoryUsage = m.Alloc
	pm.metrics.GoroutineCount = runtime.NumGoroutine()
	pm.metrics.GCPauseTime = time.Duration(m.PauseNs[(m.NumGC+255)%256])
	pm.metrics.mu.Unlock()

	// Update component metrics
	pm.memoryAnalyzer.Analyze(&m)
	pm.goroutineAnalyzer.Analyze()
	pm.allocator.Track()
}

// GenerateReport generates a performance report
func (pm *PerformanceMonitor) GenerateReport() *PerformanceReport {
	pm.metrics.mu.RLock()
	defer pm.metrics.mu.RUnlock()

	metricsSnapshot := pm.metrics.snapshot()

	report := &PerformanceReport{
		GeneratedAt:     time.Now(),
		Duration:        time.Since(pm.metrics.WindowStart),
		Metrics:         metricsSnapshot,
		Samples:         make(map[string][]PerformanceSample),
		Hotspots:        make([]Hotspot, 0),
		Recommendations: make([]OptimizationRecommendation, 0),
	}

	// Collect samples from all samplers
	pm.mu.RLock()
	for name, sampler := range pm.samplers {
		report.Samples[name] = sampler.GetSamples()
	}
	pm.mu.RUnlock()

	// Calculate rates
	elapsed := time.Since(pm.metrics.WindowStart).Seconds()
	if elapsed > 0 {
		report.Metrics.LogsPerSecond = float64(pm.metrics.TotalLogs) / elapsed

		totalRequests := pm.metrics.TotalLogs
		errors := pm.metrics.DroppedLogs
		if totalRequests > 0 {
			report.Metrics.ErrorRate = float64(errors) / float64(totalRequests) * 100
			report.Metrics.SuccessRate = 100 - report.Metrics.ErrorRate
		}
	}

	// Get optimization recommendations
	for _, optimizer := range pm.optimizers {
		recommendations := optimizer.Optimize(report.Metrics)
		report.Recommendations = append(report.Recommendations, recommendations...)
	}

	// Calculate health score
	report.Metrics.HealthScore = pm.calculateHealthScore(report.Metrics)

	return report
}

// calculateHealthScore calculates an overall health score
func (pm *PerformanceMonitor) calculateHealthScore(metrics *PerformanceMetrics) float64 {
	score := 100.0

	// Penalize based on error rate
	if metrics.ErrorRate > 1.0 {
		score -= metrics.ErrorRate * 2
	}

	// Penalize based on latency
	if metrics.AverageLatency > time.Millisecond*100 {
		latencyPenalty := float64(metrics.AverageLatency.Milliseconds()) / 100.0
		score -= latencyPenalty
	}

	// Penalize based on memory usage
	memMB := float64(metrics.MemoryUsage) / 1024 / 1024
	if memMB > 512 {
		memPenalty := (memMB - 512) / 512 * 10
		score -= memPenalty
	}

	// Penalize based on goroutine count
	if metrics.GoroutineCount > 1000 {
		goroutinePenalty := float64(metrics.GoroutineCount-1000) / 1000 * 5
		score -= goroutinePenalty
	}

	if score < 0 {
		score = 0
	}

	return score
}

// snapshot returns a copy of the metrics without copying the mutex.
func (pm *PerformanceMetrics) snapshot() *PerformanceMetrics {
	return &PerformanceMetrics{
		TotalLogs:             pm.TotalLogs,
		LogsPerSecond:         pm.LogsPerSecond,
		AverageLatency:        pm.AverageLatency,
		MaxLatency:            pm.MaxLatency,
		MinLatency:            pm.MinLatency,
		P50Latency:            pm.P50Latency,
		P95Latency:            pm.P95Latency,
		P99Latency:            pm.P99Latency,
		DroppedLogs:           pm.DroppedLogs,
		BufferUtilization:     pm.BufferUtilization,
		MemoryUsage:           pm.MemoryUsage,
		CPUUsage:              pm.CPUUsage,
		GoroutineCount:        pm.GoroutineCount,
		AllocationsPerSecond:  pm.AllocationsPerSecond,
		GCPauseTime:           pm.GCPauseTime,
		LastReportTime:        pm.LastReportTime,
		WindowStart:           pm.WindowStart,
		SampleCount:           pm.SampleCount,
		ErrorRate:             pm.ErrorRate,
		SuccessRate:           pm.SuccessRate,
		AverageProcessingTime: pm.AverageProcessingTime,
		TotalProcessingTime:   pm.TotalProcessingTime,
		CacheHitRate:          pm.CacheHitRate,
		CacheMissRate:         pm.CacheMissRate,
		QueueDepth:            pm.QueueDepth,
		BackpressureEvents:    pm.BackpressureEvents,
		CircuitBreakerState:   pm.CircuitBreakerState,
		HealthScore:           pm.HealthScore,
	}
}

// CheckThresholds checks if any thresholds are exceeded
func (pm *PerformanceMonitor) CheckThresholds() []ThresholdViolation {
	pm.metrics.mu.RLock()
	defer pm.metrics.mu.RUnlock()

	violations := make([]ThresholdViolation, 0)

	if latencyMs := float64(pm.metrics.AverageLatency.Milliseconds()); latencyMs > pm.alertThresholds["latency_ms"] {
		violations = append(violations, ThresholdViolation{
			Metric:    "latency_ms",
			Current:   latencyMs,
			Threshold: pm.alertThresholds["latency_ms"],
			Severity:  "warning",
		})
	}

	if pm.metrics.ErrorRate > pm.alertThresholds["error_rate"] {
		violations = append(violations, ThresholdViolation{
			Metric:    "error_rate",
			Current:   pm.metrics.ErrorRate,
			Threshold: pm.alertThresholds["error_rate"],
			Severity:  "critical",
		})
	}

	memMB := float64(pm.metrics.MemoryUsage) / 1024 / 1024
	if memMB > pm.alertThresholds["memory_usage_mb"] {
		violations = append(violations, ThresholdViolation{
			Metric:    "memory_usage_mb",
			Current:   memMB,
			Threshold: pm.alertThresholds["memory_usage_mb"],
			Severity:  "warning",
		})
	}

	if float64(pm.metrics.GoroutineCount) > pm.alertThresholds["goroutine_count"] {
		violations = append(violations, ThresholdViolation{
			Metric:    "goroutine_count",
			Current:   float64(pm.metrics.GoroutineCount),
			Threshold: pm.alertThresholds["goroutine_count"],
			Severity:  "warning",
		})
	}

	return violations
}

// reportLoop periodically reports metrics
func (pm *PerformanceMonitor) reportLoop() {
	for {
		select {
		case <-pm.ticker.C:
			pm.UpdateSystemMetrics()
			violations := pm.CheckThresholds()

			if len(violations) > 0 {
				pm.callbackRegistry.TriggerCallbacks("threshold_violation", violations)
			}

		case <-pm.done:
			pm.ticker.Stop()
			return
		}
	}
}

// Close closes the performance monitor
func (pm *PerformanceMonitor) Close() {
	pm.done <- true
}

// ThresholdViolation represents a threshold violation
type ThresholdViolation struct {
	Metric    string
	Current   float64
	Threshold float64
	Severity  string
	Timestamp time.Time
}

// PerformanceReport represents a performance report
type PerformanceReport struct {
	GeneratedAt     time.Time
	Duration        time.Duration
	Metrics         *PerformanceMetrics
	Samples         map[string][]PerformanceSample
	Hotspots        []Hotspot
	Recommendations []OptimizationRecommendation
	Summary         string
}

// NewPerformanceSampler creates a new performance sampler
func NewPerformanceSampler(name string, maxSamples int, windowSize time.Duration) *PerformanceSampler {
	return &PerformanceSampler{
		name:         name,
		samples:      make([]PerformanceSample, 0, maxSamples),
		maxSamples:   maxSamples,
		windowSize:   windowSize,
		aggregations: make(map[string]float64),
	}
}

// AddSample adds a sample
func (ps *PerformanceSampler) AddSample(sample PerformanceSample) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// Remove old samples
	now := time.Now()
	validSamples := make([]PerformanceSample, 0, len(ps.samples))
	for _, s := range ps.samples {
		if now.Sub(s.Timestamp) <= ps.windowSize {
			validSamples = append(validSamples, s)
		}
	}

	// Add new sample
	validSamples = append(validSamples, sample)

	// Limit to max samples
	if len(validSamples) > ps.maxSamples {
		validSamples = validSamples[len(validSamples)-ps.maxSamples:]
	}

	ps.samples = validSamples
	ps.calculateAggregations()
}

// GetSamples returns all samples
func (ps *PerformanceSampler) GetSamples() []PerformanceSample {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	samples := make([]PerformanceSample, len(ps.samples))
	copy(samples, ps.samples)
	return samples
}

// calculateAggregations calculates aggregated metrics
func (ps *PerformanceSampler) calculateAggregations() {
	if len(ps.samples) == 0 {
		return
	}

	var totalLatency time.Duration
	var totalMem uint64
	var totalCPU float64
	successCount := 0

	for _, sample := range ps.samples {
		totalLatency += sample.Latency
		totalMem += sample.MemoryUsage
		totalCPU += sample.CPUUsage
		if sample.Success {
			successCount++
		}
	}

	count := float64(len(ps.samples))
	ps.aggregations["avg_latency_ms"] = float64(totalLatency.Milliseconds()) / count
	ps.aggregations["avg_memory_mb"] = float64(totalMem) / count / 1024 / 1024
	ps.aggregations["avg_cpu_percent"] = totalCPU / count
	ps.aggregations["success_rate"] = float64(successCount) / count * 100
}

// CallbackRegistry manages performance callbacks
type CallbackRegistry struct {
	mu        sync.RWMutex
	callbacks map[string][]PerformanceCallback
}

// PerformanceCallback is a callback function
type PerformanceCallback func(event string, data interface{})

// NewCallbackRegistry creates a new callback registry
func NewCallbackRegistry() *CallbackRegistry {
	return &CallbackRegistry{
		callbacks: make(map[string][]PerformanceCallback),
	}
}

// RegisterCallback registers a callback
func (cr *CallbackRegistry) RegisterCallback(event string, callback PerformanceCallback) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if cr.callbacks[event] == nil {
		cr.callbacks[event] = make([]PerformanceCallback, 0)
	}
	cr.callbacks[event] = append(cr.callbacks[event], callback)
}

// TriggerCallbacks triggers callbacks for an event
func (cr *CallbackRegistry) TriggerCallbacks(event string, data interface{}) {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	if callbacks, exists := cr.callbacks[event]; exists {
		for _, callback := range callbacks {
			go callback(event, data)
		}
	}
}

// ResourceTracker tracks resource usage
type ResourceTracker struct {
	mu           sync.RWMutex
	snapshots    []ResourceSnapshot
	maxSnapshots int
}

// ResourceSnapshot represents a resource usage snapshot
type ResourceSnapshot struct {
	Timestamp     time.Time
	MemoryAlloc   uint64
	MemorySys     uint64
	NumGC         uint32
	GCCPUFraction float64
	NumGoroutine  int
	NumCgoCall    int64
	ThreadCount   int
	HeapObjects   uint64
	StackInUse    uint64
}

// NewResourceTracker creates a new resource tracker
func NewResourceTracker() *ResourceTracker {
	return &ResourceTracker{
		snapshots:    make([]ResourceSnapshot, 0, 100),
		maxSnapshots: 100,
	}
}

// TakeSnapshot takes a resource snapshot
func (rt *ResourceTracker) TakeSnapshot() ResourceSnapshot {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	snapshot := ResourceSnapshot{
		Timestamp:     time.Now(),
		MemoryAlloc:   m.Alloc,
		MemorySys:     m.Sys,
		NumGC:         m.NumGC,
		GCCPUFraction: m.GCCPUFraction,
		NumGoroutine:  runtime.NumGoroutine(),
		NumCgoCall:    runtime.NumCgoCall(),
		HeapObjects:   m.HeapObjects,
		StackInUse:    m.StackInuse,
	}

	rt.snapshots = append(rt.snapshots, snapshot)
	if len(rt.snapshots) > rt.maxSnapshots {
		rt.snapshots = rt.snapshots[1:]
	}

	return snapshot
}

// GetSnapshots returns all snapshots
func (rt *ResourceTracker) GetSnapshots() []ResourceSnapshot {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	snapshots := make([]ResourceSnapshot, len(rt.snapshots))
	copy(snapshots, rt.snapshots)
	return snapshots
}

// LatencyHistogram tracks latency distribution
type LatencyHistogram struct {
	mu      sync.RWMutex
	buckets map[string]int64
	samples []time.Duration
}

// NewLatencyHistogram creates a new latency histogram
func NewLatencyHistogram() *LatencyHistogram {
	return &LatencyHistogram{
		buckets: make(map[string]int64),
		samples: make([]time.Duration, 0, 1000),
	}
}

// Record records a latency measurement
func (lh *LatencyHistogram) Record(latency time.Duration) {
	lh.mu.Lock()
	defer lh.mu.Unlock()

	ms := latency.Milliseconds()
	bucket := lh.getBucket(ms)
	lh.buckets[bucket]++

	lh.samples = append(lh.samples, latency)
	if len(lh.samples) > 1000 {
		lh.samples = lh.samples[1:]
	}
}

// getBucket returns the bucket for a latency value
func (lh *LatencyHistogram) getBucket(ms int64) string {
	switch {
	case ms < 10:
		return "0-10ms"
	case ms < 50:
		return "10-50ms"
	case ms < 100:
		return "50-100ms"
	case ms < 500:
		return "100-500ms"
	case ms < 1000:
		return "500-1000ms"
	default:
		return "1000ms+"
	}
}

// GetPercentile calculates a percentile
func (lh *LatencyHistogram) GetPercentile(p float64) time.Duration {
	lh.mu.RLock()
	defer lh.mu.RUnlock()

	if len(lh.samples) == 0 {
		return 0
	}

	sorted := make([]time.Duration, len(lh.samples))
	copy(sorted, lh.samples)

	// Simple bubble sort for small arrays
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	index := int(float64(len(sorted)) * p / 100.0)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	return sorted[index]
}

// ThroughputMeter measures throughput
type ThroughputMeter struct {
	mu         sync.RWMutex
	count      int64
	startTime  time.Time
	lastReport time.Time
}

// NewThroughputMeter creates a new throughput meter
func NewThroughputMeter() *ThroughputMeter {
	now := time.Now()
	return &ThroughputMeter{
		startTime:  now,
		lastReport: now,
	}
}

// Record records an event
func (tm *ThroughputMeter) Record() {
	atomic.AddInt64(&tm.count, 1)
}

// GetRate returns the current throughput rate
func (tm *ThroughputMeter) GetRate() float64 {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	count := atomic.LoadInt64(&tm.count)
	elapsed := time.Since(tm.startTime).Seconds()

	if elapsed > 0 {
		return float64(count) / elapsed
	}
	return 0
}

// MemoryAnalyzer analyzes memory usage
type MemoryAnalyzer struct {
	mu      sync.RWMutex
	history []MemorySnapshot
}

// MemorySnapshot represents a memory usage snapshot
type MemorySnapshot struct {
	Timestamp    time.Time
	Alloc        uint64
	TotalAlloc   uint64
	Sys          uint64
	HeapAlloc    uint64
	HeapSys      uint64
	HeapIdle     uint64
	HeapInuse    uint64
	HeapReleased uint64
	HeapObjects  uint64
}

// NewMemoryAnalyzer creates a new memory analyzer
func NewMemoryAnalyzer() *MemoryAnalyzer {
	return &MemoryAnalyzer{
		history: make([]MemorySnapshot, 0, 100),
	}
}

// Analyze analyzes memory statistics
func (ma *MemoryAnalyzer) Analyze(m *runtime.MemStats) {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	snapshot := MemorySnapshot{
		Timestamp:    time.Now(),
		Alloc:        m.Alloc,
		TotalAlloc:   m.TotalAlloc,
		Sys:          m.Sys,
		HeapAlloc:    m.HeapAlloc,
		HeapSys:      m.HeapSys,
		HeapIdle:     m.HeapIdle,
		HeapInuse:    m.HeapInuse,
		HeapReleased: m.HeapReleased,
		HeapObjects:  m.HeapObjects,
	}

	ma.history = append(ma.history, snapshot)
	if len(ma.history) > 100 {
		ma.history = ma.history[1:]
	}
}

// GetTrend returns memory usage trend
func (ma *MemoryAnalyzer) GetTrend() string {
	ma.mu.RLock()
	defer ma.mu.RUnlock()

	if len(ma.history) < 2 {
		return "unknown"
	}

	first := ma.history[0].Alloc
	last := ma.history[len(ma.history)-1].Alloc

	if last > first*2 {
		return "rapidly_increasing"
	} else if last > first {
		return "increasing"
	} else if last < first/2 {
		return "rapidly_decreasing"
	} else if last < first {
		return "decreasing"
	}

	return "stable"
}

// CPUProfiler profiles CPU usage
type CPUProfiler struct {
	mu       sync.RWMutex
	active   bool
	filename string
}

// NewCPUProfiler creates a new CPU profiler
func NewCPUProfiler() *CPUProfiler {
	return &CPUProfiler{}
}

// Start starts CPU profiling
func (cp *CPUProfiler) Start(filename string) error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if cp.active {
		return fmt.Errorf("profiler already active")
	}

	cp.filename = filename
	cp.active = true

	// In real implementation, would call pprof.StartCPUProfile
	return nil
}

// Stop stops CPU profiling
func (cp *CPUProfiler) Stop() {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if cp.active {
		// In real implementation, would call pprof.StopCPUProfile
		cp.active = false
	}
}

// BlockProfiler profiles blocking events
type BlockProfiler struct {
	mu     sync.RWMutex
	active bool
	rate   int
}

// NewBlockProfiler creates a new block profiler
func NewBlockProfiler() *BlockProfiler {
	return &BlockProfiler{
		rate: 1,
	}
}

// Start starts block profiling
func (bp *BlockProfiler) Start(rate int) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	bp.rate = rate
	bp.active = true
	runtime.SetBlockProfileRate(rate)
}

// Stop stops block profiling
func (bp *BlockProfiler) Stop() {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	if bp.active {
		runtime.SetBlockProfileRate(0)
		bp.active = false
	}
}

// GetProfile returns the block profile
func (bp *BlockProfiler) GetProfile() *pprof.Profile {
	return pprof.Lookup("block")
}

// GoroutineAnalyzer analyzes goroutines
type GoroutineAnalyzer struct {
	mu      sync.RWMutex
	history []GoroutineSnapshot
}

// GoroutineSnapshot represents a goroutine count snapshot
type GoroutineSnapshot struct {
	Timestamp time.Time
	Count     int
	Peak      int
}

// NewGoroutineAnalyzer creates a new goroutine analyzer
func NewGoroutineAnalyzer() *GoroutineAnalyzer {
	return &GoroutineAnalyzer{
		history: make([]GoroutineSnapshot, 0, 100),
	}
}

// Analyze analyzes goroutine count
func (ga *GoroutineAnalyzer) Analyze() {
	ga.mu.Lock()
	defer ga.mu.Unlock()

	count := runtime.NumGoroutine()
	peak := count

	if len(ga.history) > 0 {
		lastPeak := ga.history[len(ga.history)-1].Peak
		if count > lastPeak {
			peak = count
		} else {
			peak = lastPeak
		}
	}

	snapshot := GoroutineSnapshot{
		Timestamp: time.Now(),
		Count:     count,
		Peak:      peak,
	}

	ga.history = append(ga.history, snapshot)
	if len(ga.history) > 100 {
		ga.history = ga.history[1:]
	}
}

// GetPeak returns the peak goroutine count
func (ga *GoroutineAnalyzer) GetPeak() int {
	ga.mu.RLock()
	defer ga.mu.RUnlock()

	if len(ga.history) == 0 {
		return 0
	}

	return ga.history[len(ga.history)-1].Peak
}

// AllocationTracker tracks memory allocations
type AllocationTracker struct {
	mu          sync.RWMutex
	allocations []AllocationRecord
}

// AllocationRecord represents an allocation record
type AllocationRecord struct {
	Timestamp time.Time
	Size      uint64
	Location  string
}

// NewAllocationTracker creates a new allocation tracker
func NewAllocationTracker() *AllocationTracker {
	return &AllocationTracker{
		allocations: make([]AllocationRecord, 0, 1000),
	}
}

// Track tracks allocations
func (at *AllocationTracker) Track() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	at.mu.Lock()
	defer at.mu.Unlock()

	record := AllocationRecord{
		Timestamp: time.Now(),
		Size:      m.Alloc,
	}

	at.allocations = append(at.allocations, record)
	if len(at.allocations) > 1000 {
		at.allocations = at.allocations[1:]
	}
}

// CachingMetrics tracks caching performance
type CachingMetrics struct {
	mu        sync.RWMutex
	hits      int64
	misses    int64
	evictions int64
	size      int64
}

// NewCachingMetrics creates new caching metrics
func NewCachingMetrics() *CachingMetrics {
	return &CachingMetrics{}
}

// RecordHit records a cache hit
func (cm *CachingMetrics) RecordHit() {
	atomic.AddInt64(&cm.hits, 1)
}

// RecordMiss records a cache miss
func (cm *CachingMetrics) RecordMiss() {
	atomic.AddInt64(&cm.misses, 1)
}

// RecordEviction records a cache eviction
func (cm *CachingMetrics) RecordEviction() {
	atomic.AddInt64(&cm.evictions, 1)
}

// GetHitRate returns the cache hit rate
func (cm *CachingMetrics) GetHitRate() float64 {
	hits := atomic.LoadInt64(&cm.hits)
	misses := atomic.LoadInt64(&cm.misses)
	total := hits + misses

	if total == 0 {
		return 0
	}

	return float64(hits) / float64(total) * 100
}

// IOMetrics tracks I/O performance
type IOMetrics struct {
	mu           sync.RWMutex
	bytesRead    int64
	bytesWritten int64
	readOps      int64
	writeOps     int64
}

// NewIOMetrics creates new I/O metrics
func NewIOMetrics() *IOMetrics {
	return &IOMetrics{}
}

// RecordRead records a read operation
func (io *IOMetrics) RecordRead(bytes int64) {
	atomic.AddInt64(&io.bytesRead, bytes)
	atomic.AddInt64(&io.readOps, 1)
}

// RecordWrite records a write operation
func (io *IOMetrics) RecordWrite(bytes int64) {
	atomic.AddInt64(&io.bytesWritten, bytes)
	atomic.AddInt64(&io.writeOps, 1)
}

// GetThroughput returns I/O throughput
func (io *IOMetrics) GetThroughput() (readMBps, writeMBps float64) {
	read := atomic.LoadInt64(&io.bytesRead)
	written := atomic.LoadInt64(&io.bytesWritten)

	readMBps = float64(read) / 1024 / 1024
	writeMBps = float64(written) / 1024 / 1024

	return
}

// ConcurrencyMetrics tracks concurrency metrics
type ConcurrencyMetrics struct {
	mu               sync.RWMutex
	activeWorkers    int32
	queuedTasks      int32
	completedTasks   int64
	failedTasks      int64
	avgWaitTime      time.Duration
	avgExecutionTime time.Duration
}

// NewConcurrencyMetrics creates new concurrency metrics
func NewConcurrencyMetrics() *ConcurrencyMetrics {
	return &ConcurrencyMetrics{}
}

// RecordTaskStart records task start
func (cm *ConcurrencyMetrics) RecordTaskStart() {
	atomic.AddInt32(&cm.activeWorkers, 1)
	atomic.AddInt32(&cm.queuedTasks, -1)
}

// RecordTaskComplete records task completion
func (cm *ConcurrencyMetrics) RecordTaskComplete(executionTime time.Duration) {
	atomic.AddInt32(&cm.activeWorkers, -1)
	atomic.AddInt64(&cm.completedTasks, 1)
}

// RecordTaskFailed records task failure
func (cm *ConcurrencyMetrics) RecordTaskFailed() {
	atomic.AddInt32(&cm.activeWorkers, -1)
	atomic.AddInt64(&cm.failedTasks, 1)
}

// GetMetrics returns current metrics
func (cm *ConcurrencyMetrics) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"active_workers":  atomic.LoadInt32(&cm.activeWorkers),
		"queued_tasks":    atomic.LoadInt32(&cm.queuedTasks),
		"completed_tasks": atomic.LoadInt64(&cm.completedTasks),
		"failed_tasks":    atomic.LoadInt64(&cm.failedTasks),
	}
}

// MarshalJSON custom JSON marshaling for PerformanceMetrics
func (pm *PerformanceMetrics) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"total_logs":             pm.TotalLogs,
		"logs_per_second":        pm.LogsPerSecond,
		"average_latency_ms":     pm.AverageLatency.Milliseconds(),
		"max_latency_ms":         pm.MaxLatency.Milliseconds(),
		"min_latency_ms":         pm.MinLatency.Milliseconds(),
		"p50_latency_ms":         pm.P50Latency.Milliseconds(),
		"p95_latency_ms":         pm.P95Latency.Milliseconds(),
		"p99_latency_ms":         pm.P99Latency.Milliseconds(),
		"dropped_logs":           pm.DroppedLogs,
		"buffer_utilization":     pm.BufferUtilization,
		"memory_usage_mb":        float64(pm.MemoryUsage) / 1024 / 1024,
		"cpu_usage_percent":      pm.CPUUsage,
		"goroutine_count":        pm.GoroutineCount,
		"allocations_per_second": pm.AllocationsPerSecond,
		"gc_pause_time_ms":       pm.GCPauseTime.Milliseconds(),
		"error_rate":             pm.ErrorRate,
		"success_rate":           pm.SuccessRate,
		"health_score":           pm.HealthScore,
		"last_report_time":       pm.LastReportTime,
	})
}

// ProfileContext manages profiling context
type ProfileContext struct {
	ctx           context.Context
	startTime     time.Time
	operationName string
	metadata      map[string]interface{}
}

// NewProfileContext creates a new profile context
func NewProfileContext(ctx context.Context, operationName string) *ProfileContext {
	return &ProfileContext{
		ctx:           ctx,
		startTime:     time.Now(),
		operationName: operationName,
		metadata:      make(map[string]interface{}),
	}
}

// End ends the profile context and returns the duration
func (pc *ProfileContext) End() time.Duration {
	return time.Since(pc.startTime)
}

// AddMetadata adds metadata to the profile
func (pc *ProfileContext) AddMetadata(key string, value interface{}) {
	pc.metadata[key] = value
}

// GetMetadata returns all metadata
func (pc *ProfileContext) GetMetadata() map[string]interface{} {
	return pc.metadata
}
