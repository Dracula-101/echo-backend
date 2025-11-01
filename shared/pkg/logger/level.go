package logger

import (
	"os"
)

type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

func GetLoggerLevel() Level {
	levelStr := os.Getenv("LOG_LEVEL")
	return ParseLevel(levelStr)
}

func GetLoggerFormat() Format {
	formatStr := os.Getenv("LOG_FORMAT")
	return ParseFormat(formatStr)
}

func GetLoggerOutput() *os.File {
	output := os.Getenv("LOG_OUTPUT")
	if output == "stderr" {
		return os.Stderr
	}
	// Default to stdout
	return os.Stdout
}

func GetLoggerTimeFormat() string {
	timeFormat := os.Getenv("LOG_TIME_FORMAT")
	if timeFormat == "" {
		return "2006-01-02T15:04:05Z07:00"
	}
	return timeFormat
}

func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	case FatalLevel:
		return "fatal"
	default:
		return "unknown"
	}
}

func ParseLevel(s string) Level {
	switch s {
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn", "warning":
		return WarnLevel
	case "error":
		return ErrorLevel
	case "fatal":
		return FatalLevel
	default:
		return InfoLevel
	}
}

func ParseFormat(s string) Format {
	switch s {
	case "json":
		return FormatJSON
	case "text":
		return FormatText
	default:
		return FormatText
	}
}
