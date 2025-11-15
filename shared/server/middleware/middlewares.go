package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"
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
					response.InternalServerError(r.Context(), r, w, "Internal server error", errors.New(fmt.Sprint(err)))
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
			if sessionID := GetSessionID(r.Context()); sessionID != "" {
				fields = append(fields, logger.String("session_id", sessionID))
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

// PathBasedBodyLimit applies different body size limits based on request path
// pathLimits maps request paths to their specific limits
// defaultLimit is used for paths not in pathLimits
func PathBasedBodyLimit(defaultLimit int64, pathLimits map[string]int64) Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			limit := defaultLimit

			// Check if there's a specific limit for this path
			if pathLimits != nil {
				if pathLimit, exists := pathLimits[r.URL.Path]; exists {
					limit = pathLimit
				}
			}

			r.Body = http.MaxBytesReader(w, r.Body, limit)
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

func GetSessionID(ctx context.Context) string {
	if id, ok := ctx.Value(sContext.SessionIDKey).(string); ok {
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

func TokenBucketRateLimit(requests int, window time.Duration) Handler {
	type bucket struct {
		tokens        int
		lastTokenTime time.Time
		mu            sync.Mutex
	}

	buckets := make(map[string]*bucket)
	var mu sync.Mutex
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.RemoteAddr
			now := time.Now()

			mu.Lock()
			b, exists := buckets[key]
			if !exists {
				b = &bucket{
					tokens:        requests,
					lastTokenTime: now,
				}
				buckets[key] = b
			}
			mu.Unlock()

			b.mu.Lock()
			elapsed := now.Sub(b.lastTokenTime)
			newTokens := int(elapsed / (window / time.Duration(requests)))
			if newTokens > 0 {
				b.tokens += newTokens
				if b.tokens > requests {
					b.tokens = requests
				}
				b.lastTokenTime = now
			}

			remaining := b.tokens
			resetTime := b.lastTokenTime.Add(window)
			b.mu.Unlock()

			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", requests))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))

			if remaining <= 0 {
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(window.Seconds())))
				w.WriteHeader(http.StatusTooManyRequests)
				response.TooManyRequestsError(r.Context(), r, w, "rate limit exceeded", int(window.Seconds()))
				return
			}

			b.mu.Lock()
			b.tokens--
			b.mu.Unlock()

			next.ServeHTTP(w, r)
		})
	}
}

func FixedWindowRateLimit(requests int, window time.Duration) Handler {
	type windowData struct {
		count     int
		resetTime time.Time
		mu        sync.Mutex
	}
	var clients = make(map[string]*windowData)
	var mu sync.Mutex
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.RemoteAddr
			now := time.Now()
			mu.Lock()
			data, exists := clients[key]
			if !exists {
				data = &windowData{
					count:     0,
					resetTime: now.Add(window),
				}
				clients[key] = data
			}
			mu.Unlock()
			data.mu.Lock()
			if now.After(data.resetTime) {
				data.count = 0
				data.resetTime = now.Add(window)
			}
			remaining := requests - data.count
			resetTime := data.resetTime
			data.mu.Unlock()

			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", requests))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))

			data.mu.Lock()
			if data.count >= requests {
				data.mu.Unlock()
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(resetTime.Sub(now).Seconds())))
				w.WriteHeader(http.StatusTooManyRequests)
				response.TooManyRequestsError(r.Context(), r, w, "rate limit exceeded", int(window.Seconds()))
				return
			}
			data.count++
			data.mu.Unlock()
			next.ServeHTTP(w, r)
		})
	}
}

