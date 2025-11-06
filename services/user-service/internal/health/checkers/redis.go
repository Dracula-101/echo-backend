package checkers

import (
	"user-service/internal/health"
	"context"
	"fmt"
	"shared/pkg/cache"
	"time"
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

	// Test basic set/get operations to verify cache connectivity
	testKey := fmt.Sprintf("health_check_%d", time.Now().UnixNano())
	testValue := []byte("test")

	// Test SET operation
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

	// Test GET operation
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

	// Verify the value matches
	if string(retrievedValue) != string(testValue) {
		result.Status = health.StatusDegraded
		result.Message = "Redis operations succeeded but data integrity check failed"
		result.Error = "Retrieved value does not match set value"
	} else {
		details.Connected = true
		result.Message = "Redis cache is healthy and operational"
	}

	// Clean up test key
	delCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	c.cache.Delete(delCtx, testKey)

	// Test EXISTS operation
	existsCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	if _, err := c.cache.Exists(existsCtx, "health_check_probe"); err != nil {
		if result.Status == health.StatusHealthy {
			result.Status = health.StatusDegraded
			result.Message = "Redis basic operations work but EXISTS check failed"
			result.Error = fmt.Sprintf("EXISTS operation failed: %v", err)
		}
	}

	result.ResponseTime = float64(time.Since(start).Milliseconds())
	result.Details = map[string]interface{}{
		"cache":             details,
		"operations_tested": []string{"SET", "GET", "DELETE", "EXISTS"},
	}

	return result
}

// CachePerformanceChecker checks cache performance metrics
type CachePerformanceChecker struct {
	cache cache.Cache
}

func NewCachePerformanceChecker(cache cache.Cache) *CachePerformanceChecker {
	return &CachePerformanceChecker{
		cache: cache,
	}
}

func (c *CachePerformanceChecker) Name() string {
	return "cache_performance"
}

func (c *CachePerformanceChecker) Check(ctx context.Context) health.CheckResult {
	start := time.Now()
	result := health.CheckResult{
		Status:      health.StatusHealthy,
		LastChecked: time.Now().Format(time.RFC3339),
	}

	// Test multiple operations to check performance
	testKey := fmt.Sprintf("perf_check_%d", time.Now().UnixNano())
	testValue := []byte("performance_test_value")

	operations := make(map[string]float64)

	// Test SET performance
	setStart := time.Now()
	setCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	if err := c.cache.Set(setCtx, testKey, testValue, 10*time.Second); err != nil {
		cancel()
		result.Status = health.StatusUnhealthy
		result.Error = fmt.Sprintf("Performance test SET failed: %v", err)
		result.ResponseTime = float64(time.Since(start).Milliseconds())
		return result
	}
	cancel()
	operations["set_ms"] = float64(time.Since(setStart).Microseconds()) / 1000.0

	// Test GET performance
	getStart := time.Now()
	getCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	if _, err := c.cache.Get(getCtx, testKey); err != nil {
		cancel()
		result.Status = health.StatusDegraded
		result.Error = fmt.Sprintf("Performance test GET failed: %v", err)
	}
	cancel()
	operations["get_ms"] = float64(time.Since(getStart).Microseconds()) / 1000.0

	// Test DELETE performance
	delStart := time.Now()
	delCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	c.cache.Delete(delCtx, testKey)
	cancel()
	operations["delete_ms"] = float64(time.Since(delStart).Microseconds()) / 1000.0

	totalTime := time.Since(start).Milliseconds()

	// Check if operations are taking too long
	if operations["set_ms"] > 100 || operations["get_ms"] > 100 {
		result.Status = health.StatusDegraded
		result.Message = "Redis cache performance is degraded (high latency)"
	} else {
		result.Message = "Redis cache performance is healthy"
	}

	result.ResponseTime = float64(totalTime)
	result.Details = map[string]interface{}{
		"operations":    operations,
		"total_time_ms": totalTime,
	}

	return result
}
