package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type counterVec struct {
	vec *prometheus.CounterVec
}

// NewCounter creates a new Prometheus counter
func NewCounter(namespace, subsystem, name, help string, labelNames []string) *counterVec {
	vec := promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      help,
		},
		labelNames,
	)
	return &counterVec{vec: vec}
}

func (c *counterVec) Inc(labels map[string]string) {
	c.vec.With(prometheus.Labels(labels)).Inc()
}

func (c *counterVec) Add(value float64, labels map[string]string) {
	c.vec.With(prometheus.Labels(labels)).Add(value)
}
