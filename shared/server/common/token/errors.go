package token

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidConfig = errors.New("token: invalid manager config")
	ErrInvalidToken  = errors.New("token: invalid token")
	ErrExpiredToken  = errors.New("token: token expired")
	ErrKeyNotFound   = errors.New("token: signing key not found")
)

// ErrorCode represents specific token error types
type ErrorCode string

const (
	ErrCodeExpired       ErrorCode = "EXPIRED"
	ErrCodeInvalid       ErrorCode = "INVALID"
	ErrCodeMalformed     ErrorCode = "MALFORMED"
	ErrCodeSignature     ErrorCode = "SIGNATURE_INVALID"
	ErrCodeMissingKey    ErrorCode = "KEY_NOT_FOUND"
	ErrCodeWrongType     ErrorCode = "WRONG_TYPE"
	ErrCodeWrongAudience ErrorCode = "WRONG_AUDIENCE"
	ErrCodeNotYetValid   ErrorCode = "NOT_YET_VALID"
	ErrCodeRevoked       ErrorCode = "REVOKED"
	ErrCodeConfigInvalid ErrorCode = "CONFIG_INVALID"
)

// Error represents a structured token error with additional context
type Error struct {
	Code    ErrorCode
	Message string
	Details map[string]any
	Err     error
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("token error [%s]: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("token error [%s]: %s", e.Code, e.Message)
}

// Unwrap implements the errors.Unwrap interface
func (e *Error) Unwrap() error {
	return e.Err
}

// Is implements the errors.Is interface
func (e *Error) Is(target error) bool {
	if t, ok := target.(*Error); ok {
		return e.Code == t.Code
	}
	return errors.Is(e.Err, target)
}

// WithDetail adds a detail to the error
func (e *Error) WithDetail(key string, value any) *Error {
	if e.Details == nil {
		e.Details = make(map[string]any)
	}
	e.Details[key] = value
	return e
}

// NewError creates a new Error with the given code and message
func NewError(code ErrorCode, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// WrapError wraps an existing error with a token Error
func WrapError(code ErrorCode, message string, err error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// IsErrorCode checks if an error is a token error with a specific code
func IsErrorCode(err error, code ErrorCode) bool {
	var tokenErr *Error
	if errors.As(err, &tokenErr) {
		return tokenErr.Code == code
	}
	return false
}
