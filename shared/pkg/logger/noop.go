package logger

import (
	"context"
	"time"
)

type noopLogger struct{}

func NewNoop() Logger {
	return &noopLogger{}
}

func (l *noopLogger) Debug(msg string, fields ...Field) {}
func (l *noopLogger) Info(msg string, fields ...Field)  {}
func (l *noopLogger) Warn(msg string, fields ...Field)  {}
func (l *noopLogger) Error(msg string, fields ...Field) {}
func (l *noopLogger) Fatal(msg string, fields ...Field) {}
func (l *noopLogger) Request(tx context.Context, method string, routePath string, statusCode int, duration time.Duration, bodySize int64, msg string, fields ...Field) {
}

func (l *noopLogger) With(fields ...Field) Logger {
	return l
}

func (l *noopLogger) WithContext(ctx context.Context) Logger {
	return l
}

func (l *noopLogger) Sync() error {
	return nil
}
