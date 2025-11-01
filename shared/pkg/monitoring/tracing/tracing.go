package tracing

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// Config holds tracing configuration
type Config struct {
	// ServiceName is the name of the service
	ServiceName string

	// ServiceVersion is the version of the service
	ServiceVersion string

	// Environment is the deployment environment (dev, staging, prod)
	Environment string

	// JaegerEndpoint is the Jaeger collector endpoint
	JaegerEndpoint string

	// SamplingRate is the trace sampling rate (0.0 to 1.0)
	SamplingRate float64

	// Enabled determines if tracing is enabled
	Enabled bool
}

// Tracer wraps the OpenTelemetry tracer
type Tracer struct {
	tracer   trace.Tracer
	provider *sdktrace.TracerProvider
	config   Config
}

// New creates a new tracer with the given configuration
func New(config Config) (*Tracer, error) {
	if !config.Enabled {
		// Return a no-op tracer
		return &Tracer{
			tracer: otel.Tracer(config.ServiceName),
			config: config,
		}, nil
	}

	// Create Jaeger exporter
	exporter, err := jaeger.New(
		jaeger.WithCollectorEndpoint(
			jaeger.WithEndpoint(config.JaegerEndpoint),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create jaeger exporter: %w", err)
	}

	// Create resource with service information
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			semconv.DeploymentEnvironment(config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create sampler based on sampling rate
	var sampler sdktrace.Sampler
	if config.SamplingRate <= 0 {
		sampler = sdktrace.NeverSample()
	} else if config.SamplingRate >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	} else {
		sampler = sdktrace.TraceIDRatioBased(config.SamplingRate)
	}

	// Create trace provider
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Set global trace provider
	otel.SetTracerProvider(provider)

	tracer := provider.Tracer(config.ServiceName)

	return &Tracer{
		tracer:   tracer,
		provider: provider,
		config:   config,
	}, nil
}

// StartSpan starts a new span with the given name
func (t *Tracer) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, name, opts...)
}

// Shutdown gracefully shuts down the tracer
func (t *Tracer) Shutdown(ctx context.Context) error {
	if t.provider != nil {
		return t.provider.Shutdown(ctx)
	}
	return nil
}

// Span represents a trace span with helper methods
type Span struct {
	span trace.Span
}

// WrapSpan wraps an OpenTelemetry span
func WrapSpan(span trace.Span) *Span {
	return &Span{span: span}
}

// End ends the span
func (s *Span) End() {
	s.span.End()
}

// SetStatus sets the span status
func (s *Span) SetStatus(code codes.Code, description string) {
	s.span.SetStatus(code, description)
}

// SetError records an error on the span
func (s *Span) SetError(err error) {
	if err != nil {
		s.span.RecordError(err)
		s.span.SetStatus(codes.Error, err.Error())
	}
}

// AddEvent adds an event to the span
func (s *Span) AddEvent(name string, attrs ...attribute.KeyValue) {
	s.span.AddEvent(name, trace.WithAttributes(attrs...))
}

// SetAttributes sets multiple attributes on the span
func (s *Span) SetAttributes(attrs ...attribute.KeyValue) {
	s.span.SetAttributes(attrs...)
}

// SetAttribute sets a single attribute on the span
func (s *Span) SetAttribute(key string, value any) {
	switch v := value.(type) {
	case string:
		s.span.SetAttributes(attribute.String(key, v))
	case int:
		s.span.SetAttributes(attribute.Int(key, v))
	case int64:
		s.span.SetAttributes(attribute.Int64(key, v))
	case float64:
		s.span.SetAttributes(attribute.Float64(key, v))
	case bool:
		s.span.SetAttributes(attribute.Bool(key, v))
	default:
		s.span.SetAttributes(attribute.String(key, fmt.Sprintf("%v", v)))
	}
}

// SpanContext returns the span context
func (s *Span) SpanContext() trace.SpanContext {
	return s.span.SpanContext()
}

// TraceID returns the trace ID
func (s *Span) TraceID() string {
	return s.span.SpanContext().TraceID().String()
}

// SpanID returns the span ID
func (s *Span) SpanID() string {
	return s.span.SpanContext().SpanID().String()
}

// Helper functions for common tracing patterns

// StartSpanFromContext is a convenience function to start a span from context
func StartSpanFromContext(ctx context.Context, tracer trace.Tracer, name string, opts ...trace.SpanStartOption) (context.Context, *Span) {
	ctx, span := tracer.Start(ctx, name, opts...)
	return ctx, WrapSpan(span)
}

