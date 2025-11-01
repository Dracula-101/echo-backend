package tracing

import (
	"context"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// Config holds tracing middleware configuration
type Config struct {
	// ServiceName is the name of the service
	ServiceName string

	// TracerProvider is the OpenTelemetry tracer provider
	TracerProvider trace.TracerProvider

	// Propagator is the context propagator
	Propagator propagation.TextMapPropagator

	// SpanNameFormatter formats the span name from the request
	SpanNameFormatter func(*http.Request) string

	// ExcludePaths are paths to exclude from tracing
	ExcludePaths map[string]bool
}

// Middleware creates a tracing middleware
func Middleware(config Config) func(http.Handler) http.Handler {
	if config.TracerProvider == nil {
		config.TracerProvider = otel.GetTracerProvider()
	}

	if config.Propagator == nil {
		config.Propagator = otel.GetTextMapPropagator()
	}

	if config.SpanNameFormatter == nil {
		config.SpanNameFormatter = func(r *http.Request) string {
			return r.Method + " " + r.URL.Path
		}
	}

	tracer := config.TracerProvider.Tracer(config.ServiceName)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip excluded paths
			if config.ExcludePaths != nil && config.ExcludePaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			// Extract trace context from incoming request
			ctx := config.Propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

			// Start span
			spanName := config.SpanNameFormatter(r)
			ctx, span := tracer.Start(ctx, spanName,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					attribute.String("http.method", r.Method),
					attribute.String("http.url", r.URL.String()),
					attribute.String("http.target", r.URL.Path),
					attribute.String("http.scheme", r.URL.Scheme),
					attribute.String("http.host", r.Host),
					attribute.String("http.user_agent", r.UserAgent()),
					attribute.String("http.remote_addr", r.RemoteAddr),
				),
			)
			defer span.End()

			// Add trace context to outgoing requests
			config.Propagator.Inject(ctx, propagation.HeaderCarrier(w.Header()))

			// Wrap response writer to capture status code
			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Track request start time
			start := time.Now()

			// Process request
			next.ServeHTTP(wrapped, r.WithContext(ctx))

			// Record span attributes after request completion
			duration := time.Since(start)

			span.SetAttributes(
				attribute.Int("http.status_code", wrapped.statusCode),
				attribute.Int64("http.response_size", wrapped.written),
				attribute.Int64("http.duration_ms", duration.Milliseconds()),
			)

			// Set span status based on HTTP status code
			if wrapped.statusCode >= 400 {
				if wrapped.statusCode >= 500 {
					span.SetStatus(codes.Error, http.StatusText(wrapped.statusCode))
				} else {
					span.SetStatus(codes.Error, http.StatusText(wrapped.statusCode))
				}
			} else {
				span.SetStatus(codes.Ok, "")
			}

			// Add request ID if available
			if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
				span.SetAttributes(attribute.String("http.request_id", requestID))
			}

			// Add user ID if available in context
			if userID, ok := ctx.Value("user_id").(string); ok {
				span.SetAttributes(attribute.String("user.id", userID))
			}
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture response details
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    int64
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.written += int64(n)
	return n, err
}

// ProxyTracing adds tracing context to proxied requests
func ProxyTracing(r *http.Request, targetService string) *http.Request {
	ctx := r.Context()
	span := trace.SpanFromContext(ctx)

	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("proxy.target_service", targetService),
			attribute.String("proxy.target_url", r.URL.String()),
		)
	}

	// Inject trace context into proxied request
	propagator := otel.GetTextMapPropagator()
	propagator.Inject(ctx, propagation.HeaderCarrier(r.Header))

	return r
}

// AddSpanEvent adds an event to the current span
func AddSpanEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.AddEvent(name, trace.WithAttributes(attrs...))
	}
}

// SetSpanError records an error on the current span
func SetSpanError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() && err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// SetSpanAttributes sets attributes on the current span
func SetSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(attrs...)
	}
}

// GetTraceID extracts the trace ID from context
func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// GetSpanID extracts the span ID from context
func GetSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}

// StartChildSpan starts a new child span
func StartChildSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	tracer := otel.Tracer("api-gateway")
	return tracer.Start(ctx, name, opts...)
}

// HTTPClientTracing adds tracing to HTTP client requests
type HTTPClientTracing struct {
	client     *http.Client
	tracer     trace.Tracer
	propagator propagation.TextMapPropagator
}

// NewHTTPClientTracing creates a new tracing HTTP client
func NewHTTPClientTracing(client *http.Client, serviceName string) *HTTPClientTracing {
	return &HTTPClientTracing{
		client:     client,
		tracer:     otel.Tracer(serviceName),
		propagator: otel.GetTextMapPropagator(),
	}
}

// Do executes an HTTP request with tracing
func (h *HTTPClientTracing) Do(r *http.Request) (*http.Response, error) {
	ctx := r.Context()

	// Start client span
	ctx, span := h.tracer.Start(ctx, r.Method+" "+r.URL.Path,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.url", r.URL.String()),
			attribute.String("http.target", r.URL.Path),
			attribute.String("http.host", r.URL.Host),
		),
	)
	defer span.End()

	// Inject trace context
	h.propagator.Inject(ctx, propagation.HeaderCarrier(r.Header))

	// Execute request
	start := time.Now()
	resp, err := h.client.Do(r.WithContext(ctx))
	duration := time.Since(start)

	// Record results
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(
		attribute.Int("http.status_code", resp.StatusCode),
		attribute.Int64("http.duration_ms", duration.Milliseconds()),
	)

	if resp.StatusCode >= 400 {
		span.SetStatus(codes.Error, http.StatusText(resp.StatusCode))
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return resp, nil
}

// Common span attribute helpers

// WithServiceAttributes returns attributes for service identification
func WithServiceAttributes(name, version, environment string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("service.name", name),
		attribute.String("service.version", version),
		attribute.String("deployment.environment", environment),
	}
}

// WithDatabaseAttributes returns attributes for database operations
func WithDatabaseAttributes(system, name, operation, statement string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("db.system", system),
		attribute.String("db.name", name),
		attribute.String("db.operation", operation),
		attribute.String("db.statement", statement),
	}
}

// WithCacheAttributes returns attributes for cache operations
func WithCacheAttributes(operation, key string, hit bool) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("cache.operation", operation),
		attribute.String("cache.key", key),
		attribute.Bool("cache.hit", hit),
	}
}

// WithMessageAttributes returns attributes for messaging operations
func WithMessageAttributes(system, destination, operation string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("messaging.system", system),
		attribute.String("messaging.destination", destination),
		attribute.String("messaging.operation", operation),
	}
}

// WithErrorAttributes returns attributes for errors
func WithErrorAttributes(err error, errorType string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("error.type", errorType),
		attribute.String("error.message", err.Error()),
		attribute.Bool("error", true),
	}
}
