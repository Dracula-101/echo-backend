package middleware

import (
	"context"
	"net/http"

	"shared/pkg/logger"
	"shared/server/response"
)

// ServiceAuthKey is the context key for internal service authentication
type ServiceAuthKey struct{}

// InternalServiceAuth validates that requests come from internal services
func InternalServiceAuth(log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check for internal service header
			serviceSecret := r.Header.Get("X-Internal-Service-Secret")
			sourceService := r.Header.Get("X-Source-Service")

			// TODO: Validate service secret against configured value
			// For now, just check if headers are present
			if serviceSecret == "" || sourceService == "" {
				log.Warn("Unauthorized internal service request",
					logger.String("path", r.URL.Path),
					logger.String("source_service", sourceService),
				)
				response.UnauthorizedError(r.Context(), r, w, "Unauthorized service request", nil)
				return
			}

			// Add service info to context
			ctx := context.WithValue(r.Context(), ServiceAuthKey{}, sourceService)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalServiceAuth allows both authenticated and unauthenticated requests
func OptionalServiceAuth(log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			serviceSecret := r.Header.Get("X-Internal-Service-Secret")
			sourceService := r.Header.Get("X-Source-Service")

			if serviceSecret != "" && sourceService != "" {
				// Add service info to context if authenticated
				ctx := context.WithValue(r.Context(), ServiceAuthKey{}, sourceService)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Continue without authentication
			next.ServeHTTP(w, r)
		})
	}
}

// GetSourceService retrieves the source service from context
func GetSourceService(ctx context.Context) (string, bool) {
	service, ok := ctx.Value(ServiceAuthKey{}).(string)
	return service, ok
}
