package utils

import (
	"context"
	"time"
)

func SleepWithContext(ctx context.Context, duration time.Duration) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(duration):
		return
	}
}
