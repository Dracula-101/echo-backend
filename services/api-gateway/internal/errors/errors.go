package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// ErrorCode represents application-specific error codes
type ErrorCode string

const (
	// Client errors (4xx)
	ErrCodeBadRequest        ErrorCode = "BAD_REQUEST"
	ErrCodeUnauthorized      ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden         ErrorCode = "FORBIDDEN"
	ErrCodeNotFound          ErrorCode = "NOT_FOUND"
	ErrCodeMethodNotAllowed  ErrorCode = "METHOD_NOT_ALLOWED"
	ErrCodeConflict          ErrorCode = "CONFLICT"
	ErrCodeValidation        ErrorCode = "VALIDATION_ERROR"
	ErrCodeRateLimitExceeded ErrorCode = "RATE_LIMIT_EXCEEDED"
	ErrCodeRequestTimeout    ErrorCode = "REQUEST_TIMEOUT"
	ErrCodePayloadTooLarge   ErrorCode = "PAYLOAD_TOO_LARGE"
	ErrCodeUnsupportedMedia  ErrorCode = "UNSUPPORTED_MEDIA_TYPE"

	// Server errors (5xx)
	ErrCodeInternal           ErrorCode = "INTERNAL_ERROR"
	ErrCodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	ErrCodeGatewayTimeout     ErrorCode = "GATEWAY_TIMEOUT"
	ErrCodeDependencyFailure  ErrorCode = "DEPENDENCY_FAILURE"
)

