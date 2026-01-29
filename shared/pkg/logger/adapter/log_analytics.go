package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

// LogAnalytics provides advanced log analytics capabilities
type LogAnalytics struct {
	mu                sync.RWMutex
	entries           []AnalyticsEntry
	patterns          map[string]*Pattern
	anomalies         []Anomaly
	statistics        *Statistics
	alertRules        []AlertRule
	patternDetector   *PatternDetector
	anomalyDetector   *AnomalyDetector
	trendAnalyzer     *TrendAnalyzer
	correlationEngine *CorrelationEngine
	reportGenerator   *ReportGenerator
}

// AnalyticsEntry represents a log entry for analytics
type AnalyticsEntry struct {
	Timestamp  time.Time
	Level      string
	Message    string
	Fields     map[string]interface{}
	Source     string
	Duration   time.Duration
	StatusCode int
	ErrorCode  string
	UserID     string
	SessionID  string
	TraceID    string
	Processed  bool
	Metadata   map[string]interface{}
}

// Pattern represents a detected pattern
type Pattern struct {
	ID          string
	Name        string
	Description string
	Regex       string
	Occurrences int
	FirstSeen   time.Time
	LastSeen    time.Time
	Examples    []string
	Severity    string
	Category    string
	Tags        []string
}

// Anomaly represents a detected anomaly
type Anomaly struct {
	ID          string
	Type        string
	Description string
	Severity    string
	DetectedAt  time.Time
	Entry       *AnalyticsEntry
	Score       float64
	Baseline    float64
	Deviation   float64
	Context     map[string]interface{}
}

// Statistics holds log statistics
type Statistics struct {
	mu                  sync.RWMutex
	TotalEntries        int64
	EntriesByLevel      map[string]int64
	EntriesBySource     map[string]int64
	EntriesByStatusCode map[int]int64
	EntriesByErrorCode  map[string]int64
	AverageDuration     time.Duration
	MaxDuration         time.Duration
	MinDuration         time.Duration
	ErrorRate           float64
	RequestRate         float64
	UniqueUsers         int
	UniqueSessions      int
	PeakTraffic         time.Time
	PeakTrafficCount    int64
	TimeSeriesData      map[string][]TimeSeriesPoint
	LastUpdate          time.Time
}

// TimeSeriesPoint represents a point in time series data
type TimeSeriesPoint struct {
	Timestamp time.Time
	Value     float64
	Label     string
}

// AlertRule represents an alert rule
type AlertRule struct {
	ID          string
	Name        string
	Description string
	Condition   string
	Threshold   float64
	Window      time.Duration
	Enabled     bool
	Actions     []AlertAction
	LastFired   time.Time
	FireCount   int
}

// AlertAction represents an action to take when an alert fires
type AlertAction struct {
	Type   string
	Config map[string]interface{}
}

// NewLogAnalytics creates a new log analytics instance
func NewLogAnalytics() *LogAnalytics {
	return &LogAnalytics{
		entries:           make([]AnalyticsEntry, 0),
		patterns:          make(map[string]*Pattern),
		anomalies:         make([]Anomaly, 0),
		statistics:        NewStatistics(),
		alertRules:        make([]AlertRule, 0),
		patternDetector:   NewPatternDetector(),
		anomalyDetector:   NewAnomalyDetector(),
		trendAnalyzer:     NewTrendAnalyzer(),
		correlationEngine: NewCorrelationEngine(),
		reportGenerator:   NewReportGenerator(),
	}
}

// NewStatistics creates a new statistics instance
func NewStatistics() *Statistics {
	return &Statistics{
		EntriesByLevel:      make(map[string]int64),
		EntriesBySource:     make(map[string]int64),
		EntriesByStatusCode: make(map[int]int64),
		EntriesByErrorCode:  make(map[string]int64),
		TimeSeriesData:      make(map[string][]TimeSeriesPoint),
		LastUpdate:          time.Now(),
	}
}

// AddEntry adds an entry for analysis
func (la *LogAnalytics) AddEntry(entry AnalyticsEntry) {
	la.mu.Lock()
	defer la.mu.Unlock()

	entry.Timestamp = time.Now()
	entry.Processed = false
	la.entries = append(la.entries, entry)

	// Update statistics
	la.updateStatistics(&entry)

	// Detect patterns
	if patterns := la.patternDetector.Detect(entry); len(patterns) > 0 {
		for _, pattern := range patterns {
			la.patterns[pattern.ID] = pattern
		}
	}

	// Detect anomalies
	if anomaly := la.anomalyDetector.Detect(entry, la.statistics); anomaly != nil {
		la.anomalies = append(la.anomalies, *anomaly)
	}

	// Check alert rules
	la.checkAlertRules(&entry)
}

