package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"time"

	"shared/pkg/logger"
)

// StructuredLogger provides enhanced structured logging with deep inspection
type StructuredLogger struct {
	baseLogger  *zapLogger
	formatter   *AdvancedFormatter
	colorizer   *Colorizer
	contextKeys []string
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(baseLogger *zapLogger) *StructuredLogger {
	config := DefaultFormatterConfig()
	return &StructuredLogger{
		baseLogger:  baseLogger,
		formatter:   NewAdvancedFormatter(config),
		colorizer:   NewColorizer(true),
		contextKeys: []string{"request_id", "user_id", "trace_id", "span_id", "session_id"},
	}
}

// LogStruct logs a struct with full nested structure
func (sl *StructuredLogger) LogStruct(level string, msg string, s interface{}, additionalFields ...logger.Field) {
	fields := make([]logger.Field, 0, len(additionalFields)+1)
	fields = append(fields, logger.Object("data", s))
	fields = append(fields, additionalFields...)

	switch strings.ToUpper(level) {
	case "DEBUG":
		sl.baseLogger.Debug(msg, fields...)
	case "INFO":
		sl.baseLogger.Info(msg, fields...)
	case "WARN":
		sl.baseLogger.Warn(msg, fields...)
	case "ERROR":
		sl.baseLogger.Error(msg, fields...)
	case "FATAL":
		sl.baseLogger.Fatal(msg, fields...)
	default:
		sl.baseLogger.Info(msg, fields...)
	}
}

// LogStructWithContext logs a struct with context extraction
func (sl *StructuredLogger) LogStructWithContext(ctx context.Context, level string, msg string, s interface{}, additionalFields ...logger.Field) {
	// Extract context values
	contextFields := sl.extractContextFields(ctx)
	
	allFields := make([]logger.Field, 0, len(additionalFields)+len(contextFields)+1)
	allFields = append(allFields, contextFields...)
	allFields = append(allFields, logger.Object("data", s))
	allFields = append(allFields, additionalFields...)

	sl.LogStruct(level, msg, s, allFields...)
}

// LogComparison logs a before/after comparison of structs
func (sl *StructuredLogger) LogComparison(msg string, before, after interface{}, additionalFields ...logger.Field) {
	fields := make([]logger.Field, 0, len(additionalFields)+2)
	fields = append(fields, 
		logger.Object("before", before),
		logger.Object("after", after),
	)
	fields = append(fields, additionalFields...)

	sl.baseLogger.Info(msg, fields...)
}

// LogDiff logs the differences between two structs
func (sl *StructuredLogger) LogDiff(msg string, old, new interface{}, additionalFields ...logger.Field) {
	diff := sl.computeDiff(old, new)
	
	fields := make([]logger.Field, 0, len(additionalFields)+1)
	fields = append(fields, logger.Object("diff", diff))
	fields = append(fields, additionalFields...)

	sl.baseLogger.Info(msg, fields...)
}

// LogCollection logs a collection of items with metadata
func (sl *StructuredLogger) LogCollection(msg string, items interface{}, additionalFields ...logger.Field) {
	val := reflect.ValueOf(items)
	
	metadata := map[string]interface{}{
		"type":  val.Type().String(),
		"count": val.Len(),
	}
	
	if val.Len() > 0 {
		metadata["first_item"] = val.Index(0).Interface()
		if val.Len() > 1 {
			metadata["last_item"] = val.Index(val.Len() - 1).Interface()
		}
	}

	fields := make([]logger.Field, 0, len(additionalFields)+2)
	fields = append(fields,
		logger.Object("metadata", metadata),
		logger.Object("items", items),
	)
	fields = append(fields, additionalFields...)

	sl.baseLogger.Info(msg, fields...)
}

// LogMetrics logs performance metrics for an operation
func (sl *StructuredLogger) LogMetrics(operation string, duration time.Duration, metrics map[string]interface{}, additionalFields ...logger.Field) {
	metricsData := map[string]interface{}{
		"operation":    operation,
		"duration_ms":  duration.Milliseconds(),
		"duration_str": duration.String(),
	}
	
	for k, v := range metrics {
		metricsData[k] = v
	}

	fields := make([]logger.Field, 0, len(additionalFields)+1)
	fields = append(fields, logger.Object("metrics", metricsData))
	fields = append(fields, additionalFields...)

	sl.baseLogger.Info(fmt.Sprintf("Metrics: %s", operation), fields...)
}

// LogAudit logs an audit event with detailed tracking
func (sl *StructuredLogger) LogAudit(ctx context.Context, action string, resource string, details interface{}, additionalFields ...logger.Field) {
	caller := sl.getCaller(2)
	
	auditData := map[string]interface{}{
		"action":     action,
		"resource":   resource,
		"timestamp":  time.Now().UTC(),
		"caller":     caller,
		"details":    details,
	}

	contextFields := sl.extractContextFields(ctx)
	
	fields := make([]logger.Field, 0, len(additionalFields)+len(contextFields)+1)
	fields = append(fields, contextFields...)
	fields = append(fields, logger.Object("audit", auditData))
	fields = append(fields, additionalFields...)

	sl.baseLogger.Info(fmt.Sprintf("AUDIT: %s on %s", action, resource), fields...)
}

// LogTransaction logs a transaction with its phases
func (sl *StructuredLogger) LogTransaction(transactionID string, phase string, data interface{}, additionalFields ...logger.Field) {
	txData := map[string]interface{}{
		"transaction_id": transactionID,
		"phase":          phase,
		"timestamp":      time.Now().UTC(),
		"data":           data,
	}

	fields := make([]logger.Field, 0, len(additionalFields)+1)
	fields = append(fields, logger.Object("transaction", txData))
	fields = append(fields, additionalFields...)

	sl.baseLogger.Info(fmt.Sprintf("Transaction [%s]: %s", transactionID, phase), fields...)
}

// LogStateChange logs a state transition
func (sl *StructuredLogger) LogStateChange(entity string, entityID string, fromState, toState string, data interface{}, additionalFields ...logger.Field) {
	stateData := map[string]interface{}{
		"entity":      entity,
		"entity_id":   entityID,
		"from_state":  fromState,
		"to_state":    toState,
		"timestamp":   time.Now().UTC(),
		"data":        data,
	}

	fields := make([]logger.Field, 0, len(additionalFields)+1)
	fields = append(fields, logger.Object("state_change", stateData))
	fields = append(fields, additionalFields...)

	sl.baseLogger.Info(fmt.Sprintf("State Change: %s [%s] %s -> %s", entity, entityID, fromState, toState), fields...)
}

// LogError logs an enhanced error with stack trace and context
func (sl *StructuredLogger) LogError(ctx context.Context, err error, msg string, data interface{}, additionalFields ...logger.Field) {
	errorData := map[string]interface{}{
		"error":     err.Error(),
		"timestamp": time.Now().UTC(),
	}
	
	if data != nil {
		errorData["context_data"] = data
	}

	// Extract stack trace
	pc := make([]uintptr, 15)
	n := runtime.Callers(2, pc)
	frames := runtime.CallersFrames(pc[:n])
	
	var stackTrace []string
	for {
		frame, more := frames.Next()
		stackTrace = append(stackTrace, fmt.Sprintf("%s:%d %s", frame.File, frame.Line, frame.Function))
		if !more {
			break
		}
	}
	errorData["stack_trace"] = stackTrace

	contextFields := sl.extractContextFields(ctx)
	
	fields := make([]logger.Field, 0, len(additionalFields)+len(contextFields)+1)
	fields = append(fields, contextFields...)
	fields = append(fields, logger.Object("error_details", errorData))
	fields = append(fields, additionalFields...)

	sl.baseLogger.Error(msg, fields...)
}

// LogPerformanceTrace logs detailed performance tracing
func (sl *StructuredLogger) LogPerformanceTrace(operation string, phases map[string]time.Duration, additionalFields ...logger.Field) {
	var totalDuration time.Duration
	phaseData := make(map[string]interface{})
	
	for phase, duration := range phases {
		totalDuration += duration
		phaseData[phase] = map[string]interface{}{
			"duration_ms": duration.Milliseconds(),
			"duration":    duration.String(),
			"percentage":  0.0,
		}
	}
	
	// Calculate percentages
	if totalDuration > 0 {
		for phase, duration := range phases {
			if data, ok := phaseData[phase].(map[string]interface{}); ok {
				data["percentage"] = float64(duration.Milliseconds()) / float64(totalDuration.Milliseconds()) * 100
			}
		}
	}

	traceData := map[string]interface{}{
		"operation":    operation,
		"total_ms":     totalDuration.Milliseconds(),
		"total":        totalDuration.String(),
		"phases":       phaseData,
		"phase_count":  len(phases),
	}

	fields := make([]logger.Field, 0, len(additionalFields)+1)
	fields = append(fields, logger.Object("performance_trace", traceData))
	fields = append(fields, additionalFields...)

	sl.baseLogger.Info(fmt.Sprintf("Performance Trace: %s", operation), fields...)
}

// LogDataFlow logs data flow through a pipeline
func (sl *StructuredLogger) LogDataFlow(pipeline string, stage string, input, output interface{}, metrics map[string]interface{}, additionalFields ...logger.Field) {
	flowData := map[string]interface{}{
		"pipeline":  pipeline,
		"stage":     stage,
		"timestamp": time.Now().UTC(),
	}
	
	if input != nil {
		flowData["input"] = sl.summarizeData(input)
	}
	if output != nil {
		flowData["output"] = sl.summarizeData(output)
	}
	if metrics != nil {
		flowData["metrics"] = metrics
	}

	fields := make([]logger.Field, 0, len(additionalFields)+1)
	fields = append(fields, logger.Object("data_flow", flowData))
	fields = append(fields, additionalFields...)

	sl.baseLogger.Info(fmt.Sprintf("Data Flow [%s]: %s", pipeline, stage), fields...)
}

// LogValidation logs validation results
func (sl *StructuredLogger) LogValidation(entity string, validationResults map[string][]string, data interface{}, additionalFields ...logger.Field) {
	errorCount := 0
	for _, errors := range validationResults {
		errorCount += len(errors)
	}

	validationData := map[string]interface{}{
		"entity":        entity,
		"is_valid":      errorCount == 0,
		"error_count":   errorCount,
		"errors":        validationResults,
		"validated_at":  time.Now().UTC(),
	}
	
	if data != nil {
		validationData["data"] = data
	}

	fields := make([]logger.Field, 0, len(additionalFields)+1)
	fields = append(fields, logger.Object("validation", validationData))
	fields = append(fields, additionalFields...)

	level := "INFO"
	if errorCount > 0 {
		level = "WARN"
	}

	sl.LogStruct(level, fmt.Sprintf("Validation: %s (%d errors)", entity, errorCount), validationData, fields...)
}

// extractContextFields extracts logging-relevant fields from context
func (sl *StructuredLogger) extractContextFields(ctx context.Context) []logger.Field {
	if ctx == nil {
		return nil
	}

	var fields []logger.Field
	for _, key := range sl.contextKeys {
		if val := ctx.Value(key); val != nil {
			fields = append(fields, logger.Any(key, val))
		}
	}

	return fields
}

// getCaller returns information about the caller
func (sl *StructuredLogger) getCaller(skip int) string {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown"
	}

	fn := runtime.FuncForPC(pc)
	fnName := "unknown"
	if fn != nil {
		fnName = fn.Name()
	}

	parts := strings.Split(file, "/")
	shortFile := file
	if len(parts) > 0 {
		shortFile = parts[len(parts)-1]
	}

	return fmt.Sprintf("%s:%d (%s)", shortFile, line, fnName)
}

