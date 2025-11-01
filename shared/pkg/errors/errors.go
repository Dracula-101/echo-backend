package errors

import (
	"fmt"
	"runtime"
)

type Error interface {
	error
	Code() string
	Message() string
	Details() map[string]interface{}
	StackTrace() []string
	Unwrap() error
}

type appError struct {
	code       string
	message    string
	details    map[string]interface{}
	stackTrace []string
	wrapped    error
}

func New(code, message string) Error {
	return &appError{
		code:       code,
		message:    message,
		details:    make(map[string]interface{}),
		stackTrace: captureStackTrace(),
	}
}

func Wrap(err error, code, message string) Error {
	if err == nil {
		return nil
	}

	return &appError{
		code:       code,
		message:    message,
		details:    make(map[string]interface{}),
		stackTrace: captureStackTrace(),
		wrapped:    err,
	}
}

func (e *appError) Error() string {
	if e.wrapped != nil {
		return fmt.Sprintf("%s: %s: %v", e.code, e.message, e.wrapped)
	}
	return fmt.Sprintf("%s: %s", e.code, e.message)
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

func (e *appError) WithDetail(key string, value interface{}) Error {
	e.details[key] = value
	return e
}

func captureStackTrace() []string {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(3, pcs[:])

	frames := runtime.CallersFrames(pcs[:n])
	stackTrace := make([]string, 0, n)

	for {
		frame, more := frames.Next()
		stackTrace = append(stackTrace, fmt.Sprintf("%s:%d %s", frame.File, frame.Line, frame.Function))
		if !more {
			break
		}
	}

	return stackTrace
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
