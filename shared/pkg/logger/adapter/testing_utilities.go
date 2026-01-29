package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"shared/pkg/logger"
)

// LogCapture captures log output for testing
type LogCapture struct {
	mu      sync.RWMutex
	entries []CapturedLogEntry
	buffer  *bytes.Buffer
	enabled bool
}

// CapturedLogEntry represents a captured log entry
type CapturedLogEntry struct {
	Level     string
	Message   string
	Fields    map[string]interface{}
	Timestamp time.Time
	Caller    string
	Stack     string
}

// NewLogCapture creates a new log capture
func NewLogCapture() *LogCapture {
	return &LogCapture{
		entries: make([]CapturedLogEntry, 0),
		buffer:  new(bytes.Buffer),
		enabled: true,
	}
}

// Capture captures a log entry
func (lc *LogCapture) Capture(level, message string, fields []logger.Field) {
	if !lc.enabled {
		return
	}

	lc.mu.Lock()
	defer lc.mu.Unlock()

	fieldsMap := make(map[string]interface{})
	for _, field := range fields {
		if field != nil {
			fieldsMap[field.Key()] = field.Value()
		}
	}

	entry := CapturedLogEntry{
		Level:     level,
		Message:   message,
		Fields:    fieldsMap,
		Timestamp: time.Now(),
		Caller:    getCaller(3),
	}

	lc.entries = append(lc.entries, entry)
}

// GetEntries returns all captured entries
func (lc *LogCapture) GetEntries() []CapturedLogEntry {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	entries := make([]CapturedLogEntry, len(lc.entries))
	copy(entries, lc.entries)
	return entries
}

// Clear clears all captured entries
func (lc *LogCapture) Clear() {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	lc.entries = lc.entries[:0]
	lc.buffer.Reset()
}

// FindEntry finds an entry by criteria
func (lc *LogCapture) FindEntry(level, messageSubstring string) *CapturedLogEntry {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	for i := range lc.entries {
		if lc.entries[i].Level == level && strings.Contains(lc.entries[i].Message, messageSubstring) {
			return &lc.entries[i]
		}
	}
	return nil
}

// CountByLevel counts entries by level
func (lc *LogCapture) CountByLevel(level string) int {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	count := 0
	for _, entry := range lc.entries {
		if entry.Level == level {
			count++
		}
	}
	return count
}

// getCaller returns caller information
func getCaller(skip int) string {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown"
	}

	fn := runtime.FuncForPC(pc)
	fnName := "unknown"
	if fn != nil {
		fnName = fn.Name()
	}

	return fmt.Sprintf("%s:%d (%s)", filepath.Base(file), line, fnName)
}

// MockLogger provides a mock logger for testing
type MockLogger struct {
	mu               sync.RWMutex
	capture          *LogCapture
	expectations     []LogExpectation
	failOnUnexpected bool
	t                *testing.T
}

// LogExpectation represents an expected log call
type LogExpectation struct {
	Level          string
	MessagePattern string
	FieldMatchers  map[string]FieldMatcher
	Called         bool
	CallCount      int
	MinCalls       int
	MaxCalls       int
}

// FieldMatcher matches field values
type FieldMatcher func(value interface{}) bool

// NewMockLogger creates a new mock logger
func NewMockLogger(t *testing.T) *MockLogger {
	return &MockLogger{
		capture:          NewLogCapture(),
		expectations:     make([]LogExpectation, 0),
		failOnUnexpected: true,
		t:                t,
	}
}

// Expect adds an expectation
func (ml *MockLogger) Expect(level, messagePattern string) *LogExpectation {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	exp := LogExpectation{
		Level:          level,
		MessagePattern: messagePattern,
		FieldMatchers:  make(map[string]FieldMatcher),
		MinCalls:       1,
		MaxCalls:       -1,
	}

	ml.expectations = append(ml.expectations, exp)
	return &ml.expectations[len(ml.expectations)-1]
}

// Debug implements logger.Logger
func (ml *MockLogger) Debug(msg string, fields ...logger.Field) {
	ml.capture.Capture("DEBUG", msg, fields)
	ml.checkExpectations("DEBUG", msg, fields)
}