// computeDiff computes differences between two values
func (sl *StructuredLogger) computeDiff(old, new interface{}) map[string]interface{} {
	oldVal := reflect.ValueOf(old)
	newVal := reflect.ValueOf(new)

	if oldVal.Kind() != newVal.Kind() {
		return map[string]interface{}{
			"type_change": true,
			"old_type":    oldVal.Type().String(),
			"new_type":    newVal.Type().String(),
		}
	}

	diff := make(map[string]interface{})

	if oldVal.Kind() == reflect.Struct {
		oldType := oldVal.Type()
		for i := 0; i < oldVal.NumField(); i++ {
			field := oldType.Field(i)
			if !field.IsExported() {
				continue
			}

			oldFieldVal := oldVal.Field(i)
			newFieldVal := newVal.Field(i)

			if !reflect.DeepEqual(oldFieldVal.Interface(), newFieldVal.Interface()) {
				diff[field.Name] = map[string]interface{}{
					"old": oldFieldVal.Interface(),
					"new": newFieldVal.Interface(),
				}
			}
		}
	} else if oldVal.Kind() == reflect.Map {
		oldKeys := oldVal.MapKeys()
		newKeys := newVal.MapKeys()

		// Find changed and removed keys
		for _, key := range oldKeys {
			oldMapVal := oldVal.MapIndex(key)
			newMapVal := newVal.MapIndex(key)

			keyStr := fmt.Sprintf("%v", key.Interface())
			if !newMapVal.IsValid() {
				diff[keyStr] = map[string]interface{}{
					"status": "removed",
					"old":    oldMapVal.Interface(),
				}
			} else if !reflect.DeepEqual(oldMapVal.Interface(), newMapVal.Interface()) {
				diff[keyStr] = map[string]interface{}{
					"old": oldMapVal.Interface(),
					"new": newMapVal.Interface(),
				}
			}
		}

		// Find added keys
		for _, key := range newKeys {
			oldMapVal := oldVal.MapIndex(key)
			if !oldMapVal.IsValid() {
				keyStr := fmt.Sprintf("%v", key.Interface())
				newMapVal := newVal.MapIndex(key)
				diff[keyStr] = map[string]interface{}{
					"status": "added",
					"new":    newMapVal.Interface(),
				}
			}
		}
	}

	return diff
}

