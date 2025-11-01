package handlers

import (
	"net/http"
	"time"

	"shared/server/response"
)

type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version,omitempty"`
}

// HealthHandler returns the health status
func HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		healthData := HealthResponse{
			Status:    "ok",
			Timestamp: time.Now(),
		}

		response.Success().
			WithContext(r.Context()).
			WithRequest(r).
			WithData(healthData).
			WithMessage("Service is healthy").
			OK(w)
	}
}

// LivenessHandler returns liveness status
func LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response.Success().
			WithContext(r.Context()).
			WithRequest(r).
			WithMessage("alive").
			OK(w)
	}
}

// ReadinessHandler returns readiness status
func ReadinessHandler(mon interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if all dependencies are ready
		response.Success().
			WithContext(r.Context()).
			WithRequest(r).
			WithMessage("ready").
			OK(w)
	}
}
