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

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	isDev := env.IsDevelopment()

	health := h.manager.Health(ctx, true)
	liveness := h.manager.Liveness(ctx)
	readiness := h.manager.Readiness(ctx)

	if !isDev {
		health = h.sanitizeResponse(health)
	}

	resp := HealthHandlerResponse{
		Status:    health.Status,
		Timestamp: health.Timestamp,
		Service:   health.Service,
		Version:   health.Version,
		Uptime:    health.Uptime,
		Liveness: ProbeStatus{
			Status: liveness.Status,
			OK:     liveness.Status == StatusHealthy,
		},
		Readiness: ProbeStatus{
			Status: readiness.Status,
			OK:     readiness.Status == StatusHealthy,
		},
		Environment: getEnvironment(),
	}

	if len(health.Checks) > 0 {
		resp.Checks = health.Checks
	}

	status := h.manager.HTTPStatus(health.Status)
	response.JSONWithMessage(ctx, r, w, status, "Health status", resp)
}

func (h *Handler) Liveness(w http.ResponseWriter, r *http.Request) {
	liveness := h.manager.Liveness(r.Context())
	resp := ProbeStatus{
		Status: liveness.Status,
		OK:     liveness.Status == StatusHealthy,
	}
	status := h.manager.HTTPStatus(liveness.Status)
	response.JSONWithMessage(r.Context(), r, w, status, "Liveness probe", resp)
}

func (h *Handler) Readiness(w http.ResponseWriter, r *http.Request) {
	readiness := h.manager.Readiness(r.Context())
	resp := ProbeStatus{
		Status: readiness.Status,
		OK:     readiness.Status == StatusHealthy,
	}
	status := h.manager.HTTPStatus(readiness.Status)
	response.JSONWithMessage(r.Context(), r, w, status, "Readiness probe", resp)
}

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

		if check.Status == StatusHealthy || check.Status == StatusDegraded {
			sanitizedCheck.Error = ""
		} else {
			sanitizedCheck.Error = "Service check failed"
		}

		if check.Details != nil {
			sanitizedCheck.Details = h.sanitizeDetails(check.Details, name)
		}

		sanitized.Checks[name] = sanitizedCheck
	}

	return sanitized
}

func (h *Handler) sanitizeDetails(details map[string]interface{}, checkName string) map[string]interface{} {
	sanitized := make(map[string]interface{})

	switch checkName {
	case "database":
		if db, ok := details["database"].(DatabaseDetails); ok {
			sanitized["database"] = map[string]interface{}{
				"connected": db.Connected,
			}
		}

	case "cache":
		if cache, ok := details["cache"].(CacheDetails); ok {
			sanitized["cache"] = map[string]interface{}{
				"connected": cache.Connected,
			}
		}

	case "cache_performance":
		sanitized["message"] = "Performance metrics available in development mode"

	default:
		for key, value := range details {
			if key == "connected" {
				sanitized[key] = value
			}
		}
	}

	return sanitized
}

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
