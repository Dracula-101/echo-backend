package memory

import (
	"context"
	"sync"
	"time"

	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/pkg/windowstore"
)

type MemoryWindowStore struct {
	mu   sync.Mutex
	data map[string][]int64
	log  logger.Logger
}

func NewMemory(log logger.Logger) windowstore.WindowStore {
	return &MemoryWindowStore{
		data: make(map[string][]int64),
		log:  log,
	}
}

func (m *MemoryWindowStore) Add(ctx context.Context, key string, ts int64, ttl time.Duration) pkgErrors.AppError {
	start := time.Now()

	m.mu.Lock()
	m.data[key] = append(m.data[key], ts)
	m.mu.Unlock()

	m.log.Debug("windowstore.add",
		logger.String("backend", "memory"),
		logger.String("key", key),
		logger.Int64("ts", ts),
		logger.Duration("latency", time.Since(start)),
	)

	return nil
}

func (m *MemoryWindowStore) AddMulti(ctx context.Context, key string, timestamps []int64, ttl time.Duration) pkgErrors.AppError {
	start := time.Now()

	if len(timestamps) == 0 {
		return nil
	}

	m.mu.Lock()
	m.data[key] = append(m.data[key], timestamps...)
	m.mu.Unlock()

	m.log.Debug("windowstore.addmulti",
		logger.String("backend", "memory"),
		logger.String("key", key),
		logger.Int("count", len(timestamps)),
		logger.Duration("latency", time.Since(start)),
	)

	return nil
}

func (m *MemoryWindowStore) Count(ctx context.Context, key string, from, to int64) (int64, pkgErrors.AppError) {
	start := time.Now()

	m.mu.Lock()
	defer m.mu.Unlock()

	var count int64
	for _, ts := range m.data[key] {
		if ts >= from && ts <= to {
			count++
		}
	}

	m.log.Debug("windowstore.count",
		logger.String("backend", "memory"),
		logger.String("key", key),
		logger.Int64("count", count),
		logger.Duration("latency", time.Since(start)),
	)

	return count, nil
}

func (m *MemoryWindowStore) TrimBefore(ctx context.Context, key string, before int64) pkgErrors.AppError {
	start := time.Now()

	m.mu.Lock()

	arr := m.data[key]
	i := 0
	for ; i < len(arr); i++ {
		if arr[i] >= before {
			break
		}
	}
	m.data[key] = arr[i:]

	m.mu.Unlock()

	m.log.Debug("windowstore.trim",
		logger.String("backend", "memory"),
		logger.String("key", key),
		logger.Int64("before", before),
		logger.Duration("latency", time.Since(start)),
	)

	return nil
}

func (m *MemoryWindowStore) Size(ctx context.Context, key string) (int64, pkgErrors.AppError) {
	start := time.Now()

	m.mu.Lock()
	size := int64(len(m.data[key]))
	m.mu.Unlock()

	m.log.Debug("windowstore.size",
		logger.String("backend", "memory"),
		logger.String("key", key),
		logger.Int64("size", size),
		logger.Duration("latency", time.Since(start)),
	)

	return size, nil
}

func (m *MemoryWindowStore) Clear(ctx context.Context, key string) pkgErrors.AppError {
	start := time.Now()

	m.mu.Lock()
	delete(m.data, key)
	m.mu.Unlock()

	m.log.Debug("windowstore.clear",
		logger.String("backend", "memory"),
		logger.String("key", key),
		logger.Duration("latency", time.Since(start)),
	)

	return nil
}

func (m *MemoryWindowStore) Close() pkgErrors.AppError {
	m.log.Info("windowstore closing",
		logger.String("backend", "memory"),
	)

	return nil
}
