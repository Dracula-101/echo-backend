package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	client *redis.Client
}

func NewRateLimiter(client *redis.Client) *RateLimiter {
	return &RateLimiter{client: client}
}

func (r *RateLimiter) Allow(ctx context.Context, key string, limit int64, window time.Duration) (bool, error) {
	script := `
        local key = KEYS[1]
        local limit = tonumber(ARGV[1])
        local window = tonumber(ARGV[2])
        local current = redis.call('incr', key)
        if current == 1 then
            redis.call('expire', key, window)
        end
        if current > limit then
            return 0
        end
        return 1
    `

	result, err := r.client.Eval(ctx, script, []string{key}, limit, int64(window.Seconds())).Result()
	if err != nil {
		return false, err
	}

	return result.(int64) == 1, nil
}

func (r *RateLimiter) AllowN(ctx context.Context, key string, limit int64, window time.Duration, n int64) (bool, error) {
	script := `
        local key = KEYS[1]
        local limit = tonumber(ARGV[1])
        local window = tonumber(ARGV[2])
        local increment = tonumber(ARGV[3])
        local current = redis.call('incrby', key, increment)
        if current == increment then
            redis.call('expire', key, window)
        end
        if current > limit then
            return 0
        end
        return 1
    `

	result, err := r.client.Eval(ctx, script, []string{key}, limit, int64(window.Seconds()), n).Result()
	if err != nil {
		return false, err
	}

	return result.(int64) == 1, nil
}

func (r *RateLimiter) Remaining(ctx context.Context, key string, limit int64) (int64, error) {
	count, err := r.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return limit, nil
	}
	if err != nil {
		return 0, err
	}

	remaining := limit - count
	if remaining < 0 {
		return 0, nil
	}
	return remaining, nil
}

func (r *RateLimiter) Reset(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}
