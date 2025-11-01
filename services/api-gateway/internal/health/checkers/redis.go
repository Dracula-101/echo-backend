package checkers

import (
	"context"
	"time"

	"echo-backend/services/api-gateway/internal/health"

	"github.com/redis/go-redis/v9"
)

type RedisChecker struct {
	client *redis.Client
}

func NewRedisChecker(client *redis.Client) *RedisChecker {
	return &RedisChecker{client: client}
}

func (c *RedisChecker) Name() string {
	return "redis"
}

func (c *RedisChecker) Check(ctx context.Context) health.CheckResult {
	start := time.Now()

	if err := c.client.Ping(ctx).Err(); err != nil {
		return health.CheckResult{
			Status:       health.StatusUnhealthy,
			Message:      "Redis connection failed",
			Error:        err.Error(),
			ResponseTime: time.Since(start).Seconds() * 1000,
			LastChecked:  time.Now().Format(time.RFC3339),
		}
	}

	return health.CheckResult{
		Status:       health.StatusHealthy,
		Message:      "Redis connection OK",
		ResponseTime: time.Since(start).Seconds() * 1000,
		LastChecked:  time.Now().Format(time.RFC3339),
	}
}
