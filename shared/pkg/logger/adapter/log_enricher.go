package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"shared/pkg/logger"
)

// LogEnricher adds contextual information to log entries
type LogEnricher struct {
	hostname       string
	processID      int
	appVersion     string
	environment    string
	region         string
	datacenter     string
	instanceID     string
	machineID      string
	globalTags     map[string]interface{}
	mu             sync.RWMutex
	middleware     []EnrichmentMiddleware
	correlationMgr *CorrelationManager
}

// EnrichmentMiddleware is a function that enriches log fields
type EnrichmentMiddleware func(fields []logger.Field) []logger.Field

// NewLogEnricher creates a new log enricher
func NewLogEnricher(appVersion, environment string) *LogEnricher {
	hostname, _ := os.Hostname()

	return &LogEnricher{
		hostname:       hostname,
		processID:      os.Getpid(),
		appVersion:     appVersion,
		environment:    environment,
		globalTags:     make(map[string]interface{}),
		middleware:     make([]EnrichmentMiddleware, 0),
		correlationMgr: NewCorrelationManager(),
	}
}

// SetRegion sets the region
func (le *LogEnricher) SetRegion(region string) {
	le.mu.Lock()
	defer le.mu.Unlock()
	le.region = region
}

// SetDatacenter sets the datacenter
func (le *LogEnricher) SetDatacenter(datacenter string) {
	le.mu.Lock()
	defer le.mu.Unlock()
	le.datacenter = datacenter
}

// SetInstanceID sets the instance ID
func (le *LogEnricher) SetInstanceID(instanceID string) {
	le.mu.Lock()
	defer le.mu.Unlock()
	le.instanceID = instanceID
}

// SetMachineID sets the machine ID
func (le *LogEnricher) SetMachineID(machineID string) {
	le.mu.Lock()
	defer le.mu.Unlock()
	le.machineID = machineID
}

// AddGlobalTag adds a global tag to all log entries
func (le *LogEnricher) AddGlobalTag(key string, value interface{}) {
	le.mu.Lock()
	defer le.mu.Unlock()
	le.globalTags[key] = value
}

// RemoveGlobalTag removes a global tag
func (le *LogEnricher) RemoveGlobalTag(key string) {
	le.mu.Lock()
	defer le.mu.Unlock()
	delete(le.globalTags, key)
}

// AddMiddleware adds an enrichment middleware
func (le *LogEnricher) AddMiddleware(middleware EnrichmentMiddleware) {
	le.mu.Lock()
	defer le.mu.Unlock()
	le.middleware = append(le.middleware, middleware)
}

// Enrich enriches log fields with contextual information
func (le *LogEnricher) Enrich(ctx context.Context, fields []logger.Field) []logger.Field {
	le.mu.RLock()

	enriched := make([]logger.Field, 0, len(fields)+20)

	// Add system information
	enriched = append(enriched,
		logger.String("hostname", le.hostname),
		logger.Int("pid", le.processID),
		logger.String("app_version", le.appVersion),
		logger.String("environment", le.environment),
	)

	if le.region != "" {
		enriched = append(enriched, logger.String("region", le.region))
	}
	if le.datacenter != "" {
		enriched = append(enriched, logger.String("datacenter", le.datacenter))
	}
	if le.instanceID != "" {
		enriched = append(enriched, logger.String("instance_id", le.instanceID))
	}
	if le.machineID != "" {
		enriched = append(enriched, logger.String("machine_id", le.machineID))
	}

	// Add global tags
	for key, value := range le.globalTags {
		enriched = append(enriched, logger.Any(key, value))
	}

	le.mu.RUnlock()

	// Add runtime information
	enriched = append(enriched,
		logger.Int("goroutines", runtime.NumGoroutine()),
		logger.String("go_version", runtime.Version()),
	)

	// Add memory statistics
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	enriched = append(enriched,
		logger.Uint64("mem_alloc_mb", m.Alloc/1024/1024),
		logger.Uint64("mem_total_mb", m.TotalAlloc/1024/1024),
		logger.Uint64("mem_sys_mb", m.Sys/1024/1024),
		logger.Uint32("mem_num_gc", m.NumGC),
	)

	// Add context information
	if ctx != nil {
		if correlationID := le.correlationMgr.GetCorrelationID(ctx); correlationID != "" {
			enriched = append(enriched, logger.String("correlation_id", correlationID))
		}
		if requestID := le.extractRequestID(ctx); requestID != "" {
			enriched = append(enriched, logger.String("request_id", requestID))
		}
		if userID := le.extractUserID(ctx); userID != "" {
			enriched = append(enriched, logger.String("user_id", userID))
		}
		if sessionID := le.extractSessionID(ctx); sessionID != "" {
			enriched = append(enriched, logger.String("session_id", sessionID))
		}
	}

	// Add original fields
	enriched = append(enriched, fields...)

	// Apply middleware
	le.mu.RLock()
	for _, middleware := range le.middleware {
		enriched = middleware(enriched)
	}
	le.mu.RUnlock()

	return enriched
}

