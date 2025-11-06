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

	if len(health.Checks) > 0 {
		if isDev {
			resp["checks"] = health.Checks
		} else {
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

	resp["environment"] = getEnvironment()

	status := h.manager.HTTPStatus(health.Status)
	response.JSONWithMessage(ctx, r, w, status, "Health status", resp)
}

func (h *Handler) Liveness(w http.ResponseWriter, r *http.Request) {
	liveness := h.manager.Liveness(r.Context())

	resp := map[string]interface{}{
		"status": liveness.Status,
		"ok":     liveness.Status == StatusHealthy,
	}

	status := h.manager.HTTPStatus(liveness.Status)
	response.JSONWithMessage(r.Context(), r, w, status, "Liveness probe", resp)
}

func (h *Handler) Readiness(w http.ResponseWriter, r *http.Request) {
	readiness := h.manager.Readiness(r.Context())

	resp := map[string]interface{}{
		"status": readiness.Status,
		"ok":     readiness.Status == StatusHealthy,
	}

	if env.IsDevelopment() && len(readiness.Checks) > 0 {
		resp["checks"] = readiness.Checks
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

	case "kafka":
		if kafka, ok := details["kafka"].(KafkaDetails); ok {
			sanitized["kafka"] = map[string]interface{}{
				"connected": kafka.Connected,
			}
		}

	case "websocket":
		if ws, ok := details["websocket"].(WebSocketDetails); ok {
			sanitized["websocket"] = map[string]interface{}{
				"active_connections": ws.ActiveConnections,
				"max_connections":    ws.MaxConnections,
			}
		}

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
