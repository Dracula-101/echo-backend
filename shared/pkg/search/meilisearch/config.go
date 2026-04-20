package meilisearch

import "time"

type Config struct {
	Host    string
	APIKey  string
	Timeout time.Duration

	// Task polling config
	TaskPollInterval time.Duration
	TaskPollTimeout  time.Duration
}

func (c *Config) setDefaults() {
	if c.Timeout == 0 {
		c.Timeout = 10 * time.Second
	}
	if c.TaskPollInterval == 0 {
		c.TaskPollInterval = 50 * time.Millisecond
	}
	if c.TaskPollTimeout == 0 {
		c.TaskPollTimeout = 30 * time.Second
	}
}
