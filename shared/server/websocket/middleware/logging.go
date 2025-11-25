package middleware

import (
	"context"
	"time"

	"shared/pkg/logger"
)

// Logging creates a logging middleware
func Logging(log logger.Logger) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, data []byte) error {
			start := time.Now()

			log.Debug("Processing message",
				logger.Int("size", len(data)),
			)

			err := next(ctx, data)

			duration := time.Since(start)

			if err != nil {
				log.Error("Message processing failed",
					logger.Duration("duration", duration),
					logger.Error(err),
				)
			} else {
				log.Debug("Message processed successfully",
					logger.Duration("duration", duration),
				)
			}

			return err
		}
	}
}
