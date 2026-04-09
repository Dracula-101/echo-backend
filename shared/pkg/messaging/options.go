package messaging

import "time"

type Option func(*Config)

func WithBrokers(brokers []string) Option {
	return func(c *Config) {
		c.Brokers = brokers
	}
}

func WithClientID(clientID string) Option {
	return func(c *Config) {
		c.ClientID = clientID
	}
}

func WithGroupID(groupID string) Option {
	return func(c *Config) {
		c.GroupID = groupID
	}
}

func WithMaxRetries(maxRetries int) Option {
	return func(c *Config) {
		c.MaxRetries = maxRetries
	}
}

func WithRetryBackoff(retryBackoff time.Duration) Option {
	return func(c *Config) {
		c.RetryBackoff = retryBackoff
	}
}

func WithSessionTimeout(sessionTimeout time.Duration) Option {
	return func(c *Config) {
		c.SessionTimeout = sessionTimeout
	}
}

func WithHeartbeatInterval(heartbeatInterval time.Duration) Option {
	return func(c *Config) {
		c.HeartbeatInterval = heartbeatInterval
	}
}
