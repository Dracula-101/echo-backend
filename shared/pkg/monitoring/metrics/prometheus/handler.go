package prometheus

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Handler returns an HTTP handler that serves Prometheus metrics.
// It exposes the default registry which includes Go runtime metrics
// (goroutines, GC, memory) and any custom metrics registered by the service.
func Handler() http.Handler {
	return promhttp.Handler()
}
