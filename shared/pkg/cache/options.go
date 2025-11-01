package cache

import (
	"time"
)

type Option func(*Config)

func WithHost(host string) Option {
	return func(c *Config) {
		c.Host = host
	}
}

func WithPort(port int) Option {
	return func(c *Config) {
		c.Port = port
	}
}

func WithPassword(password string) Option {
	return func(c *Config) {
		c.Password = password
	}
}

func WithDB(db int) Option {
	return func(c *Config) {
		c.DB = db
	}
}

func WithMaxRetries(maxRetries int) Option {
	return func(c *Config) {
		c.MaxRetries = maxRetries
	}
}

func WithPoolSize(poolSize int) Option {
	return func(c *Config) {
		c.PoolSize = poolSize
	}
}

func WithMinIdleConns(minIdleConns int) Option {
	return func(c *Config) {
		c.MinIdleConns = minIdleConns
	}
}

func WithDialTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.DialTimeout = timeout
	}
}

func WithReadTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.ReadTimeout = timeout
	}
}

func WithWriteTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.WriteTimeout = timeout
	}
}
