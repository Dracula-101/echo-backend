package errors

import (
	"fmt"
	"runtime"
	"strings"
)

type AppError interface {
	error
	Code() string
	Message() string
	Details() map[string]interface{}
	StackTrace() []string
	Unwrap() error
	Service() string
	CorrelationID() string
	WithService(service string) AppError
	WithCorrelationID(correlationID string) AppError
	WithDetail(key string, value interface{}) AppError
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

func New(code, message string) AppError {
	return &appError{
		code:       code,
		message:    message,
		details:    make(map[string]interface{}),
		stackTrace: captureCleanStack(3),
	}
}

func FromError(err error, code, message string) AppError {
	if err == nil {
		return nil
	}

	if appErr, ok := err.(AppError); ok {
		wrapLocation := captureCleanStack(3)

		combinedStack := make([]string, 0, len(appErr.StackTrace())+len(wrapLocation)+1)
		combinedStack = append(combinedStack, appErr.StackTrace()...)

		return &appError{
			code:          code,
			message:       message,
			stackTrace:    combinedStack,
			wrapped:       err,
			details:       appErr.Details(),
			service:       appErr.Service(),
			correlationID: appErr.CorrelationID(),
		}
	}

	return &appError{
		code:       code,
		message:    message,
		details:    make(map[string]interface{}),
		stackTrace: captureCleanStack(3),
		wrapped:    err,
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

func (e *appError) WithService(service string) AppError {
	e.service = service
	return e
}

func (e *appError) WithCorrelationID(correlationID string) AppError {
	e.correlationID = correlationID
	return e
}

func (e *appError) WithDetail(key string, value interface{}) AppError {
	e.details[key] = value
	return e
}

func captureCleanStack(skip int) []string {
	const maxDepth = 50
	pcs := make([]uintptr, maxDepth)
	n := runtime.Callers(skip, pcs)

	if n == 0 {
		return nil
	}

	frames := runtime.CallersFrames(pcs[:n])
	stack := make([]string, 0, n)

	for {
		frame, more := frames.Next()

		if shouldIncludeFrame(frame) {
			stack = append(stack, formatFrame(frame)...)
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

	return true
}

func formatFrame(frame runtime.Frame) []string {
	return []string{
		frame.Function,
		fmt.Sprintf("%s:%d", frame.File, frame.Line),
		string(""),
	}
}

func GetCode(err error) string {
	if err == nil {
		return ""
	}

	if e, ok := err.(AppError); ok {
		return e.Code()
	}

	return CodeInternal
}

func GetDetails(err error) map[string]interface{} {
	if err == nil {
		return nil
	}

	if e, ok := err.(AppError); ok {
		return e.Details()
	}

	return nil
}