// Info implements logger.Logger
func (ml *MockLogger) Info(msg string, fields ...logger.Field) {
	ml.capture.Capture("INFO", msg, fields)
	ml.checkExpectations("INFO", msg, fields)
}

// Warn implements logger.Logger
func (ml *MockLogger) Warn(msg string, fields ...logger.Field) {
	ml.capture.Capture("WARN", msg, fields)
	ml.checkExpectations("WARN", msg, fields)
}

// Error implements logger.Logger
func (ml *MockLogger) Error(msg string, fields ...logger.Field) {
	ml.capture.Capture("ERROR", msg, fields)
	ml.checkExpectations("ERROR", msg, fields)
}

// Fatal implements logger.Logger
func (ml *MockLogger) Fatal(msg string, fields ...logger.Field) {
	ml.capture.Capture("FATAL", msg, fields)
	ml.checkExpectations("FATAL", msg, fields)
}

// Request implements logger.Logger
func (ml *MockLogger) Request(ctx context.Context, method string, routePath string, statusCode int, duration time.Duration, bodySize int64, msg string, fields ...logger.Field) {
	ml.capture.Capture("REQUEST", msg, fields)
}

// With implements logger.Logger
func (ml *MockLogger) With(fields ...logger.Field) logger.Logger {
	return ml
}

// WithContext implements logger.Logger
func (ml *MockLogger) WithContext(ctx context.Context) logger.Logger {
	return ml
}

// Sync implements logger.Logger
func (ml *MockLogger) Sync() error {
	return nil
}

// checkExpectations checks if log call matches expectations
func (ml *MockLogger) checkExpectations(level, msg string, fields []logger.Field) {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	matched := false
	for i := range ml.expectations {
		exp := &ml.expectations[i]
		if exp.Level == level && strings.Contains(msg, exp.MessagePattern) {
			exp.Called = true
			exp.CallCount++
			matched = true

			// Check field matchers
			for fieldName, matcher := range exp.FieldMatchers {
				found := false
				for _, field := range fields {
					if field != nil && field.Key() == fieldName {
						if !matcher(field.Value()) {
							ml.t.Errorf("Field %s does not match expectation", fieldName)
						}
						found = true
						break
					}
				}
				if !found {
					ml.t.Errorf("Expected field %s not found", fieldName)
				}
			}
			break
		}
	}

	if !matched && ml.failOnUnexpected {
		ml.t.Errorf("Unexpected log call: %s %s", level, msg)
	}
}

// Verify verifies all expectations were met
func (ml *MockLogger) Verify() {
	ml.mu.RLock()
	defer ml.mu.RUnlock()

	for _, exp := range ml.expectations {
		if !exp.Called && exp.MinCalls > 0 {
			ml.t.Errorf("Expected log %s %s was not called", exp.Level, exp.MessagePattern)
		}

		if exp.CallCount < exp.MinCalls {
			ml.t.Errorf("Expected log %s %s was called %d times, expected at least %d", exp.Level, exp.MessagePattern, exp.CallCount, exp.MinCalls)
		}

		if exp.MaxCalls > 0 && exp.CallCount > exp.MaxCalls {
			ml.t.Errorf("Expected log %s %s was called %d times, expected at most %d", exp.Level, exp.MessagePattern, exp.CallCount, exp.MaxCalls)
		}
	}
}

// BenchmarkLogger provides benchmarking utilities for logging
type BenchmarkLogger struct {
	mu       sync.RWMutex
	results  []BenchmarkResult
	baseline *BenchmarkResult
}

// BenchmarkResult represents a benchmark result
type BenchmarkResult struct {
	Name              string
	Operations        int
	Duration          time.Duration
	OpsPerSecond      float64
	AvgLatency        time.Duration
	MemoryAllocated   uint64
	MemoryAllocations int64
	GCPauses          int
	Timestamp         time.Time
}

