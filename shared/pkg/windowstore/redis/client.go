package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/pkg/windowstore"
)

type RedisWindowStore struct {
	client *redis.Client
	log    logger.Logger
}

func NewRedis(cfg windowstore.Config, log logger.Logger) windowstore.WindowStore {
	return &RedisWindowStore{
		client: redis.NewClient(&redis.Options{
			Addr:     cfg.Addr,
			Password: cfg.Password,
			DB:       cfg.DB,
		}),
		log: log,
	}
}

func (r *RedisWindowStore) Add(ctx context.Context, key string, ts int64, ttl time.Duration) pkgErrors.AppError {
	err := r.client.ZAdd(ctx, key, redis.Z{
		Score:  float64(ts),
		Member: fmt.Sprintf("%d:%d", ts, time.Now().UnixNano()),
	}).Err()

	if err != nil {
		r.log.Error("windowstore.add failed",
			logger.String("service", "windowstore"),
			logger.String("key", key),
			logger.Int64("ts", ts),
			logger.Error(err),
		)
		return pkgErrors.FromError(err, ErrWindowStoreAdd, "add failed")
	}

	_ = r.client.Expire(ctx, key, ttl).Err()
	return nil
}

func (r *RedisWindowStore) AddMulti(ctx context.Context, key string, timestamps []int64, ttl time.Duration) pkgErrors.AppError {
	if len(timestamps) == 0 {
		return nil
	}

	z := make([]redis.Z, 0, len(timestamps))
	for _, ts := range timestamps {
		z = append(z, redis.Z{
			Score:  float64(ts),
			Member: fmt.Sprintf("%d:%d", ts, time.Now().UnixNano()),
		})
	}

	err := r.client.ZAdd(ctx, key, z...).Err()
	if err != nil {
		r.log.Error("windowstore.addmulti failed",
			logger.String("service", "windowstore"),
			logger.String("key", key),
			logger.Error(err),
		)
		return pkgErrors.FromError(err, ErrWindowStoreAddMulti, "addmulti failed")
	}

	_ = r.client.Expire(ctx, key, ttl).Err()
	return nil
}

func (r *RedisWindowStore) Count(ctx context.Context, key string, from, to int64) (int64, pkgErrors.AppError) {
	count, err := r.client.ZCount(ctx, key, fmt.Sprint(from), fmt.Sprint(to)).Result()
	if err != nil {
		r.log.Error("windowstore.count failed",
			logger.String("service", "windowstore"),
			logger.String("key", key),
			logger.Error(err),
		)
		return 0, pkgErrors.FromError(err, ErrWindowStoreCount, "count failed")
	}
	return count, nil
}

func (r *RedisWindowStore) TrimBefore(ctx context.Context, key string, before int64) pkgErrors.AppError {
	err := r.client.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprint(before)).Err()
	if err != nil {
		r.log.Error("windowstore.trim failed",
			logger.String("service", "windowstore"),
			logger.String("key", key),
			logger.Int64("before", before),
			logger.Error(err),
		)
		return pkgErrors.FromError(err, ErrWindowStoreTrim, "trim failed")
	}
	return nil
}

func (r *RedisWindowStore) Size(ctx context.Context, key string) (int64, pkgErrors.AppError) {
	size, err := r.client.ZCard(ctx, key).Result()
	if err != nil {
		r.log.Error("windowstore.size failed",
			logger.String("service", "windowstore"),
			logger.String("key", key),
			logger.Error(err),
		)
		return 0, pkgErrors.FromError(err, ErrWindowStoreSize, "size failed")
	}
	return size, nil
}

func (r *RedisWindowStore) Clear(ctx context.Context, key string) pkgErrors.AppError {
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		r.log.Error("windowstore.clear failed",
			logger.String("service", "windowstore"),
			logger.String("key", key),
			logger.Error(err),
		)
		return pkgErrors.FromError(err, ErrWindowStoreClear, "clear failed")
	}
	return nil
}

func (r *RedisWindowStore) Close() pkgErrors.AppError {
	err := r.client.Close()
	if err != nil {
		return pkgErrors.FromError(err, ErrWindowStoreClose, "close failed")
	}
	return nil
}