// updateStatistics updates statistics with a new entry
func (la *LogAnalytics) updateStatistics(entry *AnalyticsEntry) {
	la.statistics.mu.Lock()
	defer la.statistics.mu.Unlock()

	la.statistics.TotalEntries++
	la.statistics.EntriesByLevel[entry.Level]++

	if entry.Source != "" {
		la.statistics.EntriesBySource[entry.Source]++
	}

	if entry.StatusCode != 0 {
		la.statistics.EntriesByStatusCode[entry.StatusCode]++
	}

	if entry.ErrorCode != "" {
		la.statistics.EntriesByErrorCode[entry.ErrorCode]++
	}

	// Update duration statistics
	if entry.Duration > 0 {
		if la.statistics.MaxDuration == 0 || entry.Duration > la.statistics.MaxDuration {
			la.statistics.MaxDuration = entry.Duration
		}
		if la.statistics.MinDuration == 0 || entry.Duration < la.statistics.MinDuration {
			la.statistics.MinDuration = entry.Duration
		}

		totalDuration := la.statistics.AverageDuration.Nanoseconds() * (la.statistics.TotalEntries - 1)
		la.statistics.AverageDuration = time.Duration((totalDuration + entry.Duration.Nanoseconds()) / la.statistics.TotalEntries)
	}

	// Update time series data
	bucket := entry.Timestamp.Truncate(time.Minute)
	seriesKey := fmt.Sprintf("entries_per_minute_%s", entry.Level)

	if _, exists := la.statistics.TimeSeriesData[seriesKey]; !exists {
		la.statistics.TimeSeriesData[seriesKey] = make([]TimeSeriesPoint, 0)
	}

	// Find or create time series point
	found := false
	for i := range la.statistics.TimeSeriesData[seriesKey] {
		if la.statistics.TimeSeriesData[seriesKey][i].Timestamp.Equal(bucket) {
			la.statistics.TimeSeriesData[seriesKey][i].Value++
			found = true
			break
		}
	}

	if !found {
		la.statistics.TimeSeriesData[seriesKey] = append(
			la.statistics.TimeSeriesData[seriesKey],
			TimeSeriesPoint{
				Timestamp: bucket,
				Value:     1,
				Label:     entry.Level,
			},
		)
	}

	la.statistics.LastUpdate = time.Now()
}

// checkAlertRules checks if any alert rules should fire
func (la *LogAnalytics) checkAlertRules(entry *AnalyticsEntry) {
	for i := range la.alertRules {
		rule := &la.alertRules[i]
		if !rule.Enabled {
			continue
		}

		if la.evaluateAlertCondition(rule, entry) {
			rule.LastFired = time.Now()
			rule.FireCount++
			la.executeAlertActions(rule, entry)
		}
	}
}

// evaluateAlertCondition evaluates an alert condition
func (la *LogAnalytics) evaluateAlertCondition(rule *AlertRule, entry *AnalyticsEntry) bool {
	// Simple condition evaluation
	switch rule.Condition {
	case "error_rate_high":
		errorCount := la.statistics.EntriesByLevel["ERROR"]
		totalCount := la.statistics.TotalEntries
		if totalCount > 0 {
			errorRate := float64(errorCount) / float64(totalCount) * 100
			return errorRate > rule.Threshold
		}
	case "duration_high":
		if entry.Duration > 0 {
			return entry.Duration.Seconds() > rule.Threshold
		}
	case "status_code_error":
		return entry.StatusCode >= 500
	}
	return false
}

// executeAlertActions executes alert actions
func (la *LogAnalytics) executeAlertActions(rule *AlertRule, entry *AnalyticsEntry) {
	for _, action := range rule.Actions {
		switch action.Type {
		case "log":
			// Log the alert
		case "email":
			// Send email (placeholder)
		case "webhook":
			// Call webhook (placeholder)
		}
	}
}

