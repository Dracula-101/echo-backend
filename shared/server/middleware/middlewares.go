package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	cache "shared/pkg/cache"
	"shared/pkg/logger"
	sContext "shared/server/context"
	"shared/server/response"
)

type Handler func(http.Handler) http.Handler

func RequestID(header string) Handler {
	if header == "" {
		header = "X-Request-ID"
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get(header)
			if requestID == "" {
				requestID = uuid.New().String()
			}
			ctx := context.WithValue(r.Context(), sContext.RequestIDKey, requestID)
			w.Header().Set(header, requestID)
			r.Header.Set(header, requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func CorrelationID(header string) Handler {
	if header == "" {
		header = "X-Correlation-ID"
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			correlationID := r.Header.Get(header)
			if correlationID == "" {
				correlationID = uuid.New().String()
			}
			ctx := context.WithValue(r.Context(), sContext.CorrelationIDKey, correlationID)
			w.Header().Set(header, correlationID)
			r.Header.Set(header, correlationID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func Timeout(timeout time.Duration) Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			done := make(chan struct{})
			go func() {
				next.ServeHTTP(w, r.WithContext(ctx))
				close(done)
			}()

			select {
			case <-done:
				return
			case <-ctx.Done():
				serverName := ""
				if r.TLS != nil {
					serverName = r.TLS.ServerName
				}
				w.WriteHeader(http.StatusRequestTimeout)
				response.GatewayTimeoutError(r.Context(), r, w, serverName)
			}
		})
	}
}

func Cache(duration time.Duration, client cache.Cache) Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cacheKey := fmt.Sprintf("server-cache:%s", r.URL.String())
			if cachedResponse, err := client.Get(r.Context(), cacheKey); err == nil && cachedResponse != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write(cachedResponse)
				return
			}

			w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%.f", duration.Seconds()))
			next.ServeHTTP(w, r)
		})
	}
}

type RecoveryConfig struct {
	PrintStack bool
	StackSize  int
	OnPanic    func(r *http.Request, err any, stack []byte)
}

func Recovery(log logger.Logger) Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					stack := debug.Stack()
					log.Error("Panic recovered in HTTP handler",
						logger.String("method", r.Method),
						logger.String("path", r.URL.Path),
						logger.Any("error", err),
						logger.String("stack", string(stack)),
					)

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					response.InternalServerError(r.Context(), r, w, errors.New(fmt.Sprint("internal server error: ", err)))
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
func RequestReceivedLogger(log logger.Logger) Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ctx := context.WithValue(r.Context(), sContext.StartTimeKey, start)
			fields := []logger.Field{
				logger.String("method", r.Method),
				logger.String("path", r.URL.Path),
				logger.String("remote_addr", r.RemoteAddr),
				logger.String("user_agent", r.UserAgent()),
			}
			log.Info("HTTP request received", fields...)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequestCompletedLogger(log logger.Logger) Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			wrapped := &responseWriterWithSize{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
				bodySize:       0,
			}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)

			fields := []logger.Field{
				logger.String("remote_addr", r.RemoteAddr),
				logger.String("user_agent", r.UserAgent()),
			}

			if requestID := GetRequestID(r.Context()); requestID != "" {
				fields = append(fields, logger.String("request_id", requestID))
			}
			if correlationID := GetCorrelationID(r.Context()); correlationID != "" {
				fields = append(fields, logger.String("correlation_id", correlationID))
			}
			if userID := GetUserID(r.Context()); userID != "" {
				fields = append(fields, logger.String("user_id", userID))
			}

			statusCode := wrapped.statusCode
			log.Request(
				r.Context(),
				r.Method,
				r.URL.Path,
				statusCode,
				duration,
				wrapped.bodySize,
				"Completed HTTP request",
				fields...,
			)
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

type responseWriterWithSize struct {
	http.ResponseWriter
	statusCode int
	bodySize   int64
	written    bool
}

func (rw *responseWriterWithSize) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriterWithSize) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	size, err := rw.ResponseWriter.Write(b)
	rw.bodySize += int64(size)
	return size, err
}

