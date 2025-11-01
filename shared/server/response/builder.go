package response

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	contextx "shared/server/context"
	"shared/server/headers"
)

type Builder struct {
	response  *Response
	ctx       context.Context
	request   *http.Request
	config    *Config
	debug     *DebugCollector
	startTime time.Time
}

func NewBuilder() *Builder {
	return &Builder{
		response: &Response{
			Success: true,
		},
		config:    GetGlobalConfig(),
		startTime: time.Now(),
	}
}

func Success() *Builder {
	return &Builder{
		response: &Response{
			Success: true,
		},
		config:    GetGlobalConfig(),
		startTime: time.Now(),
	}
}

func Error() *Builder {
	return &Builder{
		response: &Response{
			Success: false,
		},
		config:    GetGlobalConfig(),
		startTime: time.Now(),
	}
}

// Fluent Builder Methods

func (b *Builder) WithConfig(config *Config) *Builder {
	b.config = config
	return b
}

func (b *Builder) WithContext(ctx context.Context) *Builder {
	b.ctx = ctx
	return b
}

func (b *Builder) WithRequest(r *http.Request) *Builder {
	b.request = r
	if b.ctx == nil {
		b.ctx = r.Context()
	}
	return b
}

func (b *Builder) WithMessage(message string) *Builder {
	b.response.Message = message
	return b
}

func (b *Builder) WithData(data any) *Builder {
	b.response.Data = data
	return b
}

func (b *Builder) WithError(errorDetails *ErrorDetails) *Builder {
	b.response.Error = errorDetails
	b.response.Success = false
	return b
}

func (b *Builder) WithMetadata(metadata *Metadata) *Builder {
	b.response.Metadata = metadata
	return b
}

func (b *Builder) WithPagination(pagination *PaginationInfo) *Builder {
	b.response.Pagination = pagination
	return b
}

func (b *Builder) WithLink(link Link) *Builder {
	b.response.Links = append(b.response.Links, link)
	return b
}

func (b *Builder) WithLinks(links ...Link) *Builder {
	b.response.Links = append(b.response.Links, links...)
	return b
}

func (b *Builder) EnableDebug() *Builder {
	if b.config.ShouldIncludeDebug() {
		b.debug = NewDebugCollector(true)
	}
	return b
}

func (b *Builder) GetDebugCollector() *DebugCollector {
	if b.debug == nil && b.config.ShouldIncludeDebug() {
		b.debug = NewDebugCollector(true)
	}
	return b.debug
}

func (b *Builder) Build() *Response {
	b.finalize()
	return b.response
}

func (b *Builder) Send(w http.ResponseWriter, statusCode int) error {
	b.finalize()

	// Set metadata if not already set
	if b.response.Metadata == nil {
		b.response.Metadata = b.buildMetadata(statusCode)
	}

	// Add debug info if enabled
	if b.debug != nil && b.config.ShouldIncludeDebug() {
		b.debug.Finalize()
		b.response.Debug = b.debug.GetDebugInfo()
	}

	// Set headers
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	if b.response.Metadata != nil {
		if b.response.Metadata.RequestID != "" {
			w.Header().Set("X-Request-ID", b.response.Metadata.RequestID)
		}
		if b.response.Metadata.TraceID != "" {
			w.Header().Set("X-Trace-ID", b.response.Metadata.TraceID)
		}
	}

	if statusCode >= 400 {
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	}

	if b.response.Error != nil && b.response.Error.RetryAfter != nil {
		w.Header().Set("Retry-After", string(rune(*b.response.Error.RetryAfter)))
	}

	w.WriteHeader(statusCode)

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	if b.config.IsDevelopment() {
		encoder.SetIndent("", "  ")
	}

	return encoder.Encode(b.response)
}

// HTTP Status Code Helper Methods

func (b *Builder) OK(w http.ResponseWriter) error {
	return b.Send(w, http.StatusOK)
}

func (b *Builder) Created(w http.ResponseWriter) error {
	return b.Send(w, http.StatusCreated)
}

func (b *Builder) Accepted(w http.ResponseWriter) error {
	return b.Send(w, http.StatusAccepted)
}

func (b *Builder) NoContent(w http.ResponseWriter) error {
	return b.Send(w, http.StatusNoContent)
}

