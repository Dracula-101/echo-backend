package redis

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"

	"shared/pkg/cache"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/pkg/logger/adapter"
)

type client struct {
	rdb    *redis.Client
	logger logger.Logger
}

func New(config cache.Config) (cache.Cache, error) {
	lgr, _ := adapter.NewZap(logger.Config{
		Level:      logger.GetLoggerLevel(),
		Format:     logger.FormatText,
		Output:     os.Stdout,
		TimeFormat: time.RFC3339,
		Service:    "redis-client",
	})
	lgr.Debug(
		"Initializing Redis client",
		logger.String("host", config.Host),
		logger.Int("port", config.Port),
		logger.Int("db", config.DB),
	)
	rdb := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password:     config.Password,
		DB:           config.DB,
		MaxRetries:   config.MaxRetries,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	lgr.Debug("Pinging Redis server to verify connection", logger.String("address", rdb.Options().Addr))
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return &client{rdb: rdb, logger: lgr}, nil
}

func (c *client) Get(ctx context.Context, key string) ([]byte, error) {
	c.logger.Debug("Getting key from Redis", logger.String("key", key))
	result, err := c.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, cache.ErrNotFound
	}
	return result, err
}

func (c *client) Set(ctx context.Context, key string, value []byte, ttl time.Duration) pkgErrors.AppError {
	c.logger.Debug("Setting key in Redis", logger.String("key", key))
	if err := c.rdb.Set(ctx, key, value, ttl).Err(); err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to set cache key").
			WithService("redis-client").
			WithDetail("key", key)
	}
	return nil
}

func (c *client) Delete(ctx context.Context, key string) pkgErrors.AppError {
	c.logger.Debug("Deleting key from Redis", logger.String("key", key))
	if err := c.rdb.Del(ctx, key).Err(); err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to delete cache key").
			WithService("redis-client").
			WithDetail("key", key)
	}
	return nil
}

func (c *client) Exists(ctx context.Context, key string) (bool, error) {
	c.logger.Debug("Checking if key exists in Redis", logger.String("key", key))
	count, err := c.rdb.Exists(ctx, key).Result()
	return count > 0, err
}

func (c *client) Expire(ctx context.Context, key string, ttl time.Duration) pkgErrors.AppError {
	c.logger.Debug("Setting expiration for key in Redis", logger.String("key", key), logger.Duration("ttl", ttl))
	if err := c.rdb.Expire(ctx, key, ttl).Err(); err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to set expiration").
			WithService("redis-client").
			WithDetail("key", key)
	}
	return nil
}

func (c *client) TTL(ctx context.Context, key string) (time.Duration, error) {
	c.logger.Debug("Getting TTL for key in Redis", logger.String("key", key))
	return c.rdb.TTL(ctx, key).Result()
}

func (c *client) GetMulti(ctx context.Context, keys []string) (map[string][]byte, error) {
	if len(keys) == 0 {
		return make(map[string][]byte), nil
	}

	results, err := c.rdb.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	data := make(map[string][]byte, len(keys))
	for i, key := range keys {
		if results[i] != nil {
			if val, ok := results[i].(string); ok {
				data[key] = []byte(val)
			}
		}
	}

	return data, nil
}

func (c *client) SetMulti(ctx context.Context, items map[string][]byte, ttl time.Duration) pkgErrors.AppError {
	c.logger.Debug("Setting multiple keys in Redis", logger.Int("count", len(items)))
	pipe := c.rdb.Pipeline()

	for key, value := range items {
		pipe.Set(ctx, key, value, ttl)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to set multiple keys").
			WithService("redis-client").
			WithDetail("count", len(items))
	}
	return nil
}

func (c *client) DeleteMulti(ctx context.Context, keys []string) pkgErrors.AppError {
	c.logger.Debug("Deleting multiple keys from Redis", logger.Int("count", len(keys)))
	if len(keys) == 0 {
		return nil
	}
	if err := c.rdb.Del(ctx, keys...).Err(); err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to delete multiple keys").
			WithService("redis-client").
			WithDetail("count", len(keys))
	}
	return nil
}

func (c *client) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	c.logger.Debug("Incrementing key in Redis", logger.String("key", key), logger.Int64("delta", delta))
	return c.rdb.IncrBy(ctx, key, delta).Result()
}

func (c *client) Decrement(ctx context.Context, key string, delta int64) (int64, error) {
	c.logger.Debug("Decrementing key in Redis", logger.String("key", key), logger.Int64("delta", delta))
	return c.rdb.DecrBy(ctx, key, delta).Result()
}

func (c *client) Ping(ctx context.Context) pkgErrors.AppError {
	c.logger.Debug("Pinging Redis server")
	if err := c.rdb.Ping(ctx).Err(); err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to ping redis").
			WithService("redis-client")
	}
	return nil
}

func (c *client) Info(ctx context.Context) (map[string]string, error) {
	c.logger.Debug("Fetching Redis INFO")
	result, err := c.rdb.Info(ctx).Result()
	if err != nil {
		return nil, err
	}

	info := make(map[string]string)
	lines := splitLines(result)
	var currentSection string

	for _, line := range lines {
		if len(line) == 0 || line[0] == '#' {
			if len(line) > 2 && line[0] == '#' {
				currentSection = line[2:]
			}
			continue
		}
		parts := splitKeyValue(line)
		if len(parts) == 2 {
			key := parts[0]
			if currentSection != "" {
				key = currentSection + "." + key
			}
			info[key] = parts[1]
		}
	}

	return info, nil
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func splitKeyValue(s string) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}

func (c *client) Flush(ctx context.Context) pkgErrors.AppError {
	c.logger.Debug("Flushing Redis database")
	if err := c.rdb.FlushDB(ctx).Err(); err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to flush redis").
			WithService("redis-client")
	}
	return nil
}

func (c *client) Close() error {
	c.logger.Debug("Closing Redis client")
	return c.rdb.Close()
}