// NewBenchmarkLogger creates a new benchmark logger
func NewBenchmarkLogger() *BenchmarkLogger {
	return &BenchmarkLogger{
		results: make([]BenchmarkResult, 0),
	}
}

// Benchmark benchmarks a logging operation
func (bl *BenchmarkLogger) Benchmark(name string, iterations int, operation func()) BenchmarkResult {
	runtime.GC()

	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	startTime := time.Now()

	for i := 0; i < iterations; i++ {
		operation()
	}

	duration := time.Since(startTime)

	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	result := BenchmarkResult{
		Name:              name,
		Operations:        iterations,
		Duration:          duration,
		OpsPerSecond:      float64(iterations) / duration.Seconds(),
		AvgLatency:        duration / time.Duration(iterations),
		MemoryAllocated:   m2.TotalAlloc - m1.TotalAlloc,
		MemoryAllocations: int64(m2.Mallocs - m1.Mallocs),
		GCPauses:          int(m2.NumGC - m1.NumGC),
		Timestamp:         time.Now(),
	}

	bl.mu.Lock()
	bl.results = append(bl.results, result)
	bl.mu.Unlock()

	return result
}

// SetBaseline sets the baseline for comparisons
func (bl *BenchmarkLogger) SetBaseline(result BenchmarkResult) {
	bl.mu.Lock()
	defer bl.mu.Unlock()
	bl.baseline = &result
}

// CompareToBaseline compares a result to the baseline
func (bl *BenchmarkLogger) CompareToBaseline(result BenchmarkResult) BenchmarkComparison {
	bl.mu.RLock()
	defer bl.mu.RUnlock()

	if bl.baseline == nil {
		return BenchmarkComparison{
			Valid: false,
		}
	}

	return BenchmarkComparison{
		Valid:             true,
		OpsPerSecondDelta: result.OpsPerSecond - bl.baseline.OpsPerSecond,
		OpsPerSecondPct:   (result.OpsPerSecond/bl.baseline.OpsPerSecond - 1) * 100,
		LatencyDelta:      result.AvgLatency - bl.baseline.AvgLatency,
		LatencyPct:        (float64(result.AvgLatency)/float64(bl.baseline.AvgLatency) - 1) * 100,
		MemoryDelta:       int64(result.MemoryAllocated) - int64(bl.baseline.MemoryAllocated),
		MemoryPct:         (float64(result.MemoryAllocated)/float64(bl.baseline.MemoryAllocated) - 1) * 100,
	}
}

// BenchmarkComparison represents a comparison to baseline
type BenchmarkComparison struct {
	Valid             bool
	OpsPerSecondDelta float64
	OpsPerSecondPct   float64
	LatencyDelta      time.Duration
	LatencyPct        float64
	MemoryDelta       int64
	MemoryPct         float64
}

