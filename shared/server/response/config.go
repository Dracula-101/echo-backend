package response

import "time"

// Response represents the standard API response envelope
// This structure is returned for all API responses
type Response struct {
	Success bool          `json:"success"`
	Message string        `json:"message,omitempty"`
	Data    any           `json:"data,omitempty"`
	Error   *ErrorDetails `json:"error,omitempty"`

	Metadata   *Metadata       `json:"metadata"`
	Pagination *PaginationInfo `json:"pagination,omitempty"`
	Links      []Link          `json:"links,omitempty"`
	Debug      *DebugInfo      `json:"debug,omitempty"`
}

// Metadata provides comprehensive request tracking information
type Metadata struct {
	RequestID     string            `json:"request_id"`
	CorrelationID string            `json:"correlation_id,omitempty"`
	TraceID       string            `json:"trace_id,omitempty"`
	SpanID        string            `json:"span_id,omitempty"`
	Timestamp     time.Time         `json:"timestamp"`
	TimestampUnix int64             `json:"timestamp_unix"`
	Duration      string            `json:"duration"`
	DurationMs    float64           `json:"duration_ms"`
	Method        string            `json:"method"`
	Path          string            `json:"path"`
	StatusCode    int               `json:"status_code"`
	ClientIP      string            `json:"client_ip,omitempty"`
	UserAgent     string            `json:"user_agent,omitempty"`
	Service       string            `json:"service"`
	Version       string            `json:"version"`
	Environment   string            `json:"environment"`
	UserID        string            `json:"user_id,omitempty"`
	TenantID      string            `json:"tenant_id,omitempty"`
	Extra         map[string]string `json:"extra,omitempty"`
}