func (b *Builder) BadRequest(w http.ResponseWriter) error {
	b.response.Success = false
	return b.Send(w, http.StatusBadRequest)
}

func (b *Builder) Unauthorized(w http.ResponseWriter) error {
	b.response.Success = false
	return b.Send(w, http.StatusUnauthorized)
}

func (b *Builder) Forbidden(w http.ResponseWriter) error {
	b.response.Success = false
	return b.Send(w, http.StatusForbidden)
}

func (b *Builder) MethodNotAllowed(w http.ResponseWriter) error {
	b.response.Success = false
	return b.Send(w, http.StatusMethodNotAllowed)
}

func (b *Builder) NotFound(w http.ResponseWriter) error {
	b.response.Success = false
	return b.Send(w, http.StatusNotFound)
}

func (b *Builder) Conflict(w http.ResponseWriter) error {
	b.response.Success = false
	return b.Send(w, http.StatusConflict)
}

func (b *Builder) UnprocessableEntity(w http.ResponseWriter) error {
	b.response.Success = false
	return b.Send(w, http.StatusUnprocessableEntity)
}

func (b *Builder) TooManyRequests(w http.ResponseWriter) error {
	b.response.Success = false
	return b.Send(w, http.StatusTooManyRequests)
}

func (b *Builder) UnsupportedMediaType(w http.ResponseWriter) error {
	b.response.Success = false
	return b.Send(w, http.StatusUnsupportedMediaType)
}

func (b *Builder) InternalServerError(w http.ResponseWriter) error {
	b.response.Success = false
	return b.Send(w, http.StatusInternalServerError)
}

func (b *Builder) BadGateway(w http.ResponseWriter) error {
	b.response.Success = false
	return b.Send(w, http.StatusBadGateway)
}

func (b *Builder) ServiceUnavailable(w http.ResponseWriter) error {
	b.response.Success = false
	return b.Send(w, http.StatusServiceUnavailable)
}

func (b *Builder) GatewayTimeout(w http.ResponseWriter) error {
	b.response.Success = false
	return b.Send(w, http.StatusGatewayTimeout)
}

// Private helper methods

func (b *Builder) finalize() {
	if b.debug != nil {
		b.debug.CaptureResourceUsage()

		// Set build info
		if b.config.GitCommit != "" {
			buildTime, _ := time.Parse(time.RFC3339, b.config.BuildTime)
			b.debug.SetBuildInfo(
				b.config.Version,
				b.config.GitCommit,
				b.config.GitBranch,
				buildTime,
			)
		}
	}
}

func (b *Builder) buildMetadata(statusCode int) *Metadata {
	now := time.Now()
	duration := now.Sub(b.startTime)

	metadata := &Metadata{
		Timestamp:     now,
		TimestampUnix: now.Unix(),
		Duration:      duration.String(),
		DurationMs:    float64(duration.Microseconds()) / 1000.0,
		Service:       b.config.Service,
		Version:       b.config.Version,
		Environment:   string(b.config.Environment),
		StatusCode:    statusCode,
	}

	// Extract from context
	if b.ctx != nil {
		metadata.TraceID = contextx.GetString(b.ctx, contextx.TraceIDKey)
		metadata.SpanID = contextx.GetString(b.ctx, contextx.SpanIDKey)
		metadata.UserID = contextx.GetString(b.ctx, contextx.UserIDKey)
	}

	// Extract from request
	if b.request != nil {
		metadata.Method = b.request.Method
		metadata.Path = b.request.URL.Path
		metadata.ClientIP = getClientIP(b.request)
		metadata.UserAgent = b.request.UserAgent()
		metadata.RequestID = b.request.Header.Get(headers.XRequestID)
		metadata.CorrelationID = b.request.Header.Get(headers.XCorrelationID)

		// Add request debug info if enabled
		if b.config.EnableRequestDebug && b.debug != nil {
			b.addRequestDebugInfo(b.request)
		}
	}

	return metadata
}

