package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"shared/pkg/logger"
)

type Client struct {
	rdb *redis.Client
	log logger.Logger
}

func New(rdb *redis.Client, log logger.Logger) *Client {
	return &Client{
		rdb: rdb,
		log: log,
	}
}

func (c *Client) Raw() *redis.Client {
	return c.rdb
}

func (c *Client) SAdd(ctx context.Context, key string, members ...any) error {
	return c.rdb.SAdd(ctx, key, members...).Err()
}

func (c *Client) SRem(ctx context.Context, key string, members ...any) error {
	return c.rdb.SRem(ctx, key, members...).Err()
}

func (c *Client) SMembers(ctx context.Context, key string) ([]string, error) {
	return c.rdb.SMembers(ctx, key).Result()
}

func (c *Client) SIsMember(ctx context.Context, key string, member any) (bool, error) {
	return c.rdb.SIsMember(ctx, key, member).Result()
}

func (c *Client) SCard(ctx context.Context, key string) (int64, error) {
	return c.rdb.SCard(ctx, key).Result()
}

func (c *Client) HSet(ctx context.Context, key string, values ...any) error {
	return c.rdb.HSet(ctx, key, values...).Err()
}

func (c *Client) HGet(ctx context.Context, key, field string) (string, error) {
	result, err := c.rdb.HGet(ctx, key, field).Result()
	if err == redis.Nil {
		return "", nil
	}
	return result, err
}

func (c *Client) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return c.rdb.HGetAll(ctx, key).Result()
}

func (c *Client) HDel(ctx context.Context, key string, fields ...string) error {
	return c.rdb.HDel(ctx, key, fields...).Err()
}

func (c *Client) HExists(ctx context.Context, key, field string) (bool, error) {
	return c.rdb.HExists(ctx, key, field).Result()
}

func (c *Client) HLen(ctx context.Context, key string) (int64, error) {
	return c.rdb.HLen(ctx, key).Result()
}

func (c *Client) Del(ctx context.Context, keys ...string) error {
	return c.rdb.Del(ctx, keys...).Err()
}

func (c *Client) Exists(ctx context.Context, keys ...string) (int64, error) {
	return c.rdb.Exists(ctx, keys...).Result()
}

func (c *Client) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return c.rdb.Expire(ctx, key, ttl).Err()
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	result, err := c.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return result, err
}

func (c *Client) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	return c.rdb.Set(ctx, key, value, ttl).Err()
}

func (c *Client) SetNX(ctx context.Context, key string, value any, ttl time.Duration) (bool, error) {
	return c.rdb.SetNX(ctx, key, value, ttl).Result()
}

func (c *Client) Eval(ctx context.Context, script string, keys []string, args ...any) (any, error) {
	return c.rdb.Eval(ctx, script, keys, args...).Result()
}

func (c *Client) Pipeline() redis.Pipeliner {
	return c.rdb.Pipeline()
}

func (c *Client) Publish(ctx context.Context, channel string, message any) error {
	return c.rdb.Publish(ctx, channel, message).Err()
}

func (c *Client) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return c.rdb.Subscribe(ctx, channels...)
}

func (c *Client) Keys(ctx context.Context, pattern string) ([]string, error) {
	return c.rdb.Keys(ctx, pattern).Result()
}

func (c *Client) Scan(ctx context.Context, cursor uint64, match string, count int64) ([]string, uint64, error) {
	return c.rdb.Scan(ctx, cursor, match, count).Result()
}