// GenerateReport generates a benchmark report
func (bl *BenchmarkLogger) GenerateReport() string {
	bl.mu.RLock()
	defer bl.mu.RUnlock()

	var sb strings.Builder
	sb.WriteString("=== Benchmark Report ===\n\n")

	for _, result := range bl.results {
		sb.WriteString(fmt.Sprintf("Benchmark: %s\n", result.Name))
		sb.WriteString(fmt.Sprintf("  Operations:     %d\n", result.Operations))
		sb.WriteString(fmt.Sprintf("  Duration:       %v\n", result.Duration))
		sb.WriteString(fmt.Sprintf("  Ops/sec:        %.2f\n", result.OpsPerSecond))
		sb.WriteString(fmt.Sprintf("  Avg Latency:    %v\n", result.AvgLatency))
		sb.WriteString(fmt.Sprintf("  Memory Alloc:   %d bytes\n", result.MemoryAllocated))
		sb.WriteString(fmt.Sprintf("  Allocations:    %d\n", result.MemoryAllocations))
		sb.WriteString(fmt.Sprintf("  GC Pauses:      %d\n", result.GCPauses))

		if bl.baseline != nil && result.Name != bl.baseline.Name {
			comp := bl.CompareToBaseline(result)
			if comp.Valid {
				sb.WriteString(fmt.Sprintf("  vs Baseline:\n"))
				sb.WriteString(fmt.Sprintf("    Ops/sec:      %+.2f%%\n", comp.OpsPerSecondPct))
				sb.WriteString(fmt.Sprintf("    Latency:      %+.2f%%\n", comp.LatencyPct))
				sb.WriteString(fmt.Sprintf("    Memory:       %+.2f%%\n", comp.MemoryPct))
			}
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

// TestHelper provides testing helpers
type TestHelper struct {
	t       *testing.T
	cleanup []func()
}

// NewTestHelper creates a new test helper
func NewTestHelper(t *testing.T) *TestHelper {
	return &TestHelper{
		t:       t,
		cleanup: make([]func(), 0),
	}
}

// AssertLogEntry asserts a log entry exists
func (th *TestHelper) AssertLogEntry(capture *LogCapture, level, messageSubstring string) {
	entry := capture.FindEntry(level, messageSubstring)
	if entry == nil {
		th.t.Errorf("Expected log entry not found: %s %s", level, messageSubstring)
	}
}

// AssertLogCount asserts log count by level
func (th *TestHelper) AssertLogCount(capture *LogCapture, level string, expected int) {
	actual := capture.CountByLevel(level)
	if actual != expected {
		th.t.Errorf("Expected %d %s logs, got %d", expected, level, actual)
	}
}

// AssertFieldValue asserts a field value
func (th *TestHelper) AssertFieldValue(entry *CapturedLogEntry, fieldName string, expected interface{}) {
	actual, ok := entry.Fields[fieldName]
	if !ok {
		th.t.Errorf("Field %s not found in log entry", fieldName)
		return
	}

	if !reflect.DeepEqual(actual, expected) {
		th.t.Errorf("Field %s: expected %v, got %v", fieldName, expected, actual)
	}
}

// AssertNoErrors asserts no error logs were captured
func (th *TestHelper) AssertNoErrors(capture *LogCapture) {
	errorCount := capture.CountByLevel("ERROR")
	if errorCount > 0 {
		th.t.Errorf("Expected no errors, got %d", errorCount)
	}
}

// AddCleanup adds a cleanup function
func (th *TestHelper) AddCleanup(f func()) {
	th.cleanup = append(th.cleanup, f)
}

// Cleanup runs all cleanup functions
func (th *TestHelper) Cleanup() {
	for i := len(th.cleanup) - 1; i >= 0; i-- {
		th.cleanup[i]()
	}
}

// LogValidator validates log entries
type LogValidator struct {
	rules []ValidationRule
}

// ValidationRule represents a validation rule
type ValidationRule struct {
	Name      string
	Validator func(entry CapturedLogEntry) error
	Severity  string
}

// NewLogValidator creates a new log validator
func NewLogValidator() *LogValidator {
	return &LogValidator{
		rules: make([]ValidationRule, 0),
	}
}

// AddRule adds a validation rule
func (lv *LogValidator) AddRule(rule ValidationRule) {
	lv.rules = append(lv.rules, rule)
}

// Validate validates log entries
func (lv *LogValidator) Validate(entries []CapturedLogEntry) []ValidationResult {
	results := make([]ValidationResult, 0)

	for _, entry := range entries {
		for _, rule := range lv.rules {
			if err := rule.Validator(entry); err != nil {
				results = append(results, ValidationResult{
					RuleName: rule.Name,
					Severity: rule.Severity,
					Entry:    entry,
					Error:    err,
				})
			}
		}
	}

	return results
}

// ValidationResult represents a validation result
type ValidationResult struct {
	RuleName string
	Severity string
	Entry    CapturedLogEntry
	Error    error
}

// LogGenerator generates test log data
type LogGenerator struct {
	patterns []LogPattern
	rng      *pseudoRandom
}

// LogPattern represents a log pattern
type LogPattern struct {
	Level           string
	MessageTemplate string
	FieldGenerators map[string]func() interface{}
	Weight          int
}

// NewLogGenerator creates a new log generator
func NewLogGenerator() *LogGenerator {
	return &LogGenerator{
		patterns: make([]LogPattern, 0),
		rng:      newPseudoRandom(),
	}
}

// AddPattern adds a log pattern
func (lg *LogGenerator) AddPattern(pattern LogPattern) {
	lg.patterns = append(lg.patterns, pattern)
}

// Generate generates log entries
func (lg *LogGenerator) Generate(count int) []CapturedLogEntry {
	entries := make([]CapturedLogEntry, 0, count)

	for i := 0; i < count; i++ {
		pattern := lg.selectPattern()
		entry := lg.generateEntry(pattern)
		entries = append(entries, entry)
	}

	return entries
}

// selectPattern selects a pattern based on weights
func (lg *LogGenerator) selectPattern() LogPattern {
	if len(lg.patterns) == 0 {
		return LogPattern{}
	}

	totalWeight := 0
	for _, pattern := range lg.patterns {
		totalWeight += pattern.Weight
	}

	r := lg.rng.intn(totalWeight)
	cumWeight := 0

	for _, pattern := range lg.patterns {
		cumWeight += pattern.Weight
		if r < cumWeight {
			return pattern
		}
	}

	return lg.patterns[0]
}

// generateEntry generates a log entry from a pattern
func (lg *LogGenerator) generateEntry(pattern LogPattern) CapturedLogEntry {
	fields := make(map[string]interface{})
	for name, generator := range pattern.FieldGenerators {
		fields[name] = generator()
	}

	return CapturedLogEntry{
		Level:     pattern.Level,
		Message:   pattern.MessageTemplate,
		Fields:    fields,
		Timestamp: time.Now(),
	}
}

// pseudoRandom provides pseudo-random number generation
type pseudoRandom struct {
	state uint64
}

// newPseudoRandom creates a new pseudo-random generator
func newPseudoRandom() *pseudoRandom {
	return &pseudoRandom{
		state: uint64(time.Now().UnixNano()),
	}
}

// intn returns a random number in [0, n)
func (pr *pseudoRandom) intn(n int) int {
	pr.state = pr.state*1103515245 + 12345
	return int(pr.state/65536) % n
}

// LogRecorder records logs to a file for replay
type LogRecorder struct {
	mu       sync.Mutex
	file     *os.File
	encoder  *json.Encoder
	filename string
}

// NewLogRecorder creates a new log recorder
func NewLogRecorder(filename string) (*LogRecorder, error) {
	file, err := os.Create(filename)
	if err != nil {
		return nil, err
	}

	return &LogRecorder{
		file:     file,
		encoder:  json.NewEncoder(file),
		filename: filename,
	}, nil
}

// Record records a log entry
func (lr *LogRecorder) Record(entry CapturedLogEntry) error {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	return lr.encoder.Encode(entry)
}

// Close closes the recorder
func (lr *LogRecorder) Close() error {
	return lr.file.Close()
}

// LogReplayer replays recorded logs
type LogReplayer struct {
	mu      sync.Mutex
	file    *os.File
	decoder *json.Decoder
}

// NewLogReplayer creates a new log replayer
func NewLogReplayer(filename string) (*LogReplayer, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	return &LogReplayer{
		file:    file,
		decoder: json.NewDecoder(file),
	}, nil
}

// Replay replays log entries
func (lr *LogReplayer) Replay(handler func(entry CapturedLogEntry)) error {
	lr.mu.Lock()
	defer lr.mu.Unlock()

	for {
		var entry CapturedLogEntry
		if err := lr.decoder.Decode(&entry); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		handler(entry)
	}

	return nil
}

// Close closes the replayer
func (lr *LogReplayer) Close() error {
	return lr.file.Close()
}

// LogDiffer compares log outputs
type LogDiffer struct {
	mu sync.RWMutex
}

// NewLogDiffer creates a new log differ
func NewLogDiffer() *LogDiffer {
	return &LogDiffer{}
}

// Diff compares two sets of log entries
func (ld *LogDiffer) Diff(expected, actual []CapturedLogEntry) []LogDiff {
	ld.mu.Lock()
	defer ld.mu.Unlock()

	diffs := make([]LogDiff, 0)

	// Check for missing entries
	for i, exp := range expected {
		if i >= len(actual) {
			diffs = append(diffs, LogDiff{
				Type:     "missing",
				Index:    i,
				Expected: &exp,
			})
			continue
		}

		act := actual[i]

		if exp.Level != act.Level {
			diffs = append(diffs, LogDiff{
				Type:     "level_mismatch",
				Index:    i,
				Expected: &exp,
				Actual:   &act,
			})
		}

		if exp.Message != act.Message {
			diffs = append(diffs, LogDiff{
				Type:     "message_mismatch",
				Index:    i,
				Expected: &exp,
				Actual:   &act,
			})
		}

		// Compare fields
		for key, expValue := range exp.Fields {
			actValue, ok := act.Fields[key]
			if !ok {
				diffs = append(diffs, LogDiff{
					Type:      "field_missing",
					Index:     i,
					FieldName: key,
					Expected:  &exp,
					Actual:    &act,
				})
				continue
			}

			if !reflect.DeepEqual(expValue, actValue) {
				diffs = append(diffs, LogDiff{
					Type:          "field_mismatch",
					Index:         i,
					FieldName:     key,
					Expected:      &exp,
					Actual:        &act,
					ExpectedValue: expValue,
					ActualValue:   actValue,
				})
			}
		}
	}

	// Check for extra entries
	if len(actual) > len(expected) {
		for i := len(expected); i < len(actual); i++ {
			diffs = append(diffs, LogDiff{
				Type:   "extra",
				Index:  i,
				Actual: &actual[i],
			})
		}
	}

	return diffs
}

// LogDiff represents a difference in log entries
type LogDiff struct {
	Type          string
	Index         int
	FieldName     string
	Expected      *CapturedLogEntry
	Actual        *CapturedLogEntry
	ExpectedValue interface{}
	ActualValue   interface{}
}

// String returns a string representation
func (ld LogDiff) String() string {
	switch ld.Type {
	case "missing":
		return fmt.Sprintf("[%d] Missing entry: %s %s", ld.Index, ld.Expected.Level, ld.Expected.Message)
	case "extra":
		return fmt.Sprintf("[%d] Extra entry: %s %s", ld.Index, ld.Actual.Level, ld.Actual.Message)
	case "level_mismatch":
		return fmt.Sprintf("[%d] Level mismatch: expected %s, got %s", ld.Index, ld.Expected.Level, ld.Actual.Level)
	case "message_mismatch":
		return fmt.Sprintf("[%d] Message mismatch: expected %s, got %s", ld.Index, ld.Expected.Message, ld.Actual.Message)
	case "field_missing":
		return fmt.Sprintf("[%d] Field %s missing", ld.Index, ld.FieldName)
	case "field_mismatch":
		return fmt.Sprintf("[%d] Field %s mismatch: expected %v, got %v", ld.Index, ld.FieldName, ld.ExpectedValue, ld.ActualValue)
	default:
		return fmt.Sprintf("[%d] Unknown diff type: %s", ld.Index, ld.Type)
	}
}

// LogAnalyzer analyzes test logs
type LogAnalyzer struct {
	mu sync.RWMutex
}

// NewLogAnalyzer creates a new log analyzer
func NewLogAnalyzer() *LogAnalyzer {
	return &LogAnalyzer{}
}

// Analyze analyzes log entries
func (la *LogAnalyzer) Analyze(entries []CapturedLogEntry) AnalysisReport {
	la.mu.Lock()
	defer la.mu.Unlock()

	report := AnalysisReport{
		TotalEntries: len(entries),
		LevelCounts:  make(map[string]int),
		CallerCounts: make(map[string]int),
		FieldStats:   make(map[string]FieldStats),
		TimeRange:    TimeRange{},
	}

	if len(entries) == 0 {
		return report
	}

	report.TimeRange.Start = entries[0].Timestamp
	report.TimeRange.End = entries[len(entries)-1].Timestamp

	for _, entry := range entries {
		report.LevelCounts[entry.Level]++
		report.CallerCounts[entry.Caller]++

		for key, value := range entry.Fields {
			stats, ok := report.FieldStats[key]
			if !ok {
				stats = FieldStats{
					Name:   key,
					Count:  0,
					Values: make(map[interface{}]int),
				}
			}

			stats.Count++
			stats.Values[value]++
			report.FieldStats[key] = stats
		}
	}

	return report
}

// AnalysisReport represents an analysis report
type AnalysisReport struct {
	TotalEntries int
	LevelCounts  map[string]int
	CallerCounts map[string]int
	FieldStats   map[string]FieldStats
	TimeRange    TimeRange
}

// FieldStats represents statistics for a field
type FieldStats struct {
	Name   string
	Count  int
	Values map[interface{}]int
}

// LogMatcher provides pattern matching for logs
type LogMatcher struct {
	mu sync.RWMutex
}

// NewLogMatcher creates a new log matcher
func NewLogMatcher() *LogMatcher {
	return &LogMatcher{}
}

// Match matches log entries against patterns
func (lm *LogMatcher) Match(entries []CapturedLogEntry, patterns []MatchPattern) []MatchResult {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	results := make([]MatchResult, 0)

	for i, entry := range entries {
		for _, pattern := range patterns {
			if lm.matchesPattern(entry, pattern) {
				results = append(results, MatchResult{
					Index:   i,
					Entry:   entry,
					Pattern: pattern,
				})
			}
		}
	}

	return results
}

// matchesPattern checks if an entry matches a pattern
func (lm *LogMatcher) matchesPattern(entry CapturedLogEntry, pattern MatchPattern) bool {
	if pattern.Level != "" && entry.Level != pattern.Level {
		return false
	}

	if pattern.MessagePattern != "" && !strings.Contains(entry.Message, pattern.MessagePattern) {
		return false
	}

	for key, valueMatcher := range pattern.FieldMatchers {
		value, ok := entry.Fields[key]
		if !ok {
			return false
		}

		if !valueMatcher(value) {
			return false
		}
	}

	return true
}

// MatchPattern represents a match pattern
type MatchPattern struct {
	Level          string
	MessagePattern string
	FieldMatchers  map[string]func(interface{}) bool
}

// MatchResult represents a match result
type MatchResult struct {
	Index   int
	Entry   CapturedLogEntry
	Pattern MatchPattern
}

// TestFixture provides test fixtures
type TestFixture struct {
	mu       sync.RWMutex
	data     map[string]interface{}
	setup    []func()
	teardown []func()
}

// NewTestFixture creates a new test fixture
func NewTestFixture() *TestFixture {
	return &TestFixture{
		data:     make(map[string]interface{}),
		setup:    make([]func(), 0),
		teardown: make([]func(), 0),
	}
}

// AddSetup adds a setup function
func (tf *TestFixture) AddSetup(f func()) {
	tf.setup = append(tf.setup, f)
}

// AddTeardown adds a teardown function
func (tf *TestFixture) AddTeardown(f func()) {
	tf.teardown = append(tf.teardown, f)
}

// Setup runs all setup functions
func (tf *TestFixture) Setup() {
	for _, f := range tf.setup {
		f()
	}
}

// Teardown runs all teardown functions
func (tf *TestFixture) Teardown() {
	for i := len(tf.teardown) - 1; i >= 0; i-- {
		tf.teardown[i]()
	}
}

// Set sets fixture data
func (tf *TestFixture) Set(key string, value interface{}) {
	tf.mu.Lock()
	defer tf.mu.Unlock()
	tf.data[key] = value
}

// Get gets fixture data
func (tf *TestFixture) Get(key string) (interface{}, bool) {
	tf.mu.RLock()
	defer tf.mu.RUnlock()
	value, ok := tf.data[key]
	return value, ok
}

// MarshalJSON custom JSON marshaling for CapturedLogEntry
func (cle CapturedLogEntry) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"level":     cle.Level,
		"message":   cle.Message,
		"fields":    cle.Fields,
		"timestamp": cle.Timestamp,
		"caller":    cle.Caller,
	})
}
