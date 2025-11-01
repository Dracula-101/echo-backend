package messaging

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

func WithRetryBackoff(retryBackoff int) Option {
	return func(c *Config) {
		c.RetryBackoff = retryBackoff
	}
}

func WithSessionTimeout(sessionTimeout int) Option {
	return func(c *Config) {
		c.SessionTimeout = sessionTimeout
	}
}

func WithHeartbeatInterval(heartbeatInterval int) Option {
	return func(c *Config) {
		c.HeartbeatInterval = heartbeatInterval
	}
}
