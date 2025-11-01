package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type summaryVec struct {
	vec *prometheus.SummaryVec
}

// NewSummary creates a new Prometheus summary
func NewSummary(namespace, subsystem, name, help string, labelNames []string, objectives map[float64]float64) *summaryVec {
	vec := promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  namespace,
			Subsystem:  subsystem,
			Name:       name,
			Help:       help,
			Objectives: objectives,
		},
		labelNames,
	)
	return &summaryVec{vec: vec}
}

func (s *summaryVec) Observe(value float64, labels map[string]string) {
	s.vec.With(prometheus.Labels(labels)).Observe(value)
}
