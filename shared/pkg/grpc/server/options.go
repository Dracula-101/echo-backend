package server

import (
	"time"
)

type Option func(*Config)

func WithPort(port int) Option {
	return func(c *Config) {
		c.Port = port
	}
}

func WithMaxConnectionIdle(duration time.Duration) Option {
	return func(c *Config) {
		c.MaxConnectionIdle = duration
	}
}

func WithMaxConnectionAge(duration time.Duration) Option {
	return func(c *Config) {
		c.MaxConnectionAge = duration
	}
}

func WithMaxConcurrentStreams(n uint32) Option {
	return func(c *Config) {
		c.MaxConcurrentStreams = n
	}
}

func WithMaxRecvMsgSize(size int) Option {
	return func(c *Config) {
		c.MaxRecvMsgSize = size
	}
}

func WithMaxSendMsgSize(size int) Option {
	return func(c *Config) {
		c.MaxSendMsgSize = size
	}
}

func DefaultConfig() Config {
	return Config{
		Port:                  50051,
		MaxConnectionIdle:     5 * time.Minute,
		MaxConnectionAge:      30 * time.Minute,
		MaxConnectionAgeGrace: 5 * time.Second,
		Time:                  2 * time.Hour,
		Timeout:               20 * time.Second,
		MaxConcurrentStreams:  100,
		MaxRecvMsgSize:        4 * 1024 * 1024,
		MaxSendMsgSize:        4 * 1024 * 1024,
	}
}

func ApplyOptions(config Config, opts ...Option) Config {
	for _, opt := range opts {
		opt(&config)
	}
	return config
}
