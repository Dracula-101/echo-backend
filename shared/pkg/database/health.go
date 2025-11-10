package database

import (
	"context"
	"time"
)

type HealthChecker interface {
	Check(ctx context.Context) error
}

type healthChecker struct {
	db      Database
	timeout time.Duration
}

func NewHealthChecker(db Database, timeout time.Duration) HealthChecker {
	return &healthChecker{
		db:      db,
		timeout: timeout,
	}
}

func (h *healthChecker) Check(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()

	return h.db.Ping(ctx).Unwrap()
}