// summarizeData creates a summary of data for logging
func (sl *StructuredLogger) summarizeData(data interface{}) map[string]interface{} {
	summary := make(map[string]interface{})
	
	val := reflect.ValueOf(data)
	summary["type"] = val.Type().String()

	switch val.Kind() {
	case reflect.Slice, reflect.Array:
		summary["length"] = val.Len()
		if val.Len() > 0 {
			summary["first_element_type"] = val.Index(0).Type().String()
		}
	case reflect.Map:
		summary["key_count"] = len(val.MapKeys())
	case reflect.Struct:
		summary["field_count"] = val.NumField()
	case reflect.String:
		length := val.Len()
		summary["length"] = length
		if length > 100 {
			summary["preview"] = val.String()[:100] + "..."
		}
	case reflect.Ptr:
		if !val.IsNil() {
			return sl.summarizeData(val.Elem().Interface())
		}
	}

	return summary
}

// BatchLogger provides batched logging for high-volume scenarios
type BatchLogger struct {
	structuredLogger *StructuredLogger
	batchSize        int
	flushInterval    time.Duration
	buffer           []batchEntry
	ticker           *time.Ticker
	done             chan bool
}

type batchEntry struct {
	level   string
	msg     string
	fields  []logger.Field
	logTime time.Time
}

