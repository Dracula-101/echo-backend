package logger

import (
	"context"
)

type contextKey struct{}

var loggerKey = contextKey{}

func FromContext(ctx context.Context) Logger {
	if logger, ok := ctx.Value(loggerKey).(Logger); ok {
		return logger
	}
	return NewNoop()
}

func WithContext(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}
