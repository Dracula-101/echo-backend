package checkers

import (
	"context"
	"presence-service/internal/health"

	"shared/pkg/cache"
)

type CacheChecker struct {
	cache cache.Cache
}

func NewCacheChecker(cache cache.Cache) health.Checker {
	return &CacheChecker{cache: cache}
}

func (c *CacheChecker) Name() string {
	return "cache"
}

func (c *CacheChecker) Check(ctx context.Context) health.CheckResult {
	if err := c.cache.Ping(ctx); err != nil {
		return health.CheckResult{
			Name:    c.Name(),
			Status:  health.StatusUnhealthy,
			Message: "Cache connection failed",
			Details: map[string]interface{}{
				"error": err.Error(),
			},
		}
	}

	return health.CheckResult{
		Name:    c.Name(),
		Status:  health.StatusHealthy,
		Message: "Cache is responsive",
	}
}