// NewBatchLogger creates a new batch logger
func NewBatchLogger(structuredLogger *StructuredLogger, batchSize int, flushInterval time.Duration) *BatchLogger {
	bl := &BatchLogger{
		structuredLogger: structuredLogger,
		batchSize:        batchSize,
		flushInterval:    flushInterval,
		buffer:           make([]batchEntry, 0, batchSize),
		ticker:           time.NewTicker(flushInterval),
		done:             make(chan bool),
	}

	go bl.autoFlush()
	return bl
}

// Log adds an entry to the batch
func (bl *BatchLogger) Log(level string, msg string, fields ...logger.Field) {
	entry := batchEntry{
		level:   level,
		msg:     msg,
		fields:  fields,
		logTime: time.Now(),
	}

	bl.buffer = append(bl.buffer, entry)

	if len(bl.buffer) >= bl.batchSize {
		bl.Flush()
	}
}

// Flush flushes the current batch
func (bl *BatchLogger) Flush() {
	if len(bl.buffer) == 0 {
		return
	}

	batchData := map[string]interface{}{
		"entry_count": len(bl.buffer),
		"entries":     bl.buffer,
	}

	bl.structuredLogger.LogStruct("INFO", "Batch log flush", batchData)
	bl.buffer = bl.buffer[:0]
}

// autoFlush periodically flushes the batch
func (bl *BatchLogger) autoFlush() {
	for {
		select {
		case <-bl.ticker.C:
			bl.Flush()
		case <-bl.done:
			bl.ticker.Stop()
			bl.Flush()
			return
		}
	}
}

// Close closes the batch logger
func (bl *BatchLogger) Close() {
	bl.done <- true
}

// TraceLogger provides distributed tracing support
type TraceLogger struct {
	structuredLogger *StructuredLogger
	traceID          string
	spanID           string
	parentSpanID     string
	serviceName      string
}

// NewTraceLogger creates a new trace logger
func NewTraceLogger(structuredLogger *StructuredLogger, traceID, spanID, parentSpanID, serviceName string) *TraceLogger {
	return &TraceLogger{
		structuredLogger: structuredLogger,
		traceID:          traceID,
		spanID:           spanID,
		parentSpanID:     parentSpanID,
		serviceName:      serviceName,
	}
}

// LogSpan logs a span event
func (tl *TraceLogger) LogSpan(operation string, startTime, endTime time.Time, tags map[string]interface{}, additionalFields ...logger.Field) {
	duration := endTime.Sub(startTime)
	
	spanData := map[string]interface{}{
		"trace_id":       tl.traceID,
		"span_id":        tl.spanID,
		"parent_span_id": tl.parentSpanID,
		"service_name":   tl.serviceName,
		"operation":      operation,
		"start_time":     startTime,
		"end_time":       endTime,
		"duration_ms":    duration.Milliseconds(),
		"tags":           tags,
	}

	fields := make([]logger.Field, 0, len(additionalFields)+1)
	fields = append(fields, logger.Object("span", spanData))
	fields = append(fields, additionalFields...)

	tl.structuredLogger.baseLogger.Info(fmt.Sprintf("Span: %s", operation), fields...)
}

// MarshalJSON custom JSON marshaling for batchEntry
func (be batchEntry) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"level":    be.level,
		"msg":      be.msg,
		"log_time": be.logTime,
	})
}
