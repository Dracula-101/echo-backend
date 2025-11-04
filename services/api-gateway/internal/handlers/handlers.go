package handlers

import (
	"net/http"

	"shared/server/response"
)

// ============================================================================
// Response Models
// ============================================================================

type VersionResponse struct {
	Version string `json:"version"`
	Service string `json:"service"`
}

// ============================================================================
// Handler Functions
// ============================================================================

func VersionHandler(version, service string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		versionData := VersionResponse{
			Version: version,
			Service: service,
		}

		response.Success().
			WithContext(r.Context()).
			WithRequest(r).
			WithData(versionData).
			WithMessage("Version information retrieved").
			OK(w)
	}
}

func StatusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := map[string]interface{}{
			"status":      "operational",
			"api_version": "v1",
		}

		response.Success().
			WithContext(r.Context()).
			WithRequest(r).
			WithData(status).
			WithMessage("Service is operational").
			OK(w)
	}
}

func MetricsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response.Success().
			WithContext(r.Context()).
			WithRequest(r).
			WithMessage("Metrics endpoint").
			OK(w)
	}
}

func NotFoundHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response.NotFoundError(r.Context(), r, w, "Resource")
	}
}

func MethodNotAllowedHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response.Error().
			WithContext(r.Context()).
			WithRequest(r).
			WithMessage("Method not allowed").
			Send(w, http.StatusMethodNotAllowed)
	}
}