// GetStatistics returns current statistics
func (la *LogAnalytics) GetStatistics() Statistics {
	la.statistics.mu.RLock()
	defer la.statistics.mu.RUnlock()
	return *la.statistics
}

// GetPatterns returns detected patterns
func (la *LogAnalytics) GetPatterns() []*Pattern {
	la.mu.RLock()
	defer la.mu.RUnlock()

	patterns := make([]*Pattern, 0, len(la.patterns))
	for _, pattern := range la.patterns {
		patterns = append(patterns, pattern)
	}
	return patterns
}

// GetAnomalies returns detected anomalies
func (la *LogAnalytics) GetAnomalies() []Anomaly {
	la.mu.RLock()
	defer la.mu.RUnlock()

	anomalies := make([]Anomaly, len(la.anomalies))
	copy(anomalies, la.anomalies)
	return anomalies
}

// GenerateReport generates an analytics report
func (la *LogAnalytics) GenerateReport(reportType string, timeRange TimeRange) *Report {
	la.mu.RLock()
	defer la.mu.RUnlock()

	return la.reportGenerator.Generate(reportType, la, timeRange)
}

// AnalyzeTrends analyzes trends in the data
func (la *LogAnalytics) AnalyzeTrends(metric string, window time.Duration) *TrendAnalysis {
	la.mu.RLock()
	defer la.mu.RUnlock()

	return la.trendAnalyzer.Analyze(metric, la.entries, window)
}

// FindCorrelations finds correlations between events
func (la *LogAnalytics) FindCorrelations(eventType1, eventType2 string, window time.Duration) []Correlation {
	la.mu.RLock()
	defer la.mu.RUnlock()

	return la.correlationEngine.FindCorrelations(eventType1, eventType2, la.entries, window)
}

// PatternDetector detects patterns in log entries
type PatternDetector struct {
	mu       sync.RWMutex
	patterns map[string]*Pattern
}

// NewPatternDetector creates a new pattern detector
func NewPatternDetector() *PatternDetector {
	return &PatternDetector{
		patterns: make(map[string]*Pattern),
	}
}

// Detect detects patterns in an entry
func (pd *PatternDetector) Detect(entry AnalyticsEntry) []*Pattern {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	detected := make([]*Pattern, 0)

	// Simple pattern detection based on message content
	if strings.Contains(entry.Message, "timeout") {
		pattern := pd.getOrCreatePattern("timeout", "Timeout Pattern", "Requests timing out")
		pattern.Occurrences++
		pattern.LastSeen = time.Now()
		if len(pattern.Examples) < 10 {
			pattern.Examples = append(pattern.Examples, entry.Message)
		}
		detected = append(detected, pattern)
	}

	if strings.Contains(entry.Message, "connection refused") {
		pattern := pd.getOrCreatePattern("connection_refused", "Connection Refused Pattern", "Connection attempts being refused")
		pattern.Occurrences++
		pattern.LastSeen = time.Now()
		if len(pattern.Examples) < 10 {
			pattern.Examples = append(pattern.Examples, entry.Message)
		}
		detected = append(detected, pattern)
	}

	if entry.StatusCode >= 500 {
		pattern := pd.getOrCreatePattern("server_error", "Server Error Pattern", "5xx server errors")
		pattern.Occurrences++
		pattern.LastSeen = time.Now()
		if len(pattern.Examples) < 10 {
			pattern.Examples = append(pattern.Examples, entry.Message)
		}
		detected = append(detected, pattern)
	}

	return detected
}

// getOrCreatePattern gets or creates a pattern
func (pd *PatternDetector) getOrCreatePattern(id, name, description string) *Pattern {
	if pattern, exists := pd.patterns[id]; exists {
		return pattern
	}

	pattern := &Pattern{
		ID:          id,
		Name:        name,
		Description: description,
		FirstSeen:   time.Now(),
		LastSeen:    time.Now(),
		Examples:    make([]string, 0),
		Tags:        make([]string, 0),
	}
	pd.patterns[id] = pattern
	return pattern
}

// AnomalyDetector detects anomalies in log entries
type AnomalyDetector struct {
	mu        sync.RWMutex
	baselines map[string]float64
	threshold float64
}

// NewAnomalyDetector creates a new anomaly detector
func NewAnomalyDetector() *AnomalyDetector {
	return &AnomalyDetector{
		baselines: make(map[string]float64),
		threshold: 2.0, // Standard deviations from mean
	}
}

