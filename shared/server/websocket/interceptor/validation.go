package interceptor

import (
	"context"
	"fmt"
)

// ValidationInterceptor validates messages
type ValidationInterceptor struct {
	maxSize int64
}

// NewValidationInterceptor creates a new validation interceptor
func NewValidationInterceptor(maxSize int64) *ValidationInterceptor {
	return &ValidationInterceptor{
		maxSize: maxSize,
	}
}

// Intercept validates the message
func (i *ValidationInterceptor) Intercept(ctx context.Context, msg *Message, next func(context.Context, *Message) error) error {
	// Validate message size
	if int64(len(msg.Data)) > i.maxSize {
		return fmt.Errorf("message size %d exceeds maximum %d", len(msg.Data), i.maxSize)
	}

	// Validate message is not empty
	if len(msg.Data) == 0 {
		return fmt.Errorf("message is empty")
	}

	return next(ctx, msg)
}