func CORS(allowedOrigins []string, allowedMethods []string, allowedHeaders []string) Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			allowed := false
			for _, allowedOrigin := range allowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					break
				}
			}

			if allowed {
				if origin != "" {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				} else {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				}
				w.Header().Set("Access-Control-Allow-Methods", joinStrings(allowedMethods))
				w.Header().Set("Access-Control-Allow-Headers", joinStrings(allowedHeaders))
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func joinStrings(items []string) string {
	result := ""
	for i, item := range items {
		if i > 0 {
			result += ", "
		}
		result += item
	}
	return result
}

func SecurityHeaders(headers map[string]string) Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			w.Header().Set("Content-Security-Policy", "default-src 'self'")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
			for k, v := range headers {
				w.Header().Set(k, v)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func BodyLimit(maxBytes int64) Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

func RealIP(trustedProxies []string) Handler {
	trustedMap := make(map[string]bool)
	for _, proxy := range trustedProxies {
		trustedMap[proxy] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			remoteIP := r.RemoteAddr
			if colonIdx := strings.LastIndex(remoteIP, ":"); colonIdx != -1 {
				remoteIP = remoteIP[:colonIdx]
			}

			if len(trustedMap) == 0 || trustedMap[remoteIP] {
				if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
					ips := strings.Split(xff, ",")
					if len(ips) > 0 {
						r.RemoteAddr = strings.TrimSpace(ips[0])
					}
				} else if xri := r.Header.Get("X-Real-IP"); xri != "" {
					r.RemoteAddr = xri
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(sContext.RequestIDKey).(string); ok {
		return id
	}
	return ""
}

func GetCorrelationID(ctx context.Context) string {
	if id, ok := ctx.Value(sContext.CorrelationIDKey).(string); ok {
		return id
	}
	return ""
}

func GetUserID(ctx context.Context) string {
	if id, ok := ctx.Value(sContext.UserIDKey).(string); ok {
		return id
	}
	return ""
}

func GetStartTime(ctx context.Context) time.Time {
	if t, ok := ctx.Value(sContext.StartTimeKey).(time.Time); ok {
		return t
	}
	return time.Time{}
}

func SetUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, sContext.UserIDKey, userID)
}

type MetricsRecorder interface {
	RecordRequest(method, path string, statusCode int, duration time.Duration)
}

func Metrics(recorder MetricsRecorder) Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)
			recorder.RecordRequest(r.Method, r.URL.Path, wrapped.statusCode, duration)
		})
	}
}

type KeyFuncHandler func(remoteAddr string, path string) string

type RateLimitConfig struct {
	RequestsPerWindow int
	Window            time.Duration
	KeyFunc           KeyFuncHandler
	OnLimitExceeded   func(w http.ResponseWriter, r *http.Request)
}

type rateLimitEntry struct {
	count     int
	resetTime time.Time
	mu        sync.Mutex
}

func RateLimit(config RateLimitConfig) Handler {
	store := &sync.Map{}

	if config.KeyFunc == nil {
		config.KeyFunc = func(remoteAddr string, path string) string {
			return fmt.Sprintf("%s-%s", remoteAddr, path)
		}
	}

	if config.OnLimitExceeded == nil {
		config.OnLimitExceeded = func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			retryAfter := int(time.Until(time.Now().Add(config.Window)).Seconds())
			response.TooManyRequestsError(r.Context(), r, w, "rate limit exceeded", retryAfter)
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := config.KeyFunc(r.RemoteAddr, r.URL.Path)
			now := time.Now()

			val, _ := store.LoadOrStore(key, &rateLimitEntry{
				count:     0,
				resetTime: now.Add(config.Window),
			})
			entry := val.(*rateLimitEntry)

			entry.mu.Lock()
			if now.After(entry.resetTime) {
				entry.count = 0
				entry.resetTime = now.Add(config.Window)
			}

			if entry.count >= config.RequestsPerWindow {
				remaining := int(entry.resetTime.Sub(now).Seconds())
				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", config.RequestsPerWindow))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", entry.resetTime.Unix()))
				w.Header().Set("Retry-After", fmt.Sprintf("%d", remaining))
				entry.mu.Unlock()
				config.OnLimitExceeded(w, r)
				return
			}

			entry.count++
			remaining := config.RequestsPerWindow - entry.count
			entry.mu.Unlock()

			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", config.RequestsPerWindow))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", entry.resetTime.Unix()))

			next.ServeHTTP(w, r)
		})
	}
}