// Detect detects anomalies in an entry
func (ad *AnomalyDetector) Detect(entry AnalyticsEntry, stats *Statistics) *Anomaly {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	// Check for duration anomalies
	if entry.Duration > 0 {
		avgDuration := stats.AverageDuration.Seconds()
		currentDuration := entry.Duration.Seconds()

		if avgDuration > 0 {
			deviation := math.Abs(currentDuration-avgDuration) / avgDuration

			if deviation > ad.threshold {
				return &Anomaly{
					ID:          fmt.Sprintf("anomaly-%d", time.Now().UnixNano()),
					Type:        "duration",
					Description: fmt.Sprintf("Duration significantly different from baseline: %.2fs vs %.2fs", currentDuration, avgDuration),
					Severity:    ad.calculateSeverity(deviation),
					DetectedAt:  time.Now(),
					Entry:       &entry,
					Score:       deviation,
					Baseline:    avgDuration,
					Deviation:   deviation,
					Context:     map[string]interface{}{"metric": "duration"},
				}
			}
		}
	}

	// Check for error rate anomalies
	stats.mu.RLock()
	errorCount := stats.EntriesByLevel["ERROR"]
	totalCount := stats.TotalEntries
	stats.mu.RUnlock()

	if totalCount > 0 {
		errorRate := float64(errorCount) / float64(totalCount)
		if baseline, exists := ad.baselines["error_rate"]; exists {
			deviation := math.Abs(errorRate-baseline) / baseline

			if deviation > ad.threshold && errorRate > baseline {
				return &Anomaly{
					ID:          fmt.Sprintf("anomaly-%d", time.Now().UnixNano()),
					Type:        "error_rate",
					Description: fmt.Sprintf("Error rate spike detected: %.2f%% vs baseline %.2f%%", errorRate*100, baseline*100),
					Severity:    ad.calculateSeverity(deviation),
					DetectedAt:  time.Now(),
					Entry:       &entry,
					Score:       deviation,
					Baseline:    baseline,
					Deviation:   deviation,
					Context:     map[string]interface{}{"metric": "error_rate"},
				}
			}
		} else {
			ad.baselines["error_rate"] = errorRate
		}
	}

	return nil
}

// calculateSeverity calculates severity based on deviation
func (ad *AnomalyDetector) calculateSeverity(deviation float64) string {
	if deviation > 5.0 {
		return "critical"
	} else if deviation > 3.0 {
		return "high"
	} else if deviation > 2.0 {
		return "medium"
	}
	return "low"
}

// TrendAnalyzer analyzes trends in metrics
type TrendAnalyzer struct {
	mu sync.RWMutex
}

// NewTrendAnalyzer creates a new trend analyzer
func NewTrendAnalyzer() *TrendAnalyzer {
	return &TrendAnalyzer{}
}

// TrendAnalysis represents a trend analysis result
type TrendAnalysis struct {
	Metric     string
	Trend      string
	Slope      float64
	Confidence float64
	DataPoints []TrendDataPoint
	StartTime  time.Time
	EndTime    time.Time
}

// TrendDataPoint represents a data point in trend analysis
type TrendDataPoint struct {
	Timestamp time.Time
	Value     float64
}

// Analyze analyzes trends in a metric
func (ta *TrendAnalyzer) Analyze(metric string, entries []AnalyticsEntry, window time.Duration) *TrendAnalysis {
	ta.mu.Lock()
	defer ta.mu.Unlock()

	now := time.Now()
	startTime := now.Add(-window)

	dataPoints := make([]TrendDataPoint, 0)

	// Extract data points based on metric
	for _, entry := range entries {
		if entry.Timestamp.Before(startTime) {
			continue
		}

		var value float64
		switch metric {
		case "duration":
			value = entry.Duration.Seconds()
		case "error_count":
			if entry.Level == "ERROR" {
				value = 1
			}
		case "request_count":
			value = 1
		}

		dataPoints = append(dataPoints, TrendDataPoint{
			Timestamp: entry.Timestamp,
			Value:     value,
		})
	}

	if len(dataPoints) < 2 {
		return &TrendAnalysis{
			Metric:     metric,
			Trend:      "insufficient_data",
			DataPoints: dataPoints,
			StartTime:  startTime,
			EndTime:    now,
		}
	}

	// Calculate linear regression
	slope, _ := ta.calculateLinearRegression(dataPoints)

	trend := "stable"
	if slope > 0.1 {
		trend = "increasing"
	} else if slope < -0.1 {
		trend = "decreasing"
	}

	return &TrendAnalysis{
		Metric:     metric,
		Trend:      trend,
		Slope:      slope,
		Confidence: 0.8, // Placeholder
		DataPoints: dataPoints,
		StartTime:  startTime,
		EndTime:    now,
	}
}

