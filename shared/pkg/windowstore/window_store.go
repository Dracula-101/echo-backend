package windowstore

import (
	"context"
	pkgErrors "shared/pkg/errors"
	"time"
)

type WindowStore interface {
	Add(ctx context.Context, key string, ts int64, ttl time.Duration) pkgErrors.AppError
	AddMulti(ctx context.Context, key string, timestamps []int64, ttl time.Duration) pkgErrors.AppError

	Count(ctx context.Context, key string, from, to int64) (int64, pkgErrors.AppError)
	TrimBefore(ctx context.Context, key string, before int64) pkgErrors.AppError

	Size(ctx context.Context, key string) (int64, pkgErrors.AppError)
	Clear(ctx context.Context, key string) pkgErrors.AppError

	Close() pkgErrors.AppError
}
