package middleware

import (
	"context"
	"fmt"
	"runtime/debug"

	"shared/pkg/logger"
)

// Recovery creates a recovery middleware that catches panics
func Recovery(log logger.Logger) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, data []byte) (err error) {
			defer func() {
				if r := recover(); r != nil {
					log.Error("Panic recovered in message handler",
						logger.Any("panic", r),
						logger.String("stack", string(debug.Stack())),
					)
					err = fmt.Errorf("panic recovered: %v", r)
				}
			}()

			return next(ctx, data)
		}
	}
}