func (b *Builder) addRequestDebugInfo(r *http.Request) {
	if b.debug == nil || b.debug.info == nil {
		return
	}

	reqDebug := &RequestDebugInfo{
		Headers:      make(map[string][]string),
		QueryParams:  r.URL.Query(),
		BodySize:     r.ContentLength,
		ContentType:  r.Header.Get("Content-Type"),
		Host:         r.Host,
		Protocol:     r.Proto,
		RemoteAddr:   r.RemoteAddr,
		ForwardedFor: r.Header.Get("X-Forwarded-For"),
	}

	// Copy headers (sanitize if needed)
	for key, values := range r.Header {
		if b.config.SanitizeHeaders && shouldSanitizeHeader(key) {
			reqDebug.Headers[key] = []string{"[REDACTED]"}
		} else {
			reqDebug.Headers[key] = values
		}
	}

	// TLS info
	if r.TLS != nil {
		reqDebug.TLS = &TLSInfo{
			Version:            tlsVersionString(r.TLS.Version),
			CipherSuite:        tlsCipherSuiteString(r.TLS.CipherSuite),
			ServerName:         r.TLS.ServerName,
			NegotiatedProtocol: r.TLS.NegotiatedProtocol,
		}
	}

	b.debug.info.Request = reqDebug
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return r.RemoteAddr
}

func shouldSanitizeHeader(key string) bool {
	sensitiveHeaders := []string{
		"Authorization",
		"Cookie",
		"Set-Cookie",
		"X-Api-Key",
		"X-Auth-Token",
	}

	for _, h := range sensitiveHeaders {
		if key == h {
			return true
		}
	}
	return false
}

func tlsVersionString(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return "Unknown"
	}
}

func tlsCipherSuiteString(suite uint16) string {
	suites := map[uint16]string{
		tls.TLS_AES_128_GCM_SHA256:                "TLS_AES_128_GCM_SHA256",
		tls.TLS_AES_256_GCM_SHA384:                "TLS_AES_256_GCM_SHA384",
		tls.TLS_CHACHA20_POLY1305_SHA256:          "TLS_CHACHA20_POLY1305_SHA256",
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256: "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384: "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
	}

	if name, ok := suites[suite]; ok {
		return name
	}
	return "Unknown"
}

// DebugCollector collects debug information during request processing
type DebugCollector struct {
	info      *DebugInfo
	startTime time.Time
	enabled   bool
}

// NewDebugCollector creates a new debug collector
func NewDebugCollector(enabled bool) *DebugCollector {
	if !enabled {
		return &DebugCollector{enabled: false}
	}

	return &DebugCollector{
		info: &DebugInfo{
			Performance: &PerformanceMetrics{
				Timings: TimingBreakdown{
					Custom: make(map[string]float64),
				},
			},
			Database: &DatabaseDebugInfo{
				Queries:     make([]QueryInfo, 0),
				SlowQueries: make([]QueryInfo, 0),
			},
			Cache: &CacheDebugInfo{
				Operations: make([]CacheOperation, 0),
			},
			ExternalCalls: make([]ExternalCallInfo, 0),
			Middleware:    make([]MiddlewareInfo, 0),
			FeatureFlags:  make(map[string]bool),
			Config:        make(map[string]string),
		},
		startTime: time.Now(),
		enabled:   true,
	}
}

// IsEnabled returns whether debug collection is enabled
func (dc *DebugCollector) IsEnabled() bool {
	return dc.enabled
}

// AddQuery adds a database query to debug info
func (dc *DebugCollector) AddQuery(query QueryInfo) {
	if !dc.enabled || dc.info == nil {
		return
	}
	dc.info.Database.Queries = append(dc.info.Database.Queries, query)
	dc.info.Database.TotalQueries++
	dc.info.Database.TotalTime += query.Duration
	dc.info.Performance.DBQueryCount++
	dc.info.Performance.DBQueryTime += query.Duration

	if query.Slow {
		dc.info.Database.SlowQueries = append(dc.info.Database.SlowQueries, query)
		dc.info.Performance.DBSlowQueries++
	}
}

// AddCacheOperation adds a cache operation
func (dc *DebugCollector) AddCacheOperation(op CacheOperation) {
	if !dc.enabled || dc.info == nil {
		return
	}
	dc.info.Cache.Operations = append(dc.info.Cache.Operations, op)

	if op.Hit {
		dc.info.Performance.CacheHits++
	} else {
		dc.info.Performance.CacheMisses++
	}
	dc.info.Performance.CacheTime += op.Duration
}