// ErrorDetails provides comprehensive error information
type ErrorDetails struct {
	Code        string                 `json:"code"`
	Type        ErrorType              `json:"type"`
	Message     string                 `json:"message"`
	Description string                 `json:"description,omitempty"`
	Severity    ErrorSeverity          `json:"severity"`
	Retryable   bool                   `json:"retryable"`
	RetryAfter  *int                   `json:"retry_after,omitempty"` // seconds
	Fields      []FieldError           `json:"fields,omitempty"`
	StackTrace  []string               `json:"stack_trace,omitempty"`
	InnerError  string                 `json:"inner_error,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

// ErrorType categorizes errors for client handling
type ErrorType string

const (
	ErrorTypeValidation     ErrorType = "validation_error"
	ErrorTypeAuthentication ErrorType = "authentication_error"
	ErrorTypeAuthorization  ErrorType = "authorization_error"
	ErrorTypeNotFound       ErrorType = "not_found_error"
	ErrorTypeConflict       ErrorType = "conflict_error"
	ErrorTypeRateLimit      ErrorType = "rate_limit_error"
	ErrorTypeInternal       ErrorType = "internal_error"
	ErrorTypeService        ErrorType = "service_error"
	ErrorTypeNetwork        ErrorType = "network_error"
	ErrorTypeTimeout        ErrorType = "timeout_error"
	ErrorTypeUnavailable    ErrorType = "unavailable_error"
	ErrorTypeBadRequest     ErrorType = "bad_request_error"
	ErrorTypeCircuitOpen    ErrorType = "circuit_open_error"
	ErrorTypeDependency     ErrorType = "dependency_error"
)

// ErrorSeverity indicates error severity for monitoring
type ErrorSeverity string

const (
	SeverityCritical ErrorSeverity = "critical"
	SeverityHigh     ErrorSeverity = "high"
	SeverityMedium   ErrorSeverity = "medium"
	SeverityLow      ErrorSeverity = "low"
	SeverityInfo     ErrorSeverity = "info"
)

// FieldError represents a validation error for a specific field
type FieldError struct {
	Field       string `json:"field"`
	Message     string `json:"message"`
	Code        string `json:"code"`
	Value       any    `json:"value,omitempty"`
	Constraints string `json:"constraints,omitempty"`
}

// ServiceError tracks errors across microservices for distributed tracing
type ServiceError struct {
	Service   string    `json:"service"`
	Timestamp time.Time `json:"timestamp"`
	Error     string    `json:"error"`
	Code      string    `json:"code,omitempty"`
	TraceID   string    `json:"trace_id,omitempty"`
	Duration  float64   `json:"duration_ms,omitempty"`
}

// PaginationInfo supports both offset and cursor-based pagination
type PaginationInfo struct {
	Type PaginationType `json:"type"`

	// Offset-based pagination
	TotalItems  *int64 `json:"total_items,omitempty"`
	TotalPages  *int   `json:"total_pages,omitempty"`
	CurrentPage *int   `json:"current_page,omitempty"`
	PageSize    int    `json:"page_size"`

	// Navigation
	HasNext     bool `json:"has_next"`
	HasPrevious bool `json:"has_previous"`

	// Cursor-based pagination
	NextCursor *string `json:"next_cursor,omitempty"`
	PrevCursor *string `json:"prev_cursor,omitempty"`

	// Current page info
	ItemsInPage int `json:"items_in_page"`
}

// PaginationType indicates the pagination strategy
type PaginationType string

const (
	PaginationOffset PaginationType = "offset"
	PaginationCursor PaginationType = "cursor"
	PaginationKeyset PaginationType = "keyset"
)

// Link provides HATEOAS links for resource discovery
type Link struct {
	Rel         string `json:"rel"`
	Href        string `json:"href"`
	Method      string `json:"method,omitempty"`
	Type        string `json:"type,omitempty"`
	Templated   bool   `json:"templated,omitempty"`
	Description string `json:"description,omitempty"`
}

// DebugInfo contains comprehensive debugging information (development only)
type DebugInfo struct {
	Request       *RequestDebugInfo   `json:"request,omitempty"`
	Performance   *PerformanceMetrics `json:"performance,omitempty"`
	Database      *DatabaseDebugInfo  `json:"database,omitempty"`
	Cache         *CacheDebugInfo     `json:"cache,omitempty"`
	ExternalCalls []ExternalCallInfo  `json:"external_calls,omitempty"`
	Resources     *ResourceUsage      `json:"resources,omitempty"`
	Middleware    []MiddlewareInfo    `json:"middleware,omitempty"`
	FeatureFlags  map[string]bool     `json:"feature_flags,omitempty"`
	Config        map[string]string   `json:"config,omitempty"`
	Build         *BuildInfo          `json:"build,omitempty"`
	Security      *SecurityDebugInfo  `json:"security,omitempty"`
}

// RequestDebugInfo provides detailed request information
type RequestDebugInfo struct {
	Headers      map[string][]string `json:"headers,omitempty"`
	QueryParams  map[string][]string `json:"query_params,omitempty"`
	PathParams   map[string]string   `json:"path_params,omitempty"`
	BodySize     int64               `json:"body_size_bytes"`
	ContentType  string              `json:"content_type"`
	Host         string              `json:"host"`
	Protocol     string              `json:"protocol"`
	TLS          *TLSInfo            `json:"tls,omitempty"`
	RemoteAddr   string              `json:"remote_addr"`
	ForwardedFor string              `json:"forwarded_for,omitempty"`
}

// TLSInfo contains TLS/SSL information
type TLSInfo struct {
	Version            string `json:"version"`
	CipherSuite        string `json:"cipher_suite"`
	ServerName         string `json:"server_name"`
	NegotiatedProtocol string `json:"negotiated_protocol"`
}

// PerformanceMetrics tracks detailed performance metrics
type PerformanceMetrics struct {
	Timings           TimingBreakdown `json:"timings"`
	DBQueryTime       float64         `json:"db_query_time_ms"`
	DBQueryCount      int             `json:"db_query_count"`
	DBSlowQueries     int             `json:"db_slow_queries"`
	CacheHits         int             `json:"cache_hits"`
	CacheMisses       int             `json:"cache_misses"`
	CacheHitRatio     float64         `json:"cache_hit_ratio"`
	CacheTime         float64         `json:"cache_time_ms"`
	ExternalCallCount int             `json:"external_call_count"`
	ExternalCallTime  float64         `json:"external_call_time_ms"`
	ExternalErrors    int             `json:"external_errors"`
	MemoryUsedMB      float64         `json:"memory_used_mb"`
	GoroutineCount    int             `json:"goroutine_count"`
	GCPauses          int             `json:"gc_pauses"`
}

// TimingBreakdown provides detailed timing information
type TimingBreakdown struct {
	Total         float64            `json:"total_ms"`
	Middleware    float64            `json:"middleware_ms"`
	Auth          float64            `json:"auth_ms"`
	Validation    float64            `json:"validation_ms"`
	BusinessLogic float64            `json:"business_logic_ms"`
	Database      float64            `json:"database_ms"`
	Cache         float64            `json:"cache_ms"`
	External      float64            `json:"external_ms"`
	Serialization float64            `json:"serialization_ms"`
	Custom        map[string]float64 `json:"custom,omitempty"`
}

// DatabaseDebugInfo contains database operation details
type DatabaseDebugInfo struct {
	Queries        []QueryInfo         `json:"queries,omitempty"`
	SlowQueries    []QueryInfo         `json:"slow_queries,omitempty"`
	ConnectionPool *ConnectionPoolInfo `json:"connection_pool,omitempty"`
	TotalQueries   int                 `json:"total_queries"`
	TotalTime      float64             `json:"total_time_ms"`
}

// QueryInfo contains information about a database query
type QueryInfo struct {
	SQL          string    `json:"sql"`
	Duration     float64   `json:"duration_ms"`
	RowsAffected int64     `json:"rows_affected,omitempty"`
	Error        string    `json:"error,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
	Slow         bool      `json:"slow,omitempty"`
}

