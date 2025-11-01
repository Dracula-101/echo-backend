package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type histogramVec struct {
	vec *prometheus.HistogramVec
}

// NewHistogram creates a new Prometheus histogram
func NewHistogram(namespace, subsystem, name, help string, labelNames []string, buckets []float64) *histogramVec {
	vec := promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      help,
			Buckets:   buckets,
		},
		labelNames,
	)
	return &histogramVec{vec: vec}
}

func (h *histogramVec) Observe(value float64, labels map[string]string) {
	h.vec.With(prometheus.Labels(labels)).Observe(value)
}
