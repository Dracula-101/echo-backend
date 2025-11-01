// Package health provides comprehensive health check functionality for the auth service.
//
// The health check system monitors PostgreSQL database and Redis cache connectivity
// and performance. It provides multiple endpoints for different use cases:
//
// Endpoints:
//   - GET /health or /healthz - Comprehensive health status with all checks
//   - GET /livez or /health/liveness - Liveness probe (always healthy if running)
//   - GET /readyz or /health/readiness - Readiness probe (checks dependencies)
//
// Security:
//   - In production (APP_ENV=production): Only basic status information is exposed
//   - In development (APP_ENV=development): Full detailed information including metrics
//   - Sensitive system details like connection pool stats are hidden in production
//
// Response Format (Production):
//   {
//     "status": "healthy|degraded|unhealthy",
//     "timestamp": "2025-11-01T12:00:00Z",
//     "service": "auth-service",
//     "version": "1.0.0",
//     "uptime": "1h30m45s",
//     "environment": "production",
//     "liveness": {"status": "healthy", "ok": true},
//     "readiness": {"status": "healthy", "ok": true},
//     "checks": {
//       "database": {
//         "status": "healthy",
//         "message": "PostgreSQL database is healthy",
//         "response_time": 5.2
//       },
//       "cache": {
//         "status": "healthy",
//         "message": "Redis cache is healthy and operational",
//         "response_time": 2.1
//       }
//     }
//   }
//
// Response Format (Development):
//   - Includes all production fields plus:
//   - Full error messages
//   - Detailed connection pool statistics
//   - Database version information
//   - Redis performance metrics
//   - Cache operation details
package health

import (
	"net/http"
	"shared/server/env"
	"shared/server/response"
)

type Handler struct {
	manager *Manager
}

func NewHandler(manager *Manager) *Handler {
	return &Handler{
		manager: manager,
	}
}

// Health returns comprehensive health status with all information
// Sensitive details are only exposed in development environment
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	isDev := env.IsDevelopment()

	// Get all health information
	health := h.manager.Health(ctx, true)
	liveness := h.manager.Liveness(ctx)
	readiness := h.manager.Readiness(ctx)

	// Sanitize sensitive information in production
	if !isDev {
		health = h.sanitizeResponse(health)
	}

	// Build comprehensive response
	resp := map[string]interface{}{
		"status":    health.Status,
		"timestamp": health.Timestamp,
		"service":   health.Service,
		"version":   health.Version,
		"uptime":    health.Uptime,
		"liveness": map[string]interface{}{
			"status": liveness.Status,
			"ok":     liveness.Status == StatusHealthy,
		},
		"readiness": map[string]interface{}{
			"status": readiness.Status,
			"ok":     readiness.Status == StatusHealthy,
		},
	}

	// Add detailed checks information
	if len(health.Checks) > 0 {
		if isDev {
			// In development, show full details
			resp["checks"] = health.Checks
		} else {
			// In production, show sanitized status only
			sanitizedChecks := make(map[string]interface{})
			for name, check := range health.Checks {
				sanitizedChecks[name] = map[string]interface{}{
					"status":        check.Status,
					"message":       check.Message,
					"response_time": check.ResponseTime,
				}
			}
			resp["checks"] = sanitizedChecks
		}
	}

	// Add environment indicator
	resp["environment"] = getEnvironment()

	status := h.manager.HTTPStatus(health.Status)
	response.JSONWithMessage(ctx, r, w, status, "Health status", resp)
}

// Liveness returns the liveness probe status (always healthy if service is running)
func (h *Handler) Liveness(w http.ResponseWriter, r *http.Request) {
	liveness := h.manager.Liveness(r.Context())

	resp := map[string]interface{}{
		"status": liveness.Status,
		"ok":     liveness.Status == StatusHealthy,
	}

	status := h.manager.HTTPStatus(liveness.Status)
	response.JSONWithMessage(r.Context(), r, w, status, "Liveness probe", resp)
}

// Readiness returns the readiness probe status (checks dependencies)
func (h *Handler) Readiness(w http.ResponseWriter, r *http.Request) {
	readiness := h.manager.Readiness(r.Context())

	resp := map[string]interface{}{
		"status": readiness.Status,
		"ok":     readiness.Status == StatusHealthy,
	}

	// In production, only show if ready or not
	if !env.IsDevelopment() {
		status := h.manager.HTTPStatus(readiness.Status)
		response.JSONWithMessage(r.Context(), r, w, status, "Readiness probe", resp)
		return
	}

	// In development, show detailed checks
	if readiness.Checks != nil {
		resp["checks"] = readiness.Checks
	}

	status := h.manager.HTTPStatus(readiness.Status)
	response.JSONWithMessage(r.Context(), r, w, status, "Readiness probe", resp)
}

// sanitizeResponse removes sensitive information from health response
func (h *Handler) sanitizeResponse(resp Response) Response {
	if resp.Checks == nil {
		return resp
	}

	sanitized := resp
	sanitized.Checks = make(map[string]CheckResult)

	for name, check := range resp.Checks {
		sanitizedCheck := CheckResult{
			Status:       check.Status,
			Message:      check.Message,
			ResponseTime: check.ResponseTime,
			LastChecked:  check.LastChecked,
		}

		// Remove error details in production (potential info leak)
		if check.Status == StatusHealthy || check.Status == StatusDegraded {
			sanitizedCheck.Error = ""
		} else {
			// Keep generic error message for unhealthy checks
			sanitizedCheck.Error = "Service check failed"
		}

		// Remove sensitive details
		if check.Details != nil {
			sanitizedCheck.Details = h.sanitizeDetails(check.Details, name)
		}

		sanitized.Checks[name] = sanitizedCheck
	}

	return sanitized
}

// sanitizeDetails removes sensitive information from check details
func (h *Handler) sanitizeDetails(details map[string]interface{}, checkName string) map[string]interface{} {
	sanitized := make(map[string]interface{})

	switch checkName {
	case "database":
		// Only show connection status, not detailed metrics
		if db, ok := details["database"].(DatabaseDetails); ok {
			sanitized["database"] = map[string]interface{}{
				"connected": db.Connected,
			}
		}

	case "cache":
		// Only show connection status
		if cache, ok := details["cache"].(CacheDetails); ok {
			sanitized["cache"] = map[string]interface{}{
				"connected": cache.Connected,
			}
		}

	case "cache_performance":
		// Don't expose performance metrics in production
		sanitized["message"] = "Performance metrics available in development mode"

	default:
		// For other checks, only show if they have a connected status
		for key, value := range details {
			if key == "connected" {
				sanitized[key] = value
			}
		}
	}

	return sanitized
}

// getEnvironment returns the current environment name
func getEnvironment() string {
	if env.IsDevelopment() {
		return "development"
	}
	if env.IsProduction() {
		return "production"
	}
	if env.IsTest() {
		return "test"
	}
	return "unknown"
}