// WithSpan executes a function within a span
func WithSpan(ctx context.Context, tracer trace.Tracer, name string, fn func(context.Context, *Span) error) error {
	ctx, span := StartSpanFromContext(ctx, tracer, name)
	defer span.End()

	err := fn(ctx, span)
	if err != nil {
		span.SetError(err)
	}

	return err
}

// MeasureLatency measures the latency of an operation and records it as a span event
func MeasureLatency(span *Span, name string, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start)

	span.AddEvent(name,
		attribute.Int64("duration_ms", duration.Milliseconds()),
		attribute.Bool("success", err == nil),
	)

	if err != nil {
		span.SetError(err)
	}

	return err
}

// Common attribute keys
const (
	AttrHTTPMethod       = "http.method"
	AttrHTTPURL          = "http.url"
	AttrHTTPStatusCode   = "http.status_code"
	AttrHTTPRoute        = "http.route"
	AttrHTTPUserAgent    = "http.user_agent"
	AttrHTTPRequestID    = "http.request_id"
	AttrDBSystem         = "db.system"
	AttrDBName           = "db.name"
	AttrDBStatement      = "db.statement"
	AttrDBOperation      = "db.operation"
	AttrMessageSystem    = "messaging.system"
	AttrMessageDest      = "messaging.destination"
	AttrMessageOperation = "messaging.operation"
	AttrUserID           = "user.id"
	AttrUserRole         = "user.role"
	AttrErrorType        = "error.type"
	AttrErrorMessage     = "error.message"
	AttrServiceName      = "service.name"
	AttrServiceVersion   = "service.version"
)

// HTTPAttrs returns HTTP-related attributes
func HTTPAttrs(method, url, route string, statusCode int) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String(AttrHTTPMethod, method),
		attribute.String(AttrHTTPURL, url),
		attribute.String(AttrHTTPRoute, route),
		attribute.Int(AttrHTTPStatusCode, statusCode),
	}
}

// DBAttrs returns database-related attributes
func DBAttrs(system, dbName, operation, statement string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String(AttrDBSystem, system),
		attribute.String(AttrDBName, dbName),
		attribute.String(AttrDBOperation, operation),
		attribute.String(AttrDBStatement, statement),
	}
}

// MessageAttrs returns messaging-related attributes
func MessageAttrs(system, destination, operation string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String(AttrMessageSystem, system),
		attribute.String(AttrMessageDest, destination),
		attribute.String(AttrMessageOperation, operation),
	}
}

// UserAttrs returns user-related attributes
func UserAttrs(userID, role string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String(AttrUserID, userID),
		attribute.String(AttrUserRole, role),
	}
}

// ErrorAttrs returns error-related attributes
func ErrorAttrs(err error) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String(AttrErrorType, fmt.Sprintf("%T", err)),
		attribute.String(AttrErrorMessage, err.Error()),
	}
}

// SpanKind constants for different span types
type SpanKind trace.SpanKind

const (
	SpanKindInternal SpanKind = SpanKind(trace.SpanKindInternal)
	SpanKindServer   SpanKind = SpanKind(trace.SpanKindServer)
	SpanKindClient   SpanKind = SpanKind(trace.SpanKindClient)
	SpanKindProducer SpanKind = SpanKind(trace.SpanKindProducer)
	SpanKindConsumer SpanKind = SpanKind(trace.SpanKindConsumer)
)

// WithSpanKind returns a SpanStartOption that sets the span kind
func WithSpanKind(kind SpanKind) trace.SpanStartOption {
	return trace.WithSpanKind(trace.SpanKind(kind))
}

// WithAttributes returns a SpanStartOption that sets attributes
func WithAttributes(attrs ...attribute.KeyValue) trace.SpanStartOption {
	return trace.WithAttributes(attrs...)
}

// TraceIDFromContext extracts the trace ID from context
func TraceIDFromContext(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		return spanCtx.TraceID().String()
	}
	return ""
}

// SpanIDFromContext extracts the span ID from context
func SpanIDFromContext(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		return spanCtx.SpanID().String()
	}
	return ""
}

// IsTracing checks if tracing is active in the context
func IsTracing(ctx context.Context) bool {
	return trace.SpanContextFromContext(ctx).IsValid()
}