// extractRequestID extracts request ID from context
func (le *LogEnricher) extractRequestID(ctx context.Context) string {
	if val := ctx.Value("request_id"); val != nil {
		if id, ok := val.(string); ok {
			return id
		}
	}
	return ""
}

// extractUserID extracts user ID from context
func (le *LogEnricher) extractUserID(ctx context.Context) string {
	if val := ctx.Value("user_id"); val != nil {
		if id, ok := val.(string); ok {
			return id
		}
	}
	return ""
}

// extractSessionID extracts session ID from context
func (le *LogEnricher) extractSessionID(ctx context.Context) string {
	if val := ctx.Value("session_id"); val != nil {
		if id, ok := val.(string); ok {
			return id
		}
	}
	return ""
}

// GetSystemInfo returns system information as a map
func (le *LogEnricher) GetSystemInfo() map[string]interface{} {
	le.mu.RLock()
	defer le.mu.RUnlock()

	info := map[string]interface{}{
		"hostname":    le.hostname,
		"pid":         le.processID,
		"app_version": le.appVersion,
		"environment": le.environment,
		"go_version":  runtime.Version(),
		"os":          runtime.GOOS,
		"arch":        runtime.GOARCH,
		"num_cpu":     runtime.NumCPU(),
		"goroutines":  runtime.NumGoroutine(),
	}

	if le.region != "" {
		info["region"] = le.region
	}
	if le.datacenter != "" {
		info["datacenter"] = le.datacenter
	}
	if le.instanceID != "" {
		info["instance_id"] = le.instanceID
	}
	if le.machineID != "" {
		info["machine_id"] = le.machineID
	}

	return info
}

// CorrelationManager manages correlation IDs across requests
type CorrelationManager struct {
	mu    sync.RWMutex
	store map[context.Context]string
}

// NewCorrelationManager creates a new correlation manager
func NewCorrelationManager() *CorrelationManager {
	return &CorrelationManager{
		store: make(map[context.Context]string),
	}
}

// SetCorrelationID sets a correlation ID for a context
func (cm *CorrelationManager) SetCorrelationID(ctx context.Context, correlationID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.store[ctx] = correlationID
}

// GetCorrelationID gets the correlation ID for a context
func (cm *CorrelationManager) GetCorrelationID(ctx context.Context) string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.store[ctx]
}

// GenerateCorrelationID generates a new correlation ID
func (cm *CorrelationManager) GenerateCorrelationID() string {
	return fmt.Sprintf("corr-%d-%d", time.Now().UnixNano(), os.Getpid())
}

// Clear clears the correlation store
func (cm *CorrelationManager) Clear() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.store = make(map[context.Context]string)
}

// MetadataManager manages metadata for log entries
type MetadataManager struct {
	mu       sync.RWMutex
	metadata map[string]map[string]interface{}
}

// NewMetadataManager creates a new metadata manager
func NewMetadataManager() *MetadataManager {
	return &MetadataManager{
		metadata: make(map[string]map[string]interface{}),
	}
}

// SetMetadata sets metadata for a namespace
func (mm *MetadataManager) SetMetadata(namespace string, key string, value interface{}) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if mm.metadata[namespace] == nil {
		mm.metadata[namespace] = make(map[string]interface{})
	}
	mm.metadata[namespace][key] = value
}

// GetMetadata gets metadata for a namespace
func (mm *MetadataManager) GetMetadata(namespace string) map[string]interface{} {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	if metadata, ok := mm.metadata[namespace]; ok {
		// Return a copy
		copy := make(map[string]interface{})
		for k, v := range metadata {
			copy[k] = v
		}
		return copy
	}
	return nil
}

// GetAllMetadata gets all metadata
func (mm *MetadataManager) GetAllMetadata() map[string]map[string]interface{} {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	// Return a deep copy
	copy := make(map[string]map[string]interface{})
	for namespace, metadata := range mm.metadata {
		copy[namespace] = make(map[string]interface{})
		for k, v := range metadata {
			copy[namespace][k] = v
		}
	}
	return copy
}

// DeleteMetadata deletes metadata for a namespace
func (mm *MetadataManager) DeleteMetadata(namespace string, key string) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if metadata, ok := mm.metadata[namespace]; ok {
		delete(metadata, key)
		if len(metadata) == 0 {
			delete(mm.metadata, namespace)
		}
	}
}

