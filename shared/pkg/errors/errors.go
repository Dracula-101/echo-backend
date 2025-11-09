package errors

import (
	"fmt"
	"runtime"
	"strings"
)

// Error is our custom error interface with rich context
type Error interface {
	error
	Code() string
	Message() string
	Details() map[string]interface{}
	StackTrace() []string
	Unwrap() error
	Service() string
	CorrelationID() string
	WithService(service string) Error
	WithCorrelationID(correlationID string) Error
	WithDetail(key string, value interface{}) Error
}

type appError struct {
	code          string
	message       string
	details       map[string]interface{}
	stackTrace    []string
	wrapped       error
	service       string
	correlationID string
}

// New creates a new error with automatic stack trace capture
func New(code, message string) Error {
	return &appError{
		code:       code,
		message:    message,
		details:    make(map[string]interface{}),
		stackTrace: captureCleanStack(),
	}
}

func (e *appError) Error() string {
	var prefix string
	if e.service != "" {
		prefix = fmt.Sprintf("[%s]", e.service)
	}
	if e.correlationID != "" {
		if prefix != "" {
			prefix = fmt.Sprintf("%s [%s]", prefix, e.correlationID)
		} else {
			prefix = fmt.Sprintf("[%s]", e.correlationID)
		}
	}

	if prefix != "" {
		prefix = prefix + " "
	}

	if e.wrapped != nil {
		return fmt.Sprintf("%s%s: %s: %v", prefix, e.code, e.message, e.wrapped)
	}
	return fmt.Sprintf("%s%s: %s", prefix, e.code, e.message)
}

func (e *appError) Code() string {
	return e.code
}

func (e *appError) Message() string {
	return e.message
}

func (e *appError) Details() map[string]interface{} {
	return e.details
}

func (e *appError) StackTrace() []string {
	return e.stackTrace
}

func (e *appError) Unwrap() error {
	return e.wrapped
}

func (e *appError) Service() string {
	return e.service
}

func (e *appError) CorrelationID() string {
	return e.correlationID
}

func (e *appError) WithService(service string) Error {
	e.service = service
	return e
}

func (e *appError) WithCorrelationID(correlationID string) Error {
	e.correlationID = correlationID
	return e
}

func (e *appError) WithDetail(key string, value interface{}) Error {
	e.details[key] = value
	return e
}

func captureCleanStack() []string {
	const maxDepth = 32
	pcs := make([]uintptr, maxDepth)
	n := runtime.Callers(3, pcs)

	if n == 0 {
		return nil
	}

	frames := runtime.CallersFrames(pcs[:n])
	stack := make([]string, 0, n)

	for {
		frame, more := frames.Next()

		if shouldIncludeFrame(frame) {
			stack = append(stack, formatFrame(frame))
		}

		if !more {
			break
		}
	}

	return stack
}

func shouldIncludeFrame(frame runtime.Frame) bool {
	file := frame.File
	fn := frame.Function

	if file == "" || fn == "" {
		return false
	}

	if strings.HasPrefix(fn, "runtime.") {
		return false
	}

	if strings.Contains(fn, "testing.") {
		return false
	}

	if !strings.Contains(file, "/") ||
		(!strings.Contains(file, ".") && !strings.Contains(file, "vendor")) {
		if !strings.Contains(file, "go/src") {
			return false
		}
	}

	if strings.Contains(fn, "/errors.") || strings.HasSuffix(fn, "/errors.New") {
		return false
	}

	return true
}

func formatFrame(frame runtime.Frame) string {
	file := frame.File
	if idx := strings.LastIndex(file, "/"); idx >= 0 {
		if idx2 := strings.LastIndex(file[:idx], "/"); idx2 >= 0 {
			file = file[idx2+1:]
		} else {
			file = file[idx+1:]
		}
	}

	fn := frame.Function
	if idx := strings.LastIndex(fn, "/"); idx >= 0 {
		fn = fn[idx+1:]
	}

	return fmt.Sprintf("%s:%d %s", file, frame.Line, fn)
}

func GetCode(err error) string {
	if err == nil {
		return ""
	}

	if e, ok := err.(Error); ok {
		return e.Code()
	}

	return CodeInternal
}

func GetDetails(err error) map[string]interface{} {
	if err == nil {
		return nil
	}

	if e, ok := err.(Error); ok {
		return e.Details()
	}

	return nil
}
