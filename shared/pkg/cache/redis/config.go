package redis

import (
	"time"

	"shared/pkg/cache"
)

func DefaultConfig() cache.Config {
	return cache.Config{
		Host:         "localhost",
		Port:         6379,
		Password:     "",
		DB:           0,
		MaxRetries:   3,
		PoolSize:     10,
		MinIdleConns: 2,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}
}