func SlidingWindowRateLimit(requests int, window time.Duration) Handler {
	type clientData struct {
		timestamps []time.Time
		mu         sync.Mutex
	}

	var clients = make(map[string]*clientData)
	var mu sync.Mutex

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.RemoteAddr
			now := time.Now()
			mu.Lock()
			data, exists := clients[key]
			if !exists {
				data = &clientData{
					timestamps: make([]time.Time, 0),
				}
				clients[key] = data
			}
			mu.Unlock()
			data.mu.Lock()
			validTimestamps := make([]time.Time, 0)
			for _, t := range data.timestamps {
				if now.Sub(t) < window {
					validTimestamps = append(validTimestamps, t)
				}
			}
			data.timestamps = validTimestamps
			remaining := requests - len(data.timestamps)
			var resetTime time.Time
			if len(data.timestamps) > 0 {
				resetTime = data.timestamps[0].Add(window)
			} else {
				resetTime = now.Add(window)
			}
			data.mu.Unlock()

			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", requests))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))

			data.mu.Lock()
			if len(data.timestamps) >= requests {
				data.mu.Unlock()
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(resetTime.Sub(now).Seconds())))
				w.WriteHeader(http.StatusTooManyRequests)
				response.TooManyRequestsError(r.Context(), r, w, "rate limit exceeded", int(window.Seconds()))
				return
			}
			data.timestamps = append(data.timestamps, now)
			data.mu.Unlock()
			next.ServeHTTP(w, r)
		})
	}
}

func RoundRobinRateLimit(requestsPerInstance int, instanceCountFunc func() int) Handler {
	var mu sync.Mutex
	var requestCount int
	var resetTime time.Time

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			instanceCount := instanceCountFunc()
			if instanceCount <= 0 {
				instanceCount = 1
			}
			totalRequests := requestsPerInstance * instanceCount

			mu.Lock()
			if resetTime.IsZero() || time.Now().After(resetTime) {
				requestCount = 0
				resetTime = time.Now().Add(time.Minute)
			}
			remaining := totalRequests - requestCount
			mu.Unlock()

			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", totalRequests))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))

			mu.Lock()
			if requestCount >= totalRequests {
				mu.Unlock()
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(time.Until(resetTime).Seconds())))
				w.WriteHeader(http.StatusTooManyRequests)
				response.TooManyRequestsError(r.Context(), r, w, "rate limit exceeded", 60)
				return
			}
			requestCount++
			mu.Unlock()

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
					response.BadRequestError(r.Context(), r, w, "Content-Type header required", errors.New("missing content type"))
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
	SkipPaths     []string
}

// Auth validates bearer tokens and adds the authenticated user ID to both
// the request context and the X-User-ID header. The header ensures the user ID
// is forwarded when proxying requests to downstream services.
//
// Downstream services can use ExtractUserIDFromHeader middleware to retrieve
// the user ID from the header into their request context.
func Auth(config AuthConfig) Handler {
	if config.ValidateToken == nil {
		panic("AuthConfig.ValidateToken cannot be nil")
	}

	if config.OnAuthFailed == nil {
		config.OnAuthFailed = func(w http.ResponseWriter, r *http.Request, err error) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			response.UnauthorizedError(r.Context(), r, w, "Authentication failed", err)
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, skipPattern := range config.SkipPaths {
				if matchPath(r.URL.Path, skipPattern) {
					next.ServeHTTP(w, r)
					return
				}
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
			r.Header.Set("X-User-ID", userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func matchPath(requestPath, pattern string) bool {
	if pattern == "" {
		return false
	}
	if pattern == requestPath {
		return true
	}
	if matched, err := path.Match(pattern, requestPath); err == nil && matched {
		return true
	}
	return false
}

func InterceptUserId() Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := r.Header.Get("X-User-ID")
			if userID != "" {
				ctx := SetUserID(r.Context(), userID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			response.UnauthorizedError(r.Context(), r, w, "Missing or Invalid Auth Token", errors.New("missing user id in header"))
		})
	}
}

func InterceptSessionId() Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sessionID := r.Header.Get("X-Session-ID")
			if sessionID != "" {
				ctx := context.WithValue(r.Context(), sContext.SessionIDKey, sessionID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			response.UnauthorizedError(r.Context(), r, w, "Missing or Invalid Session Token", errors.New("missing session id in header"))
		})
	}
}

func InterceptSessionToken() Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sessionToken := r.Header.Get("X-Session-Token")
			ctx := context.WithValue(r.Context(), sContext.SessionTokenKey, sessionToken)
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