// APIError represents a structured API error
type APIError struct {
	Code       ErrorCode              `json:"code"`
	Message    string                 `json:"message"`
	StatusCode int                    `json:"-"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Err        error                  `json:"-"`
}

func (e *APIError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *APIError) Unwrap() error {
	return e.Err
}

// WithDetails adds details to the error
func (e *APIError) WithDetails(key string, value interface{}) *APIError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// WithError wraps an underlying error
func (e *APIError) WithError(err error) *APIError {
	e.Err = err
	return e
}

// New creates a new APIError
func New(code ErrorCode, message string, statusCode int) *APIError {
	return &APIError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
	}
}

// Client error constructors

// BadRequest creates a bad request error
func BadRequest(message string) *APIError {
	return New(ErrCodeBadRequest, message, http.StatusBadRequest)
}

// Unauthorized creates an unauthorized error
func Unauthorized(message string) *APIError {
	return New(ErrCodeUnauthorized, message, http.StatusUnauthorized)
}

// Forbidden creates a forbidden error
func Forbidden(message string) *APIError {
	return New(ErrCodeForbidden, message, http.StatusForbidden)
}

// NotFound creates a not found error
func NotFound(message string) *APIError {
	return New(ErrCodeNotFound, message, http.StatusNotFound)
}

// MethodNotAllowed creates a method not allowed error
func MethodNotAllowed(message string) *APIError {
	return New(ErrCodeMethodNotAllowed, message, http.StatusMethodNotAllowed)
}

// Conflict creates a conflict error
func Conflict(message string) *APIError {
	return New(ErrCodeConflict, message, http.StatusConflict)
}

// ValidationError creates a validation error
func ValidationError(message string) *APIError {
	return New(ErrCodeValidation, message, http.StatusUnprocessableEntity)
}

// RateLimitExceeded creates a rate limit exceeded error
func RateLimitExceeded(message string, retryAfter int) *APIError {
	return New(ErrCodeRateLimitExceeded, message, http.StatusTooManyRequests).
		WithDetails("retry_after", retryAfter)
}

// RequestTimeout creates a request timeout error
func RequestTimeout(message string) *APIError {
	return New(ErrCodeRequestTimeout, message, http.StatusRequestTimeout)
}

// PayloadTooLarge creates a payload too large error
func PayloadTooLarge(message string, maxSize int64) *APIError {
	return New(ErrCodePayloadTooLarge, message, http.StatusRequestEntityTooLarge).
		WithDetails("max_size_bytes", maxSize)
}

// UnsupportedMediaType creates an unsupported media type error
func UnsupportedMediaType(message string, supported []string) *APIError {
	return New(ErrCodeUnsupportedMedia, message, http.StatusUnsupportedMediaType).
		WithDetails("supported_types", supported)
}

// Server error constructors

// Internal creates an internal server error
func Internal(message string) *APIError {
	return New(ErrCodeInternal, message, http.StatusInternalServerError)
}

// ServiceUnavailable creates a service unavailable error
func ServiceUnavailable(message string, retryAfter int) *APIError {
	return New(ErrCodeServiceUnavailable, message, http.StatusServiceUnavailable).
		WithDetails("retry_after", retryAfter)
}

// GatewayTimeout creates a gateway timeout error
func GatewayTimeout(message string, service string) *APIError {
	return New(ErrCodeGatewayTimeout, message, http.StatusGatewayTimeout).
		WithDetails("service", service)
}

// DependencyFailure creates a dependency failure error
func DependencyFailure(service string, err error) *APIError {
	return New(ErrCodeDependencyFailure,
		fmt.Sprintf("downstream service %s failed", service),
		http.StatusBadGateway).
		WithError(err).
		WithDetails("service", service)
}

// Error type checking helpers

// IsAPIError checks if an error is an APIError
func IsAPIError(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr)
}

// AsAPIError converts an error to APIError if possible
func AsAPIError(err error) (*APIError, bool) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr, true
	}
	return nil, false
}

// GetStatusCode extracts status code from error, defaults to 500
func GetStatusCode(err error) int {
	if apiErr, ok := AsAPIError(err); ok {
		return apiErr.StatusCode
	}
	return http.StatusInternalServerError
}

// GetErrorCode extracts error code from error
func GetErrorCode(err error) ErrorCode {
	if apiErr, ok := AsAPIError(err); ok {
		return apiErr.Code
	}
	return ErrCodeInternal
}

// ValidationErrors represents multiple validation errors
type ValidationErrors struct {
	Errors []ValidationFieldError `json:"errors"`
}

func (v *ValidationErrors) Error() string {
	return fmt.Sprintf("validation failed: %d errors", len(v.Errors))
}

// ValidationFieldError represents a single field validation error
type ValidationFieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Tag     string `json:"tag,omitempty"`
	Value   string `json:"value,omitempty"`
}

// NewValidationErrors creates a new validation errors collection
func NewValidationErrors() *ValidationErrors {
	return &ValidationErrors{
		Errors: make([]ValidationFieldError, 0),
	}
}

// Add adds a field error
func (v *ValidationErrors) Add(field, message, tag string) {
	v.Errors = append(v.Errors, ValidationFieldError{
		Field:   field,
		Message: message,
		Tag:     tag,
	})
}

// AddWithValue adds a field error with the invalid value
func (v *ValidationErrors) AddWithValue(field, message, tag, value string) {
	v.Errors = append(v.Errors, ValidationFieldError{
		Field:   field,
		Message: message,
		Tag:     tag,
		Value:   value,
	})
}

// HasErrors returns true if there are validation errors
func (v *ValidationErrors) HasErrors() bool {
	return len(v.Errors) > 0
}

// ToAPIError converts validation errors to APIError
func (v *ValidationErrors) ToAPIError() *APIError {
	return ValidationError("validation failed").
		WithDetails("fields", v.Errors)
}

// Error categories for monitoring and alerting

// ErrorCategory represents the category of an error
type ErrorCategory string

const (
	CategoryClient     ErrorCategory = "client"
	CategoryServer     ErrorCategory = "server"
	CategoryDependency ErrorCategory = "dependency"
	CategoryTimeout    ErrorCategory = "timeout"
	CategoryRateLimit  ErrorCategory = "rate_limit"
	CategoryValidation ErrorCategory = "validation"
)

// GetCategory returns the error category
func GetCategory(err error) ErrorCategory {
	apiErr, ok := AsAPIError(err)
	if !ok {
		return CategoryServer
	}

	switch apiErr.Code {
	case ErrCodeBadRequest, ErrCodeUnauthorized, ErrCodeForbidden,
		ErrCodeNotFound, ErrCodeMethodNotAllowed, ErrCodeConflict,
		ErrCodeUnsupportedMedia, ErrCodePayloadTooLarge:
		return CategoryClient

	case ErrCodeValidation:
		return CategoryValidation

	case ErrCodeRateLimitExceeded:
		return CategoryRateLimit

	case ErrCodeRequestTimeout, ErrCodeGatewayTimeout:
		return CategoryTimeout

	case ErrCodeDependencyFailure:
		return CategoryDependency

	default:
		return CategoryServer
	}
}

// IsRetriable returns true if the error is potentially retriable
func IsRetriable(err error) bool {
	apiErr, ok := AsAPIError(err)
	if !ok {
		return false
	}

	switch apiErr.Code {
	case ErrCodeServiceUnavailable, ErrCodeGatewayTimeout,
		ErrCodeRequestTimeout:
		return true
	default:
		return false
	}
}

// IsClientError returns true if the error is a client error (4xx)
func IsClientError(err error) bool {
	statusCode := GetStatusCode(err)
	return statusCode >= 400 && statusCode < 500
}

// IsServerError returns true if the error is a server error (5xx)
func IsServerError(err error) bool {
	statusCode := GetStatusCode(err)
	return statusCode >= 500
}
