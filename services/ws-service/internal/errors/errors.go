package errors

import "fmt"

// Error codes for ws-service
const (
	ErrCodeInvalidRequest      = "INVALID_REQUEST"
	ErrCodeUserNotFound        = "USER_NOT_FOUND"
	ErrCodeUserNotOnline       = "USER_NOT_ONLINE"
	ErrCodeInvalidEventType    = "INVALID_EVENT_TYPE"
	ErrCodeBroadcastFailed     = "BROADCAST_FAILED"
	ErrCodeConnectionFailed    = "CONNECTION_FAILED"
	ErrCodeInvalidUserID       = "INVALID_USER_ID"
	ErrCodeInvalidDeviceID     = "INVALID_DEVICE_ID"
	ErrCodeDatabaseError       = "DATABASE_ERROR"
	ErrCodeCacheError          = "CACHE_ERROR"
)

// ServiceError represents a ws-service specific error
type ServiceError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

func (e *ServiceError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// New creates a new ServiceError
func New(code, message string) *ServiceError {
	return &ServiceError{
		Code:    code,
		Message: message,
	}
}

// NewWithDetails creates a new ServiceError with details
func NewWithDetails(code, message string, details interface{}) *ServiceError {
	return &ServiceError{
		Code:    code,
		Message: message,
		Details: details,
	}
}