// calculateLinearRegression calculates linear regression for trend analysis
func (ta *TrendAnalyzer) calculateLinearRegression(points []TrendDataPoint) (float64, float64) {
	if len(points) < 2 {
		return 0, 0
	}

	n := float64(len(points))
	var sumX, sumY, sumXY, sumXX float64

	for i, point := range points {
		x := float64(i)
		y := point.Value
		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}

	slope := (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)
	intercept := (sumY - slope*sumX) / n

	return slope, intercept
}

// CorrelationEngine finds correlations between events
type CorrelationEngine struct {
	mu sync.RWMutex
}

// NewCorrelationEngine creates a new correlation engine
func NewCorrelationEngine() *CorrelationEngine {
	return &CorrelationEngine{}
}

// Correlation represents a correlation between events
type Correlation struct {
	Event1       string
	Event2       string
	Strength     float64
	Count        int
	AverageDelay time.Duration
	Examples     []CorrelationExample
}

// CorrelationExample represents an example of correlated events
type CorrelationExample struct {
	Event1Time time.Time
	Event2Time time.Time
	Delay      time.Duration
	Context    map[string]interface{}
}

// FindCorrelations finds correlations between two event types
func (ce *CorrelationEngine) FindCorrelations(eventType1, eventType2 string, entries []AnalyticsEntry, window time.Duration) []Correlation {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	correlations := make(map[string]*Correlation)

	// Find events of type 1
	event1Entries := make([]AnalyticsEntry, 0)
	for _, entry := range entries {
		if ce.matchesEventType(entry, eventType1) {
			event1Entries = append(event1Entries, entry)
		}
	}

	// Find events of type 2
	event2Entries := make([]AnalyticsEntry, 0)
	for _, entry := range entries {
		if ce.matchesEventType(entry, eventType2) {
			event2Entries = append(event2Entries, entry)
		}
	}

	// Find correlations
	for _, e1 := range event1Entries {
		for _, e2 := range event2Entries {
			if e2.Timestamp.After(e1.Timestamp) {
				delay := e2.Timestamp.Sub(e1.Timestamp)
				if delay <= window {
					key := fmt.Sprintf("%s-%s", eventType1, eventType2)

					if _, exists := correlations[key]; !exists {
						correlations[key] = &Correlation{
							Event1:   eventType1,
							Event2:   eventType2,
							Examples: make([]CorrelationExample, 0),
						}
					}

					corr := correlations[key]
					corr.Count++
					corr.AverageDelay = (corr.AverageDelay*time.Duration(corr.Count-1) + delay) / time.Duration(corr.Count)

					if len(corr.Examples) < 10 {
						corr.Examples = append(corr.Examples, CorrelationExample{
							Event1Time: e1.Timestamp,
							Event2Time: e2.Timestamp,
							Delay:      delay,
						})
					}
				}
			}
		}
	}

	// Calculate correlation strength
	result := make([]Correlation, 0, len(correlations))
	for _, corr := range correlations {
		corr.Strength = float64(corr.Count) / float64(len(event1Entries))
		result = append(result, *corr)
	}

	// Sort by strength
	sort.Slice(result, func(i, j int) bool {
		return result[i].Strength > result[j].Strength
	})

	return result
}

// matchesEventType checks if an entry matches an event type
func (ce *CorrelationEngine) matchesEventType(entry AnalyticsEntry, eventType string) bool {
	return strings.Contains(strings.ToLower(entry.Message), strings.ToLower(eventType))
}

// ReportGenerator generates analytics reports
type ReportGenerator struct {
	mu sync.RWMutex
}

// NewReportGenerator creates a new report generator
func NewReportGenerator() *ReportGenerator {
	return &ReportGenerator{}
}

// Report represents an analytics report
type Report struct {
	Title       string
	GeneratedAt time.Time
	TimeRange   TimeRange
	Sections    []ReportSection
	Summary     map[string]interface{}
	Charts      []Chart
}

