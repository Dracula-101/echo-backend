package handlers

import (
	"net/http"

	"shared/server/response"
)

type VersionResponse struct {
	Version string `json:"version"`
	Service string `json:"service"`
}

// VersionHandler returns version information
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

// StatusHandler returns operational status
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

// MetricsHandler returns metrics endpoint
func MetricsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response.Success().
			WithContext(r.Context()).
			WithRequest(r).
			WithMessage("Metrics endpoint").
			OK(w)
	}
}

// NotFoundHandler returns 404 for undefined routes
func NotFoundHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response.NotFoundError(r.Context(), r, w, "Resource")
	}
}

// MethodNotAllowedHandler returns 405 for invalid methods
func MethodNotAllowedHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response.Error().
			WithContext(r.Context()).
			WithRequest(r).
			WithMessage("Method not allowed").
			Send(w, http.StatusMethodNotAllowed)
	}
}