// ClearNamespace clears all metadata for a namespace
func (mm *MetadataManager) ClearNamespace(namespace string) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	delete(mm.metadata, namespace)
}

// TagManager manages tags for log entries
type TagManager struct {
	mu   sync.RWMutex
	tags map[string][]string
}

// NewTagManager creates a new tag manager
func NewTagManager() *TagManager {
	return &TagManager{
		tags: make(map[string][]string),
	}
}

// AddTag adds a tag to a category
func (tm *TagManager) AddTag(category, tag string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.tags[category] == nil {
		tm.tags[category] = make([]string, 0)
	}

	// Check if tag already exists
	for _, existingTag := range tm.tags[category] {
		if existingTag == tag {
			return
		}
	}

	tm.tags[category] = append(tm.tags[category], tag)
}

// RemoveTag removes a tag from a category
func (tm *TagManager) RemoveTag(category, tag string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tags, ok := tm.tags[category]; ok {
		for i, existingTag := range tags {
			if existingTag == tag {
				tm.tags[category] = append(tags[:i], tags[i+1:]...)
				break
			}
		}

		if len(tm.tags[category]) == 0 {
			delete(tm.tags, category)
		}
	}
}

// GetTags gets all tags for a category
func (tm *TagManager) GetTags(category string) []string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if tags, ok := tm.tags[category]; ok {
		// Return a copy
		copy := make([]string, len(tags))
		for i, tag := range tags {
			copy[i] = tag
		}
		return copy
	}
	return nil
}

// GetAllTags gets all tags
func (tm *TagManager) GetAllTags() map[string][]string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	// Return a deep copy
	copy := make(map[string][]string)
	for category, tags := range tm.tags {
		copy[category] = make([]string, len(tags))
		for i, tag := range tags {
			copy[category][i] = tag
		}
	}
	return copy
}

// HasTag checks if a tag exists in a category
func (tm *TagManager) HasTag(category, tag string) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if tags, ok := tm.tags[category]; ok {
		for _, existingTag := range tags {
			if existingTag == tag {
				return true
			}
		}
	}
	return false
}

// ClearCategory clears all tags for a category
func (tm *TagManager) ClearCategory(category string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	delete(tm.tags, category)
}

// ContextEnricher enriches context with logging metadata
type ContextEnricher struct {
	metadataManager *MetadataManager
	tagManager      *TagManager
	correlationMgr  *CorrelationManager
}

// NewContextEnricher creates a new context enricher
func NewContextEnricher() *ContextEnricher {
	return &ContextEnricher{
		metadataManager: NewMetadataManager(),
		tagManager:      NewTagManager(),
		correlationMgr:  NewCorrelationManager(),
	}
}

// WithMetadata adds metadata to context
func (ce *ContextEnricher) WithMetadata(ctx context.Context, namespace string, key string, value interface{}) context.Context {
	ce.metadataManager.SetMetadata(namespace, key, value)
	return ctx
}

// WithTag adds a tag to context
func (ce *ContextEnricher) WithTag(ctx context.Context, category, tag string) context.Context {
	ce.tagManager.AddTag(category, tag)
	return ctx
}

// WithCorrelationID adds a correlation ID to context
func (ce *ContextEnricher) WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	ce.correlationMgr.SetCorrelationID(ctx, correlationID)
	return context.WithValue(ctx, "correlation_id", correlationID)
}

// ExtractEnrichment extracts all enrichment data from context
func (ce *ContextEnricher) ExtractEnrichment() map[string]interface{} {
	return map[string]interface{}{
		"metadata": ce.metadataManager.GetAllMetadata(),
		"tags":     ce.tagManager.GetAllTags(),
	}
}

// FilterManager manages log filtering rules
type FilterManager struct {
	mu      sync.RWMutex
	filters map[string]FilterRule
}

// FilterRule represents a log filtering rule
type FilterRule struct {
	Name        string
	Field       string
	Operator    string
	Value       interface{}
	Action      string
	Priority    int
	Enabled     bool
	Description string
}

// NewFilterManager creates a new filter manager
func NewFilterManager() *FilterManager {
	return &FilterManager{
		filters: make(map[string]FilterRule),
	}
}

// AddFilter adds a filter rule
func (fm *FilterManager) AddFilter(rule FilterRule) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.filters[rule.Name] = rule
}

// RemoveFilter removes a filter rule
func (fm *FilterManager) RemoveFilter(name string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	delete(fm.filters, name)
}

