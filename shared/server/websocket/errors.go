package websocket

import (
	"errors"
	"fmt"

	"github.com/gorilla/websocket"
)

// Common WebSocket errors
var (
	// Connection errors
	ErrConnectionClosed       = errors.New("websocket: connection closed")
	ErrConnectionTimeout      = errors.New("websocket: connection timeout")
	ErrConnectionRefused      = errors.New("websocket: connection refused")
	ErrConnectionLimitExceeded = errors.New("websocket: connection limit exceeded")
	ErrInvalidConnection      = errors.New("websocket: invalid connection")

	// Client errors
	ErrClientNotFound         = errors.New("websocket: client not found")
	ErrClientAlreadyConnected = errors.New("websocket: client already connected")
	ErrClientDisconnected     = errors.New("websocket: client disconnected")
	ErrClientBusy             = errors.New("websocket: client busy")

	// Message errors
	ErrMessageTooLarge        = errors.New("websocket: message too large")
	ErrInvalidMessageType     = errors.New("websocket: invalid message type")
	ErrInvalidMessage         = errors.New("websocket: invalid message")
	ErrMessageQueueFull       = errors.New("websocket: message queue full")
	ErrMessageTimeout         = errors.New("websocket: message send timeout")

	// Hub errors
	ErrHubClosed              = errors.New("websocket: hub closed")
	ErrHubNotRunning          = errors.New("websocket: hub not running")

	// Configuration errors
	ErrInvalidConfiguration   = errors.New("websocket: invalid configuration")

	// Rate limiting errors
	ErrRateLimitExceeded      = errors.New("websocket: rate limit exceeded")

	// Upgrade errors
	ErrUpgradeFailed          = errors.New("websocket: upgrade failed")
	ErrInvalidUpgrade         = errors.New("websocket: invalid upgrade request")
	ErrOriginNotAllowed       = errors.New("websocket: origin not allowed")
)

// Error types for better error handling

// ConnectionError represents a connection-related error
type ConnectionError struct {
	ClientID string
	Err      error
	Message  string
}

func (e *ConnectionError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("connection error for client %s: %s: %v", e.ClientID, e.Message, e.Err)
	}
	return fmt.Sprintf("connection error for client %s: %v", e.ClientID, e.Err)
}

func (e *ConnectionError) Unwrap() error {
	return e.Err
}

// NewConnectionError creates a new connection error
func NewConnectionError(clientID string, err error, message string) *ConnectionError {
	return &ConnectionError{
		ClientID: clientID,
		Err:      err,
		Message:  message,
	}
}

// MessageError represents a message-related error
type MessageError struct {
	ClientID    string
	MessageType string
	Err         error
	Message     string
}

func (e *MessageError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("message error for client %s (type: %s): %s: %v",
			e.ClientID, e.MessageType, e.Message, e.Err)
	}
	return fmt.Sprintf("message error for client %s (type: %s): %v",
		e.ClientID, e.MessageType, e.Err)
}

func (e *MessageError) Unwrap() error {
	return e.Err
}

// NewMessageError creates a new message error
func NewMessageError(clientID, messageType string, err error, message string) *MessageError {
	return &MessageError{
		ClientID:    clientID,
		MessageType: messageType,
		Err:         err,
		Message:     message,
	}
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for field %s: %s (value: %v)",
		e.Field, e.Message, e.Value)
}

// NewValidationError creates a new validation error
func NewValidationError(field string, value interface{}, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// ConfigError represents a configuration error
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("configuration error for field %s: %s", e.Field, e.Message)
}

// ErrInvalidConfig creates a new config error
func ErrInvalidConfig(message string) error {
	return &ConfigError{
		Message: message,
	}
}

// IsConnectionError checks if error is a connection error
func IsConnectionError(err error) bool {
	var connErr *ConnectionError
	return errors.As(err, &connErr)
}

// IsMessageError checks if error is a message error
func IsMessageError(err error) bool {
	var msgErr *MessageError
	return errors.As(err, &msgErr)
}

// IsValidationError checks if error is a validation error
func IsValidationError(err error) bool {
	var valErr *ValidationError
	return errors.As(err, &valErr)
}

// IsConfigError checks if error is a config error
func IsConfigError(err error) bool {
	var cfgErr *ConfigError
	return errors.As(err, &cfgErr)
}

// IsClosed checks if the error is a close error
func IsClosed(err error) bool {
	return errors.Is(err, ErrConnectionClosed) ||
		   errors.Is(err, ErrHubClosed) ||
		   websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway)
}

// IsTimeout checks if the error is a timeout error
func IsTimeout(err error) bool {
	return errors.Is(err, ErrConnectionTimeout) ||
		   errors.Is(err, ErrMessageTimeout)
}

// IsRateLimit checks if the error is a rate limit error
func IsRateLimit(err error) bool {
	return errors.Is(err, ErrRateLimitExceeded)
}
