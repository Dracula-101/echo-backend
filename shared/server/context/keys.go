package contextx

type ContextKey string

const (
	RequestIDKey     ContextKey = "request_id"
	UserIDKey        ContextKey = "user_id"
	ClientIPKey      ContextKey = "client_ip"
	SessionIDKey     ContextKey = "session_id"
	TraceIDKey       ContextKey = "trace_id"
	SpanIDKey        ContextKey = "span_id"
	CorrelationIDKey ContextKey = "correlation_id"
	AuthTokenKey     ContextKey = "auth_token"
	StartTimeKey     ContextKey = "start_time"
	ServiceKey       ContextKey = "service"
	VersionKey       ContextKey = "version"
	EnvKey           ContextKey = "env"
	APIVersionKey    ContextKey = "api_version"
	ResponseKey      ContextKey = "response"
)