// ConnectionPoolInfo provides connection pool statistics
type ConnectionPoolInfo struct {
	MaxOpen      int     `json:"max_open"`
	Open         int     `json:"open"`
	InUse        int     `json:"in_use"`
	Idle         int     `json:"idle"`
	WaitCount    int64   `json:"wait_count"`
	WaitDuration float64 `json:"wait_duration_ms"`
}

// CacheDebugInfo contains cache operation details
type CacheDebugInfo struct {
	Operations []CacheOperation `json:"operations,omitempty"`
	Stats      *CacheStats      `json:"stats,omitempty"`
}

// CacheOperation represents a cache operation
type CacheOperation struct {
	Operation string    `json:"operation"`
	Key       string    `json:"key"`
	Hit       bool      `json:"hit"`
	Duration  float64   `json:"duration_ms"`
	Size      int64     `json:"size_bytes,omitempty"`
	TTL       int       `json:"ttl_seconds,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// CacheStats provides cache statistics
type CacheStats struct {
	TotalOps    int     `json:"total_ops"`
	Hits        int     `json:"hits"`
	Misses      int     `json:"misses"`
	HitRatio    float64 `json:"hit_ratio"`
	AvgDuration float64 `json:"avg_duration_ms"`
	TotalSize   int64   `json:"total_size_bytes"`
}

// ExternalCallInfo tracks calls to external services
type ExternalCallInfo struct {
	Service      string                 `json:"service"`
	Method       string                 `json:"method"`
	URL          string                 `json:"url"`
	StatusCode   int                    `json:"status_code,omitempty"`
	Duration     float64                `json:"duration_ms"`
	Retries      int                    `json:"retries"`
	CircuitState string                 `json:"circuit_state,omitempty"`
	Error        string                 `json:"error,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
	Context      map[string]interface{} `json:"context,omitempty"`
}

// ResourceUsage tracks system resource usage
type ResourceUsage struct {
	Memory     *MemoryUsage `json:"memory"`
	Goroutines int          `json:"goroutines"`
	CGoCalls   int64        `json:"cgo_calls"`
	GC         *GCStats     `json:"gc,omitempty"`
}

// MemoryUsage tracks memory usage
type MemoryUsage struct {
	AllocMB      float64 `json:"alloc_mb"`
	TotalAllocMB float64 `json:"total_alloc_mb"`
	SysMB        float64 `json:"sys_mb"`
	HeapAllocMB  float64 `json:"heap_alloc_mb"`
	HeapInuseMB  float64 `json:"heap_inuse_mb"`
	StackInuseMB float64 `json:"stack_inuse_mb"`
	NumGC        uint32  `json:"num_gc"`
}

// GCStats tracks garbage collection statistics
type GCStats struct {
	NumGC         uint32  `json:"num_gc"`
	PauseTotalMs  float64 `json:"pause_total_ms"`
	LastPauseMs   float64 `json:"last_pause_ms"`
	PauseAvgMs    float64 `json:"pause_avg_ms"`
	GCCPUFraction float64 `json:"gc_cpu_fraction"`
}

// MiddlewareInfo tracks middleware execution
type MiddlewareInfo struct {
	Name      string    `json:"name"`
	Duration  float64   `json:"duration_ms"`
	Order     int       `json:"order"`
	Status    string    `json:"status"`
	Error     string    `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// BuildInfo provides build and version information
type BuildInfo struct {
	Version   string    `json:"version"`
	GitCommit string    `json:"git_commit"`
	GitBranch string    `json:"git_branch"`
	BuildTime time.Time `json:"build_time"`
	GoVersion string    `json:"go_version"`
	OS        string    `json:"os"`
	Arch      string    `json:"arch"`
	Compiler  string    `json:"compiler"`
}

// SecurityDebugInfo provides security context (sanitized)
type SecurityDebugInfo struct {
	Authenticated bool           `json:"authenticated"`
	UserID        string         `json:"user_id,omitempty"`
	Roles         []string       `json:"roles,omitempty"`
	Permissions   []string       `json:"permissions,omitempty"`
	TokenType     string         `json:"token_type,omitempty"`
	TokenExpiry   *time.Time     `json:"token_expiry,omitempty"`
	IPAllowed     bool           `json:"ip_allowed"`
	RateLimitInfo *RateLimitInfo `json:"rate_limit,omitempty"`
}

// RateLimitInfo provides rate limiting information
type RateLimitInfo struct {
	Limit     int   `json:"limit"`
	Remaining int   `json:"remaining"`
	Reset     int64 `json:"reset"`
}
