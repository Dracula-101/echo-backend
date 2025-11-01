package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type gaugeVec struct {
	vec *prometheus.GaugeVec
}

// NewGauge creates a new Prometheus gauge
func NewGauge(namespace, subsystem, name, help string, labelNames []string) *gaugeVec {
	vec := promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      help,
		},
		labelNames,
	)
	return &gaugeVec{vec: vec}
}

func (g *gaugeVec) Set(value float64, labels map[string]string) {
	g.vec.With(prometheus.Labels(labels)).Set(value)
}

func (g *gaugeVec) Inc(labels map[string]string) {
	g.vec.With(prometheus.Labels(labels)).Inc()
}

func (g *gaugeVec) Dec(labels map[string]string) {
	g.vec.With(prometheus.Labels(labels)).Dec()
}

func (g *gaugeVec) Add(value float64, labels map[string]string) {
	g.vec.With(prometheus.Labels(labels)).Add(value)
}

func (g *gaugeVec) Sub(value float64, labels map[string]string) {
	g.vec.With(prometheus.Labels(labels)).Sub(value)
}
