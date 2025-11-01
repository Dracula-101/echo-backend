package metrics

// Counter represents a metric that can only increase
type Counter interface {
	// Inc increments the counter by 1 with the given labels
	Inc(labels map[string]string)
	// Add adds the given value to the counter with the given labels
	Add(value float64, labels map[string]string)
}

// Histogram represents a metric that samples observations
type Histogram interface {
	// Observe records an observation with the given labels
	Observe(value float64, labels map[string]string)
}

// Gauge represents a metric that can go up and down
type Gauge interface {
	// Set sets the gauge to the given value with the given labels
	Set(value float64, labels map[string]string)
	// Inc increments the gauge by 1 with the given labels
	Inc(labels map[string]string)
	// Dec decrements the gauge by 1 with the given labels
	Dec(labels map[string]string)
	// Add adds the given value to the gauge with the given labels
	Add(value float64, labels map[string]string)
	// Sub subtracts the given value from the gauge with the given labels
	Sub(value float64, labels map[string]string)
}

// Summary represents a metric that samples observations and calculates quantiles
type Summary interface {
	// Observe records an observation with the given labels
	Observe(value float64, labels map[string]string)
}
