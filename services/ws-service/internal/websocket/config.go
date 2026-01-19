package websocket

import "time"

type Config struct {
	MaxConnections           int
	MaxConnectionsPerUser    int
	MaxSubscriptionsPerConn  int
	MaxMessageSize           int64
	DispatcherWorkers        int
	DispatcherQueueSize      int
	SessionTTL               time.Duration
	HealthCheckInterval      time.Duration
	RateLimitPerSecond       float64
	RateLimitBurst           int
	TypingTimeout            time.Duration
	TypingCleanupInterval    time.Duration
	RateLimiterCleanupInterval time.Duration
	MessageBufferSize        int
	MessageRetryAttempts     int
	MessageRetryDelay        time.Duration
	MetricsEnabled           bool
	MetricsPrefix            string
	GracefulShutdownTimeout  time.Duration
	HeartbeatInterval        time.Duration
	HeartbeatTimeout         time.Duration
	CompressionEnabled       bool
	CompressionThreshold     int
}

func DefaultConfig() *Config {
	return &Config{
		MaxConnections:           10000,
		MaxConnectionsPerUser:    5,
		MaxSubscriptionsPerConn:  100,
		MaxMessageSize:           65536,
		DispatcherWorkers:        10,
		DispatcherQueueSize:      1000,
		SessionTTL:               24 * time.Hour,
		HealthCheckInterval:      30 * time.Second,
		RateLimitPerSecond:       20,
		RateLimitBurst:           50,
		TypingTimeout:            5 * time.Second,
		TypingCleanupInterval:    5 * time.Second,
		RateLimiterCleanupInterval: 5 * time.Minute,
		MessageBufferSize:        100,
		MessageRetryAttempts:     3,
		MessageRetryDelay:        100 * time.Millisecond,
		MetricsEnabled:           true,
		MetricsPrefix:            "ws",
		GracefulShutdownTimeout:  30 * time.Second,
		HeartbeatInterval:        30 * time.Second,
		HeartbeatTimeout:         10 * time.Second,
		CompressionEnabled:       true,
		CompressionThreshold:     1024,
	}
}

type Option func(*Config)

func WithMaxConnections(n int) Option {
	return func(c *Config) { c.MaxConnections = n }
}

func WithMaxConnectionsPerUser(n int) Option {
	return func(c *Config) { c.MaxConnectionsPerUser = n }
}

func WithMaxSubscriptionsPerConn(n int) Option {
	return func(c *Config) { c.MaxSubscriptionsPerConn = n }
}

func WithMaxMessageSize(size int64) Option {
	return func(c *Config) { c.MaxMessageSize = size }
}

func WithDispatcher(workers, queueSize int) Option {
	return func(c *Config) {
		c.DispatcherWorkers = workers
		c.DispatcherQueueSize = queueSize
	}
}

func WithSessionTTL(ttl time.Duration) Option {
	return func(c *Config) { c.SessionTTL = ttl }
}

func WithHealthCheckInterval(interval time.Duration) Option {
	return func(c *Config) { c.HealthCheckInterval = interval }
}

func WithRateLimit(perSecond float64, burst int) Option {
	return func(c *Config) {
		c.RateLimitPerSecond = perSecond
		c.RateLimitBurst = burst
	}
}

func WithTypingTimeout(timeout time.Duration) Option {
	return func(c *Config) { c.TypingTimeout = timeout }
}

func WithMessageBuffer(size, retryAttempts int, retryDelay time.Duration) Option {
	return func(c *Config) {
		c.MessageBufferSize = size
		c.MessageRetryAttempts = retryAttempts
		c.MessageRetryDelay = retryDelay
	}
}

func WithMetrics(enabled bool, prefix string) Option {
	return func(c *Config) {
		c.MetricsEnabled = enabled
		c.MetricsPrefix = prefix
	}
}

func WithGracefulShutdownTimeout(timeout time.Duration) Option {
	return func(c *Config) { c.GracefulShutdownTimeout = timeout }
}

func WithHeartbeat(interval, timeout time.Duration) Option {
	return func(c *Config) {
		c.HeartbeatInterval = interval
		c.HeartbeatTimeout = timeout
	}
}

func WithCompression(enabled bool, threshold int) Option {
	return func(c *Config) {
		c.CompressionEnabled = enabled
		c.CompressionThreshold = threshold
	}
}
