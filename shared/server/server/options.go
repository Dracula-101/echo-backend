package server

import (
	"net/http"
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

func WithIdleTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.IdleTimeout = timeout
	}
}

func WithShutdownTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.ShutdownTimeout = timeout
	}
}

func WithMaxHeaderBytes(bytes int) Option {
	return func(c *Config) {
		c.MaxHeaderBytes = bytes
	}
}

func WithTLS(certFile, keyFile string) Option {
	return func(c *Config) {
		c.TLSEnabled = true
		c.TLSCertFile = certFile
		c.TLSKeyFile = keyFile
	}
}

func WithHandler(handler http.Handler) Option {
	return func(c *Config) {
		c.Handler = handler
	}
}

func WithTimeouts(read, write, idle time.Duration) Option {
	return func(c *Config) {
		c.ReadTimeout = read
		c.WriteTimeout = write
		c.IdleTimeout = idle
	}
}

func WithProductionDefaults() Option {
	return func(c *Config) {
		c.ReadTimeout = 30 * time.Second
		c.WriteTimeout = 30 * time.Second
		c.IdleTimeout = 120 * time.Second
		c.ShutdownTimeout = 30 * time.Second
		c.MaxHeaderBytes = 1 << 20
	}
}

func WithDevelopmentDefaults() Option {
	return func(c *Config) {
		c.ReadTimeout = 60 * time.Second
		c.WriteTimeout = 60 * time.Second
		c.IdleTimeout = 300 * time.Second
		c.ShutdownTimeout = 10 * time.Second
		c.MaxHeaderBytes = 1 << 20
	}
}

func ApplyOptions(cfg *Config, opts ...Option) {
	for _, opt := range opts {
		opt(cfg)
	}
}
