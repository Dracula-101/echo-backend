package database

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

func WithUser(user string) Option {
	return func(c *Config) {
		c.User = user
	}
}

func WithPassword(password string) Option {
	return func(c *Config) {
		c.Password = password
	}
}

func WithDatabase(database string) Option {
	return func(c *Config) {
		c.Database = database
	}
}

func WithSSLMode(sslMode string) Option {
	return func(c *Config) {
		c.SSLMode = sslMode
	}
}

func WithMaxOpenConns(maxOpenConns int) Option {
	return func(c *Config) {
		c.MaxOpenConns = maxOpenConns
	}
}

func WithMaxIdleConns(maxIdleConns int) Option {
	return func(c *Config) {
		c.MaxIdleConns = maxIdleConns
	}
}

func WithConnMaxLifetime(lifetime time.Duration) Option {
	return func(c *Config) {
		c.ConnMaxLifetime = lifetime
	}
}

func WithConnMaxIdleTime(idleTime time.Duration) Option {
	return func(c *Config) {
		c.ConnMaxIdleTime = idleTime
	}
}
