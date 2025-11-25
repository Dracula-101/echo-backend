package handler

import (
	"encoding/json"
	"time"
)

// ErrorCode represents WebSocket error codes
type ErrorCode string

const (
	// Client errors (4xxx)
	ErrCodeInvalidJSON        ErrorCode = "INVALID_JSON"
	ErrCodeInvalidMessageType ErrorCode = "INVALID_MESSAGE_TYPE"
	ErrCodeMissingField       ErrorCode = "MISSING_FIELD"
	ErrCodeInvalidField       ErrorCode = "INVALID_FIELD"
	ErrCodeMessageTooLarge    ErrorCode = "MESSAGE_TOO_LARGE"
	ErrCodeRateLimitExceeded  ErrorCode = "RATE_LIMIT_EXCEEDED"
	ErrCodeUnauthorized       ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden          ErrorCode = "FORBIDDEN"

	// Server errors (5xxx)
	ErrCodeInternalError  ErrorCode = "INTERNAL_ERROR"
	ErrCodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	ErrCodeTimeout        ErrorCode = "TIMEOUT"
)

// ErrorResponse represents a WebSocket error response
type ErrorResponse struct {
	Type      string    `json:"type"`
	Code      ErrorCode `json:"code"`
	Message   string    `json:"message"`
	Details   any       `json:"details,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// NewErrorResponse creates a new error response
func NewErrorResponse(code ErrorCode, message string, details any) *ErrorResponse {
	return &ErrorResponse{
		Type:      "error",
		Code:      code,
		Message:   message,
		Details:   details,
		Timestamp: time.Now(),
	}
}

// Error implements the error interface
func (e *ErrorResponse) Error() string {
	return string(e.Code) + ": " + e.Message
}

// ToJSON converts error response to JSON bytes
func (e *ErrorResponse) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// Common error messages
var (
	ErrInvalidJSON = NewErrorResponse(
		ErrCodeInvalidJSON,
		"Invalid JSON format",
		nil,
	)

	ErrInvalidMessageType = NewErrorResponse(
		ErrCodeInvalidMessageType,
		"Invalid or missing message type",
		nil,
	)

	ErrMessageTooLarge = NewErrorResponse(
		ErrCodeMessageTooLarge,
		"Message size exceeds maximum allowed",
		nil,
	)

	ErrUnauthorized = NewErrorResponse(
		ErrCodeUnauthorized,
		"Unauthorized access",
		nil,
	)

	ErrInternalError = NewErrorResponse(
		ErrCodeInternalError,
		"Internal server error",
		nil,
	)
)

// ErrorValidator validates messages and returns errors
type ErrorValidator interface {
	Validate(message []byte) error
}

// MessageTypeValidator validates message type field
type MessageTypeValidator struct {
	RequiredField string
	AllowedTypes  []string
}

// Validate checks if message has valid type
func (v *MessageTypeValidator) Validate(message []byte) error {
	if len(message) == 0 {
		return NewErrorResponse(ErrCodeInvalidJSON, "Empty message", nil)
	}

	var data map[string]any
	if err := json.Unmarshal(message, &data); err != nil {
		return NewErrorResponse(ErrCodeInvalidJSON, "Invalid JSON format: "+err.Error(), nil)
	}

	// Check if type field exists
	typeField, ok := data[v.RequiredField]
	if !ok {
		return NewErrorResponse(
			ErrCodeMissingField,
			"Missing required field: "+v.RequiredField,
			map[string]any{"field": v.RequiredField},
		)
	}

	// Check if type is string
	typeStr, ok := typeField.(string)
	if !ok {
		return NewErrorResponse(
			ErrCodeInvalidField,
			"Field must be a string: "+v.RequiredField,
			map[string]any{"field": v.RequiredField},
		)
	}

	// If allowed types specified, validate against them
	if len(v.AllowedTypes) > 0 {
		valid := false
		for _, allowed := range v.AllowedTypes {
			if typeStr == allowed {
				valid = true
				break
			}
		}
		if !valid {
			return NewErrorResponse(
				ErrCodeInvalidMessageType,
				"Invalid message type: "+typeStr,
				map[string]any{
					"received": typeStr,
					"allowed":  v.AllowedTypes,
				},
			)
		}
	}

	return nil
}

// DefaultMessageTypeValidator returns a validator for "type" field
func DefaultMessageTypeValidator(allowedTypes []string) *MessageTypeValidator {
	return &MessageTypeValidator{
		RequiredField: "type",
		AllowedTypes:  allowedTypes,
	}
}