// AddExternalCall adds an external service call
func (dc *DebugCollector) AddExternalCall(call ExternalCallInfo) {
	if !dc.enabled || dc.info == nil {
		return
	}
	dc.info.ExternalCalls = append(dc.info.ExternalCalls, call)
	dc.info.Performance.ExternalCallCount++
	dc.info.Performance.ExternalCallTime += call.Duration

	if call.Error != "" {
		dc.info.Performance.ExternalErrors++
	}
}

// AddMiddleware adds middleware execution info
func (dc *DebugCollector) AddMiddleware(mw MiddlewareInfo) {
	if !dc.enabled || dc.info == nil {
		return
	}
	dc.info.Middleware = append(dc.info.Middleware, mw)
}

// SetTiming sets a custom timing value
func (dc *DebugCollector) SetTiming(name string, duration float64) {
	if !dc.enabled || dc.info == nil {
		return
	}
	dc.info.Performance.Timings.Custom[name] = duration
}

// SetFeatureFlag sets a feature flag value
func (dc *DebugCollector) SetFeatureFlag(name string, enabled bool) {
	if !dc.enabled || dc.info == nil {
		return
	}
	dc.info.FeatureFlags[name] = enabled
}

// SetConfig sets a configuration value
func (dc *DebugCollector) SetConfig(key, value string) {
	if !dc.enabled || dc.info == nil {
		return
	}
	dc.info.Config[key] = value
}

// SetBuildInfo sets build information
func (dc *DebugCollector) SetBuildInfo(version, gitCommit, gitBranch string, buildTime time.Time) {
	if !dc.enabled || dc.info == nil {
		return
	}

	dc.info.Build = &BuildInfo{
		Version:   version,
		GitCommit: gitCommit,
		GitBranch: gitBranch,
		BuildTime: buildTime,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		Compiler:  runtime.Compiler,
	}
}

// CaptureResourceUsage captures current resource usage
func (dc *DebugCollector) CaptureResourceUsage() {
	if !dc.enabled || dc.info == nil {
		return
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	dc.info.Resources = &ResourceUsage{
		Memory: &MemoryUsage{
			AllocMB:      float64(m.Alloc) / 1024 / 1024,
			TotalAllocMB: float64(m.TotalAlloc) / 1024 / 1024,
			SysMB:        float64(m.Sys) / 1024 / 1024,
			HeapAllocMB:  float64(m.HeapAlloc) / 1024 / 1024,
			HeapInuseMB:  float64(m.HeapInuse) / 1024 / 1024,
			StackInuseMB: float64(m.StackInuse) / 1024 / 1024,
			NumGC:        m.NumGC,
		},
		Goroutines: runtime.NumGoroutine(),
		CGoCalls:   runtime.NumCgoCall(),
		GC: &GCStats{
			NumGC:         m.NumGC,
			PauseTotalMs:  float64(m.PauseTotalNs) / 1_000_000,
			GCCPUFraction: m.GCCPUFraction,
		},
	}

	if m.NumGC > 0 {
		dc.info.Resources.GC.LastPauseMs = float64(m.PauseNs[(m.NumGC+255)%256]) / 1_000_000
		dc.info.Resources.GC.PauseAvgMs = dc.info.Resources.GC.PauseTotalMs / float64(m.NumGC)
	}

	dc.info.Performance.MemoryUsedMB = dc.info.Resources.Memory.AllocMB
	dc.info.Performance.GoroutineCount = dc.info.Resources.Goroutines
	dc.info.Performance.GCPauses = int(m.NumGC)
}

// CalculateCacheHitRatio calculates cache hit ratio
func (dc *DebugCollector) CalculateCacheHitRatio() {
	if !dc.enabled || dc.info == nil {
		return
	}

	total := dc.info.Performance.CacheHits + dc.info.Performance.CacheMisses
	if total > 0 {
		dc.info.Performance.CacheHitRatio = float64(dc.info.Performance.CacheHits) / float64(total)
	}
}

// Finalize finalizes debug info collection
func (dc *DebugCollector) Finalize() {
	if !dc.enabled || dc.info == nil {
		return
	}

	totalDuration := time.Since(dc.startTime).Milliseconds()
	dc.info.Performance.Timings.Total = float64(totalDuration)
	dc.CalculateCacheHitRatio()
	dc.CaptureResourceUsage()
}

// GetDebugInfo returns the collected debug info
func (dc *DebugCollector) GetDebugInfo() *DebugInfo {
	if !dc.enabled {
		return nil
	}
	return dc.info
}
