package checkers

import (
	"context"
	"fmt"
	"time"

	"echo-backend/services/message-service/internal/health"
	"shared/pkg/cache"
)

type CacheChecker struct {
	cache cache.Cache
}

func NewCacheChecker(cache cache.Cache) *CacheChecker {
	return &CacheChecker{
		cache: cache,
	}
}

func (c *CacheChecker) Name() string {
	return "cache"
}

func (c *CacheChecker) Check(ctx context.Context) health.CheckResult {
	start := time.Now()
	result := health.CheckResult{
		Status:      health.StatusHealthy,
		LastChecked: time.Now().Format(time.RFC3339),
	}

	details := health.CacheDetails{
		Connected: false,
	}

	testKey := fmt.Sprintf("health_check_%d", time.Now().UnixNano())
	testValue := []byte("test")

	setCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := c.cache.Set(setCtx, testKey, testValue, 5*time.Second); err != nil {
		result.Status = health.StatusUnhealthy
		result.Error = fmt.Sprintf("Redis SET operation failed: %v", err)
		result.Message = "Unable to write to Redis cache"
		result.ResponseTime = float64(time.Since(start).Milliseconds())
		result.Details = map[string]interface{}{
			"cache": details,
		}
		return result
	}

	getCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	retrievedValue, err := c.cache.Get(getCtx, testKey)
	if err != nil {
		result.Status = health.StatusDegraded
		result.Message = "Redis SET succeeded but GET operation failed"
		result.Error = fmt.Sprintf("GET operation failed: %v", err)
		result.ResponseTime = float64(time.Since(start).Milliseconds())
		result.Details = map[string]interface{}{
			"cache": details,
		}
		return result
	}

	if string(retrievedValue) != string(testValue) {
		result.Status = health.StatusDegraded
		result.Message = "Redis operations succeeded but data integrity check failed"
		result.Error = "Retrieved value does not match set value"
	} else {
		details.Connected = true
		result.Message = "Redis cache is healthy and operational"
	}

	delCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	c.cache.Delete(delCtx, testKey)

	result.ResponseTime = float64(time.Since(start).Milliseconds())
	result.Details = map[string]interface{}{
		"cache": details,
	}

	return result
}
