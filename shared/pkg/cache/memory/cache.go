package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"shared/pkg/cache"
	pkgErrors "shared/pkg/errors"
)

type item struct {
	value      []byte
	expiration int64
}

type memoryCache struct {
	mu    sync.RWMutex
	items map[string]*item
}

func New() cache.Cache {
	c := &memoryCache{
		items: make(map[string]*item),
	}

	go c.cleanup()

	return c
}

func (c *memoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found {
		return nil, cache.ErrNotFound
	}

	if item.expiration > 0 && time.Now().UnixNano() > item.expiration {
		return nil, cache.ErrNotFound
	}

	return item.value, nil
}

func (c *memoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) pkgErrors.AppError {
	c.mu.Lock()
	defer c.mu.Unlock()

	var expiration int64
	if ttl > 0 {
		expiration = time.Now().Add(ttl).UnixNano()
	}

	c.items[key] = &item{
		value:      value,
		expiration: expiration,
	}

	return nil
}

func (c *memoryCache) GetString(ctx context.Context, key string) (string, pkgErrors.AppError) {
	data, err := c.Get(ctx, key)
	if err != nil {
		if err == cache.ErrNotFound {
			return "", pkgErrors.FromError(err, pkgErrors.CodeNotFound, "key not found").
				WithService("memory-cache").
				WithDetail("key", key)
		}
		return "", pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to get string key").
			WithService("memory-cache").
			WithDetail("key", key)
	}
	return string(data), nil
}

func (c *memoryCache) SetString(ctx context.Context, key string, value string, ttl time.Duration) pkgErrors.AppError {
	return c.Set(ctx, key, []byte(value), ttl)
}

func (c *memoryCache) GetInt(ctx context.Context, key string) (int64, pkgErrors.AppError) {
	data, err := c.Get(ctx, key)
	if err != nil {
		if err == cache.ErrNotFound {
			return 0, pkgErrors.FromError(err, pkgErrors.CodeNotFound, "key not found").
				WithService("memory-cache").
				WithDetail("key", key)
		}
		return 0, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to get int key").
			WithService("memory-cache").
			WithDetail("key", key)
	}

	var result int64
	_, parseErr := fmt.Sscanf(string(data), "%d", &result)
	if parseErr != nil {
		return 0, pkgErrors.FromError(parseErr, pkgErrors.CodeCacheError, "failed to parse int value").
			WithService("memory-cache").
			WithDetail("key", key)
	}
	return result, nil
}

func (c *memoryCache) SetInt(ctx context.Context, key string, value int64, ttl time.Duration) pkgErrors.AppError {
	return c.Set(ctx, key, []byte(fmt.Sprintf("%d", value)), ttl)
}

func (c *memoryCache) GetBool(ctx context.Context, key string) (bool, pkgErrors.AppError) {
	data, err := c.Get(ctx, key)
	if err != nil {
		if err == cache.ErrNotFound {
			return false, pkgErrors.FromError(err, pkgErrors.CodeNotFound, "key not found").
				WithService("memory-cache").
				WithDetail("key", key)
		}
		return false, pkgErrors.FromError(err, pkgErrors.CodeCacheError, "failed to get bool key").
			WithService("memory-cache").
			WithDetail("key", key)
	}

	str := string(data)
	if str == "true" || str == "1" {
		return true, nil
	} else if str == "false" || str == "0" {
		return false, nil
	}

	return false, pkgErrors.New(pkgErrors.CodeCacheError, "invalid bool value").
		WithService("memory-cache").
		WithDetail("key", key).
		WithDetail("value", str)
}

func (c *memoryCache) SetBool(ctx context.Context, key string, value bool, ttl time.Duration) pkgErrors.AppError {
	var strValue string
	if value {
		strValue = "true"
	} else {
		strValue = "false"
	}
	return c.Set(ctx, key, []byte(strValue), ttl)
}

func (c *memoryCache) Delete(ctx context.Context, key string) pkgErrors.AppError {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
	return nil
}

func (c *memoryCache) Exists(ctx context.Context, key string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found {
		return false, nil
	}

	if item.expiration > 0 && time.Now().UnixNano() > item.expiration {
		return false, nil
	}

	return true, nil
}

func (c *memoryCache) Expire(ctx context.Context, key string, ttl time.Duration) pkgErrors.AppError {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, found := c.items[key]
	if !found {
		return pkgErrors.FromError(cache.ErrNotFound, pkgErrors.CodeNotFound, "key not found").
			WithService("memory-cache").
			WithDetail("key", key)
	}

	item.expiration = time.Now().Add(ttl).UnixNano()
	return nil
}

func (c *memoryCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found {
		return 0, cache.ErrNotFound
	}

	if item.expiration == 0 {
		return cache.NoExpiration, nil
	}

	ttl := time.Duration(item.expiration - time.Now().UnixNano())
	if ttl < 0 {
		return 0, nil
	}

	return ttl, nil
}

func (c *memoryCache) GetMulti(ctx context.Context, keys []string) (map[string][]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string][]byte)
	now := time.Now().UnixNano()

	for _, key := range keys {
		if item, found := c.items[key]; found {
			if item.expiration == 0 || item.expiration > now {
				result[key] = item.value
			}
		}
	}

	return result, nil
}

func (c *memoryCache) SetMulti(ctx context.Context, items map[string][]byte, ttl time.Duration) pkgErrors.AppError {
	c.mu.Lock()
	defer c.mu.Unlock()

	var expiration int64
	if ttl > 0 {
		expiration = time.Now().Add(ttl).UnixNano()
	}

	for key, value := range items {
		c.items[key] = &item{
			value:      value,
			expiration: expiration,
		}
	}

	return nil
}

func (c *memoryCache) DeleteMulti(ctx context.Context, keys []string) pkgErrors.AppError {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, key := range keys {
		delete(c.items, key)
	}

	return nil
}

func (c *memoryCache) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	return 0, cache.ErrNotSupported
}

func (c *memoryCache) Decrement(ctx context.Context, key string, delta int64) (int64, error) {
	return 0, cache.ErrNotSupported
}

func (c *memoryCache) Ping(ctx context.Context) pkgErrors.AppError {
	return nil
}

func (c *memoryCache) Info(ctx context.Context) (map[string]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	info := make(map[string]string)
	info["item_count"] = fmt.Sprintf("%d", len(c.items))
	info["implementation"] = "in-memory"
	info["notes"] = "This is a simple in-memory cache implementation and does not support advanced features."
	info["timestamp"] = time.Now().Format(time.RFC3339)
	info["uptime"] = time.Since(time.Now().Add(-time.Duration(len(c.items)) * time.Minute)).String()
	return info, nil
}

func (c *memoryCache) Flush(ctx context.Context) pkgErrors.AppError {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*item)
	return nil
}

func (c *memoryCache) Close() error {
	return nil
}

func (c *memoryCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now().UnixNano()

		for key, item := range c.items {
			if item.expiration > 0 && now > item.expiration {
				delete(c.items, key)
			}
		}

		c.mu.Unlock()
	}
}