// TimeRange represents a time range for analysis
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// ReportSection represents a section of a report
type ReportSection struct {
	Title   string
	Content string
	Data    map[string]interface{}
	Type    string
}

// Chart represents a chart in a report
type Chart struct {
	Type   string
	Title  string
	Data   []ChartDataPoint
	Labels []string
}

// ChartDataPoint represents a data point in a chart
type ChartDataPoint struct {
	Label string
	Value float64
	Color string
}

// Generate generates a report
func (rg *ReportGenerator) Generate(reportType string, analytics *LogAnalytics, timeRange TimeRange) *Report {
	rg.mu.Lock()
	defer rg.mu.Unlock()

	report := &Report{
		Title:       fmt.Sprintf("%s Report", reportType),
		GeneratedAt: time.Now(),
		TimeRange:   timeRange,
		Sections:    make([]ReportSection, 0),
		Summary:     make(map[string]interface{}),
		Charts:      make([]Chart, 0),
	}

	stats := analytics.GetStatistics()

	// Add summary section
	report.Summary = map[string]interface{}{
		"total_entries":    stats.TotalEntries,
		"error_rate":       fmt.Sprintf("%.2f%%", float64(stats.EntriesByLevel["ERROR"])/float64(stats.TotalEntries)*100),
		"average_duration": stats.AverageDuration.String(),
		"unique_users":     stats.UniqueUsers,
		"time_range":       fmt.Sprintf("%s to %s", timeRange.Start.Format(time.RFC3339), timeRange.End.Format(time.RFC3339)),
	}

	// Add log level distribution chart
	levelChart := Chart{
		Type:   "pie",
		Title:  "Log Level Distribution",
		Data:   make([]ChartDataPoint, 0),
		Labels: make([]string, 0),
	}

	for level, count := range stats.EntriesByLevel {
		levelChart.Data = append(levelChart.Data, ChartDataPoint{
			Label: level,
			Value: float64(count),
		})
		levelChart.Labels = append(levelChart.Labels, level)
	}
	report.Charts = append(report.Charts, levelChart)

	// Add patterns section
	patterns := analytics.GetPatterns()
	if len(patterns) > 0 {
		patternsData := make(map[string]interface{})
		for _, pattern := range patterns {
			patternsData[pattern.ID] = map[string]interface{}{
				"name":        pattern.Name,
				"occurrences": pattern.Occurrences,
				"first_seen":  pattern.FirstSeen,
				"last_seen":   pattern.LastSeen,
			}
		}

		report.Sections = append(report.Sections, ReportSection{
			Title:   "Detected Patterns",
			Content: fmt.Sprintf("Found %d patterns in the logs", len(patterns)),
			Data:    patternsData,
			Type:    "patterns",
		})
	}

	// Add anomalies section
	anomalies := analytics.GetAnomalies()
	if len(anomalies) > 0 {
		anomaliesData := make(map[string]interface{})
		for i, anomaly := range anomalies {
			anomaliesData[fmt.Sprintf("anomaly_%d", i)] = map[string]interface{}{
				"type":        anomaly.Type,
				"description": anomaly.Description,
				"severity":    anomaly.Severity,
				"detected_at": anomaly.DetectedAt,
				"score":       anomaly.Score,
			}
		}

		report.Sections = append(report.Sections, ReportSection{
			Title:   "Detected Anomalies",
			Content: fmt.Sprintf("Found %d anomalies in the logs", len(anomalies)),
			Data:    anomaliesData,
			Type:    "anomalies",
		})
	}

	return report
}

// ExportReport exports a report to a format
func (rg *ReportGenerator) ExportReport(report *Report, format string) ([]byte, error) {
	switch format {
	case "json":
		return json.MarshalIndent(report, "", "  ")
	case "text":
		return []byte(rg.formatReportAsText(report)), nil
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// formatReportAsText formats a report as text
func (rg *ReportGenerator) formatReportAsText(report *Report) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("=== %s ===\n", report.Title))
	sb.WriteString(fmt.Sprintf("Generated: %s\n", report.GeneratedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("Time Range: %s to %s\n\n", report.TimeRange.Start.Format(time.RFC3339), report.TimeRange.End.Format(time.RFC3339)))

	sb.WriteString("Summary:\n")
	for key, value := range report.Summary {
		sb.WriteString(fmt.Sprintf("  %s: %v\n", key, value))
	}
	sb.WriteString("\n")

	for _, section := range report.Sections {
		sb.WriteString(fmt.Sprintf("--- %s ---\n", section.Title))
		sb.WriteString(fmt.Sprintf("%s\n\n", section.Content))
	}

	return sb.String()
}

// MarshalJSON custom JSON marshaling for TimeSeriesPoint
func (tsp TimeSeriesPoint) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"timestamp": tsp.Timestamp,
		"value":     tsp.Value,
		"label":     tsp.Label,
	})
}

