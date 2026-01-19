package domain

import "fmt"

const (
	UserIdMetaKey       = "user_id"
	DeviceIdMetaKey     = "device_id"
	ConnectionIdMetaKey = "connection_id"
)

const (
	ErrCodeInvalidRequest   = "INVALID_REQUEST"
	ErrCodeUserNotFound     = "USER_NOT_FOUND"
	ErrCodeUserNotOnline    = "USER_NOT_ONLINE"
	ErrCodeInvalidEventType = "INVALID_EVENT_TYPE"
	ErrCodeBroadcastFailed  = "BROADCAST_FAILED"
	ErrCodeConnectionFailed = "CONNECTION_FAILED"
	ErrCodeInvalidUserID    = "INVALID_USER_ID"
	ErrCodeInvalidDeviceID  = "INVALID_DEVICE_ID"
	ErrCodeDatabaseError    = "DATABASE_ERROR"
	ErrCodeCacheError       = "CACHE_ERROR"
)

type ServiceError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

func (e *ServiceError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func NewError(code, message string) *ServiceError {
	return &ServiceError{
		Code:    code,
		Message: message,
	}
}

func NewErrorWithDetails(code, message string, details interface{}) *ServiceError {
	return &ServiceError{
		Code:    code,
		Message: message,
		Details: details,
	}
}