// GetFilter gets a filter rule
func (fm *FilterManager) GetFilter(name string) (FilterRule, bool) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	rule, ok := fm.filters[name]
	return rule, ok
}

// GetAllFilters gets all filter rules
func (fm *FilterManager) GetAllFilters() []FilterRule {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	rules := make([]FilterRule, 0, len(fm.filters))
	for _, rule := range fm.filters {
		rules = append(rules, rule)
	}
	return rules
}

// EnableFilter enables a filter rule
func (fm *FilterManager) EnableFilter(name string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if rule, ok := fm.filters[name]; ok {
		rule.Enabled = true
		fm.filters[name] = rule
	}
}

// DisableFilter disables a filter rule
func (fm *FilterManager) DisableFilter(name string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if rule, ok := fm.filters[name]; ok {
		rule.Enabled = false
		fm.filters[name] = rule
	}
}

// ShouldLog determines if a log entry should be logged based on filters
func (fm *FilterManager) ShouldLog(fields map[string]interface{}) bool {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	for _, rule := range fm.filters {
		if !rule.Enabled {
			continue
		}

		if fieldValue, ok := fields[rule.Field]; ok {
			matched := fm.evaluateFilter(rule, fieldValue)

			if matched {
				switch rule.Action {
				case "drop":
					return false
				case "allow":
					return true
				}
			}
		}
	}

	return true
}

// evaluateFilter evaluates a filter rule
func (fm *FilterManager) evaluateFilter(rule FilterRule, fieldValue interface{}) bool {
	switch rule.Operator {
	case "equals":
		return fmt.Sprintf("%v", fieldValue) == fmt.Sprintf("%v", rule.Value)
	case "not_equals":
		return fmt.Sprintf("%v", fieldValue) != fmt.Sprintf("%v", rule.Value)
	case "contains":
		str := fmt.Sprintf("%v", fieldValue)
		substr := fmt.Sprintf("%v", rule.Value)
		return strings.Contains(str, substr)
	case "not_contains":
		str := fmt.Sprintf("%v", fieldValue)
		substr := fmt.Sprintf("%v", rule.Value)
		return !strings.Contains(str, substr)
	case "starts_with":
		str := fmt.Sprintf("%v", fieldValue)
		prefix := fmt.Sprintf("%v", rule.Value)
		return strings.HasPrefix(str, prefix)
	case "ends_with":
		str := fmt.Sprintf("%v", fieldValue)
		suffix := fmt.Sprintf("%v", rule.Value)
		return strings.HasSuffix(str, suffix)
	case "matches":
		// For regex matching (simplified)
		str := fmt.Sprintf("%v", fieldValue)
		pattern := fmt.Sprintf("%v", rule.Value)
		return strings.Contains(str, pattern)
	}

	return false
}

// SamplingManager manages log sampling to reduce volume
type SamplingManager struct {
	mu            sync.RWMutex
	samplingRates map[string]float64
	counters      map[string]int
}

// NewSamplingManager creates a new sampling manager
func NewSamplingManager() *SamplingManager {
	return &SamplingManager{
		samplingRates: make(map[string]float64),
		counters:      make(map[string]int),
	}
}

// SetSamplingRate sets the sampling rate for a category
func (sm *SamplingManager) SetSamplingRate(category string, rate float64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if rate < 0 {
		rate = 0
	}
	if rate > 1 {
		rate = 1
	}

	sm.samplingRates[category] = rate
}

// ShouldSample determines if a log entry should be sampled
func (sm *SamplingManager) ShouldSample(category string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	rate, ok := sm.samplingRates[category]
	if !ok {
		return true // If no rate is set, sample everything
	}

	sm.counters[category]++
	count := sm.counters[category]

	// Simple modulo-based sampling
	sampleEvery := int(1.0 / rate)
	if sampleEvery <= 0 {
		return false
	}

	return count%sampleEvery == 0
}

// GetSamplingRate gets the sampling rate for a category
func (sm *SamplingManager) GetSamplingRate(category string) float64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	rate, ok := sm.samplingRates[category]
	if !ok {
		return 1.0
	}
	return rate
}

// GetSampleCount gets the sample count for a category
func (sm *SamplingManager) GetSampleCount(category string) int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.counters[category]
}

// ResetCounters resets all sampling counters
func (sm *SamplingManager) ResetCounters() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.counters = make(map[string]int)
}

// MarshalJSON custom JSON marshaling for FilterRule
func (fr FilterRule) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"name":        fr.Name,
		"field":       fr.Field,
		"operator":    fr.Operator,
		"value":       fr.Value,
		"action":      fr.Action,
		"priority":    fr.Priority,
		"enabled":     fr.Enabled,
		"description": fr.Description,
	})
}
