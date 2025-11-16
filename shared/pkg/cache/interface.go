package cache

import (
	"context"
	"time"

	pkgErrors "shared/pkg/errors"
)

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) pkgErrors.AppError

	GetString(ctx context.Context, key string) (string, pkgErrors.AppError)
	SetString(ctx context.Context, key string, value string, ttl time.Duration) pkgErrors.AppError

	GetInt(ctx context.Context, key string) (int64, pkgErrors.AppError)
	SetInt(ctx context.Context, key string, value int64, ttl time.Duration) pkgErrors.AppError

	GetBool(ctx context.Context, key string) (bool, pkgErrors.AppError)
	SetBool(ctx context.Context, key string, value bool, ttl time.Duration) pkgErrors.AppError

	Delete(ctx context.Context, key string) pkgErrors.AppError
	Exists(ctx context.Context, key string) (bool, error)
	Expire(ctx context.Context, key string, ttl time.Duration) pkgErrors.AppError
	TTL(ctx context.Context, key string) (time.Duration, error)

	GetMulti(ctx context.Context, keys []string) (map[string][]byte, error)
	SetMulti(ctx context.Context, items map[string][]byte, ttl time.Duration) pkgErrors.AppError
	DeleteMulti(ctx context.Context, keys []string) pkgErrors.AppError

	Increment(ctx context.Context, key string, delta int64) (int64, error)
	Decrement(ctx context.Context, key string, delta int64) (int64, error)

	Ping(ctx context.Context) pkgErrors.AppError
	Info(ctx context.Context) (map[string]string, error)

	Flush(ctx context.Context) pkgErrors.AppError
	Close() error
}

type Config struct {
	Host         string
	Port         int
	Password     string
	DB           int
	MaxRetries   int
	PoolSize     int
	MinIdleConns int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}
