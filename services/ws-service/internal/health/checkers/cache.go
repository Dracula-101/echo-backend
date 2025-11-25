package checkers

import (
	"context"
	"ws-service/internal/health"

	"shared/pkg/cache"
)

type CacheChecker struct {
	cache cache.Cache
}

func NewCacheChecker(cache cache.Cache) *CacheChecker {
	return &CacheChecker{cache: cache}
}

func (c *CacheChecker) Name() string {
	return "cache"
}

func (c *CacheChecker) Check(ctx context.Context) (health.Status, string) {
	if err := c.cache.Ping(ctx); err != nil {
		return health.StatusUnhealthy, "Cache connection failed: " + err.Error()
	}
	return health.StatusHealthy, "Cache connection successful"
}