// MarshalJSON custom JSON marshaling for Pattern
func (p Pattern) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"id":          p.ID,
		"name":        p.Name,
		"description": p.Description,
		"occurrences": p.Occurrences,
		"first_seen":  p.FirstSeen,
		"last_seen":   p.LastSeen,
		"severity":    p.Severity,
		"category":    p.Category,
		"tags":        p.Tags,
	})
}

// MarshalJSON custom JSON marshaling for Anomaly
func (a Anomaly) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"id":          a.ID,
		"type":        a.Type,
		"description": a.Description,
		"severity":    a.Severity,
		"detected_at": a.DetectedAt,
		"score":       a.Score,
		"baseline":    a.Baseline,
		"deviation":   a.Deviation,
		"context":     a.Context,
	})
}

// QueryEngine provides advanced querying capabilities
type QueryEngine struct {
	mu      sync.RWMutex
	indexes map[string]map[string][]int
}

// NewQueryEngine creates a new query engine
func NewQueryEngine() *QueryEngine {
	return &QueryEngine{
		indexes: make(map[string]map[string][]int),
	}
}

// Query executes a query on log entries
func (qe *QueryEngine) Query(ctx context.Context, entries []AnalyticsEntry, conditions []QueryCondition) []AnalyticsEntry {
	qe.mu.RLock()
	defer qe.mu.RUnlock()

	results := make([]AnalyticsEntry, 0)

	for i, entry := range entries {
		if qe.matchesConditions(entry, conditions) {
			results = append(results, entries[i])
		}
	}

	return results
}

// QueryCondition represents a query condition
type QueryCondition struct {
	Field    string
	Operator string
	Value    interface{}
}

// matchesConditions checks if an entry matches all conditions
func (qe *QueryEngine) matchesConditions(entry AnalyticsEntry, conditions []QueryCondition) bool {
	for _, condition := range conditions {
		if !qe.matchesCondition(entry, condition) {
			return false
		}
	}
	return true
}

// matchesCondition checks if an entry matches a condition
func (qe *QueryEngine) matchesCondition(entry AnalyticsEntry, condition QueryCondition) bool {
	var fieldValue interface{}

	switch condition.Field {
	case "level":
		fieldValue = entry.Level
	case "message":
		fieldValue = entry.Message
	case "source":
		fieldValue = entry.Source
	case "status_code":
		fieldValue = entry.StatusCode
	case "error_code":
		fieldValue = entry.ErrorCode
	default:
		if val, exists := entry.Fields[condition.Field]; exists {
			fieldValue = val
		}
	}

	switch condition.Operator {
	case "equals":
		return fmt.Sprintf("%v", fieldValue) == fmt.Sprintf("%v", condition.Value)
	case "contains":
		return strings.Contains(fmt.Sprintf("%v", fieldValue), fmt.Sprintf("%v", condition.Value))
	case "gt":
		if fv, ok := fieldValue.(float64); ok {
			if cv, ok := condition.Value.(float64); ok {
				return fv > cv
			}
		}
	case "lt":
		if fv, ok := fieldValue.(float64); ok {
			if cv, ok := condition.Value.(float64); ok {
				return fv < cv
			}
		}
	}

	return false
}

// BuildIndex builds an index for faster querying
func (qe *QueryEngine) BuildIndex(entries []AnalyticsEntry, field string) {
	qe.mu.Lock()
	defer qe.mu.Unlock()

	if _, exists := qe.indexes[field]; !exists {
		qe.indexes[field] = make(map[string][]int)
	}

	for i, entry := range entries {
		var value string
		switch field {
		case "level":
			value = entry.Level
		case "source":
			value = entry.Source
		case "error_code":
			value = entry.ErrorCode
		}

		qe.indexes[field][value] = append(qe.indexes[field][value], i)
	}
}
