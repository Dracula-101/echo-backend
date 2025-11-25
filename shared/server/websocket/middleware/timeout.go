package middleware

import (
	"context"
	"time"
)

// Timeout creates a timeout middleware
func Timeout(timeout time.Duration) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, data []byte) error {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			done := make(chan error, 1)

			go func() {
				done <- next(ctx, data)
			}()

			select {
			case err := <-done:
				return err
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}
