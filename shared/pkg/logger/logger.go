package logger

import (
	"context"
	"io"
	"time"
)

type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)

	Request(tx context.Context, method string, routePath string, statusCode int, duration time.Duration, bodySize int64, msg string, fields ...Field)

	With(fields ...Field) Logger
	WithContext(ctx context.Context) Logger

	Sync() error
}

type Field interface {
	Key() string
	Value() interface{}
}

type field struct {
	key   string
	value interface{}
}

func (f *field) Key() string {
	return f.key
}

func (f *field) Value() interface{} {
	return f.value
}

func String(key, val string) Field {
	return &field{key: key, value: val}
}

func Int(key string, val int) Field {
	return &field{key: key, value: val}
}

func Int64(key string, val int64) Field {
	return &field{key: key, value: val}
}

func Float64(key string, val float64) Field {
	return &field{key: key, value: val}
}

func Bool(key string, val bool) Field {
	return &field{key: key, value: val}
}

func Any(key string, val interface{}) Field {
	return &field{key: key, value: val}
}

func Duration(key string, val time.Duration) Field {
	return &field{key: key, value: val}
}

func Error(err error) Field {
	return &field{key: "error", value: err.Error()}
}

type Config struct {
	Level      Level
	Output     io.Writer
	Format     Format
	TimeFormat string
	Service    string
}

type Format string

const (
	FormatJSON Format = "json"
	FormatText Format = "text"
)
