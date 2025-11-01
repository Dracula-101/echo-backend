package memory

import (
	"context"
	"sync"
	"time"

	"shared/pkg/cache"
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

func (c *memoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
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

func (c *memoryCache) Delete(ctx context.Context, key string) error {
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

func (c *memoryCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, found := c.items[key]
	if !found {
		return cache.ErrNotFound
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

func (c *memoryCache) SetMulti(ctx context.Context, items map[string][]byte, ttl time.Duration) error {
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

func (c *memoryCache) DeleteMulti(ctx context.Context, keys []string) error {
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

func (c *memoryCache) Flush(ctx context.Context) error {
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
