package middleware

import (
	"net/http"
)

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposeHeaders    []string
	AllowCredentials bool
}

// DefaultCORSConfig returns default CORS configuration
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"X-Request-ID",
			"X-User-ID",
			"X-Device-ID",
			"X-Platform",
			"X-App-Version",
			"X-Internal-Service-Secret",
			"X-Source-Service",
		},
		ExposeHeaders: []string{
			"X-Request-ID",
			"X-Total-Count",
		},
		AllowCredentials: false,
	}
}

// CORS middleware adds CORS headers to responses
func CORS(config CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set CORS headers
			if len(config.AllowedOrigins) > 0 {
				origin := r.Header.Get("Origin")
				if origin != "" {
					// Check if origin is allowed
					allowed := false
					for _, allowedOrigin := range config.AllowedOrigins {
						if allowedOrigin == "*" || allowedOrigin == origin {
							w.Header().Set("Access-Control-Allow-Origin", origin)
							allowed = true
							break
						}
					}
					if !allowed && len(config.AllowedOrigins) > 0 {
						w.Header().Set("Access-Control-Allow-Origin", config.AllowedOrigins[0])
					}
				}
			}

			if len(config.AllowedMethods) > 0 {
				methods := ""
				for i, method := range config.AllowedMethods {
					if i > 0 {
						methods += ", "
					}
					methods += method
				}
				w.Header().Set("Access-Control-Allow-Methods", methods)
			}

			if len(config.AllowedHeaders) > 0 {
				headers := ""
				for i, header := range config.AllowedHeaders {
					if i > 0 {
						headers += ", "
					}
					headers += header
				}
				w.Header().Set("Access-Control-Allow-Headers", headers)
			}

			if len(config.ExposeHeaders) > 0 {
				headers := ""
				for i, header := range config.ExposeHeaders {
					if i > 0 {
						headers += ", "
					}
					headers += header
				}
				w.Header().Set("Access-Control-Expose-Headers", headers)
			}

			if config.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
