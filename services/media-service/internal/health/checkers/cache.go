package checkers

import (
	"context"
	"time"

	"media-service/internal/health"

	"shared/pkg/cache"
)

// CacheChecker checks cache connectivity
type CacheChecker struct {
	cache cache.Cache
}

// NewCacheChecker creates a new cache health checker
func NewCacheChecker(cache cache.Cache) *CacheChecker {
	return &CacheChecker{
		cache: cache,
	}
}

// Name returns the name of this checker
func (c *CacheChecker) Name() string {
	return "cache"
}

// Check performs the cache health check
func (c *CacheChecker) Check(ctx context.Context) health.CheckResult {
	start := time.Now()

	// Try to get cache info
	if _, err := c.cache.Info(ctx); err != nil {
		return health.CheckResult{
			Status:    health.StatusUnhealthy,
			Timestamp: time.Now(),
			Error:     err.Error(),
			Details: map[string]interface{}{
				"response_time_ms": time.Since(start).Milliseconds(),
			},
		}
	}

	return health.CheckResult{
		Status:    health.StatusHealthy,
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"response_time_ms": time.Since(start).Milliseconds(),
		},
	}
}

// CachePerformanceChecker checks cache performance
type CachePerformanceChecker struct {
	cache cache.Cache
}

// NewCachePerformanceChecker creates a new cache performance checker
func NewCachePerformanceChecker(cache cache.Cache) *CachePerformanceChecker {
	return &CachePerformanceChecker{
		cache: cache,
	}
}

// Name returns the name of this checker
func (c *CachePerformanceChecker) Name() string {
	return "cache_performance"
}

// Check performs the cache performance check
func (c *CachePerformanceChecker) Check(ctx context.Context) health.CheckResult {
	start := time.Now()
	testKey := "health:check:test"
	testValue := []byte("test")

	// Try a write operation
	if err := c.cache.Set(ctx, testKey, testValue, 10*time.Second); err != nil {
		return health.CheckResult{
			Status:    health.StatusDegraded,
			Timestamp: time.Now(),
			Error:     "cache write failed: " + err.Error(),
		}
	}

	// Try a read operation
	if _, err := c.cache.Get(ctx, testKey); err != nil {
		return health.CheckResult{
			Status:    health.StatusDegraded,
			Timestamp: time.Now(),
			Error:     "cache read failed: " + err.Error(),
		}
	}

	// Cleanup
	c.cache.Delete(ctx, testKey)

	responseTime := time.Since(start).Milliseconds()
	status := health.StatusHealthy
	if responseTime > 100 {
		status = health.StatusDegraded
	}

	return health.CheckResult{
		Status:    status,
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"response_time_ms": responseTime,
		},
	}
}