type CompressionConfig struct {
	Level        int
	MinSize      int
	ContentTypes []string
}

func Compression(config CompressionConfig) Handler {
	if config.Level == 0 {
		config.Level = 6
	}
	if config.MinSize == 0 {
		config.MinSize = 1024
	}
	if len(config.ContentTypes) == 0 {
		config.ContentTypes = []string{
			"text/html",
			"text/css",
			"text/plain",
			"text/javascript",
			"application/javascript",
			"application/json",
			"application/xml",
			"text/xml",
		}
	}

	contentTypeMap := make(map[string]bool)
	for _, ct := range config.ContentTypes {
		contentTypeMap[ct] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				next.ServeHTTP(w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func ContentTypeValidator(allowedTypes []string) Handler {
	allowedMap := make(map[string]bool)
	for _, ct := range allowedTypes {
		allowedMap[ct] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
				contentType := r.Header.Get("Content-Type")
				if contentType == "" {
					w.WriteHeader(http.StatusBadRequest)
					response.BadRequestError(r.Context(), r, w, "Content-Type header required")
					return
				}

				if idx := strings.Index(contentType, ";"); idx != -1 {
					contentType = strings.TrimSpace(contentType[:idx])
				}

				if !allowedMap[contentType] {
					w.WriteHeader(http.StatusUnsupportedMediaType)
					response.UnsupportedMediaTypeError(r.Context(), r, w, "unsupported content type")
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func NoCache() Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")

			next.ServeHTTP(w, r)
		})
	}
}

func CacheControl(maxAge int, public bool) Handler {
	cacheValue := fmt.Sprintf("max-age=%d", maxAge)
	if public {
		cacheValue = "public, " + cacheValue
	} else {
		cacheValue = "private, " + cacheValue
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", cacheValue)
			next.ServeHTTP(w, r)
		})
	}
}

type AuthConfig struct {
	ValidateToken func(token string) (userID string, err error)
	OnAuthFailed  func(w http.ResponseWriter, r *http.Request, err error)
	SkipPaths     map[string]bool
}

func Auth(config AuthConfig) Handler {
	if config.ValidateToken == nil {
		panic("AuthConfig.ValidateToken cannot be nil")
	}

	if config.OnAuthFailed == nil {
		config.OnAuthFailed = func(w http.ResponseWriter, r *http.Request, err error) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			response.UnauthorizedError(r.Context(), r, w, "authentication failed")
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if config.SkipPaths != nil && config.SkipPaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				config.OnAuthFailed(w, r, errors.New("missing authorization header"))
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				config.OnAuthFailed(w, r, errors.New("invalid authorization format"))
				return
			}

			token := parts[1]
			userID, err := config.ValidateToken(token)
			if err != nil {
				config.OnAuthFailed(w, r, err)
				return
			}

			ctx := SetUserID(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func APIVersion(headerName string, defaultVersion string) Handler {
	if headerName == "" {
		headerName = "X-API-Version"
	}
	if defaultVersion == "" {
		defaultVersion = "v1"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			version := r.Header.Get(headerName)
			if version == "" {
				version = defaultVersion
			}

			ctx := context.WithValue(r.Context(), sContext.APIVersionKey, version)
			w.Header().Set(headerName, version)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetAPIVersion(ctx context.Context) string {
	if version, ok := ctx.Value(sContext.APIVersionKey).(string); ok {
		return version
	}
	return "v1"
}
