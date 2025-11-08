package request

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/go-playground/validator/v10"
)

const (
	DefaultMaxBodySize = 1 << 20 // 1MB
)

// ParseJSON parses JSON body from request
func (h *RequestHandler) ParseJSON(v interface{}) error {
	return h.ParseJSONWithMaxSize(v, DefaultMaxBodySize)
}

// ParseJSONWithMaxSize parses JSON body with custom max size
func (h *RequestHandler) ParseJSONWithMaxSize(v interface{}, maxSize int64) error {
	if h.request.Body == nil {
		return fmt.Errorf("request body is empty")
	}

	decoder := json.NewDecoder(h.request.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(v); err != nil {
		if err == io.EOF {
			return fmt.Errorf("request body is empty")
		}
		if jsonErr, ok := err.(*json.SyntaxError); ok {
			return fmt.Errorf("invalid JSON at position %d", jsonErr.Offset)
		}
		if jsonErr, ok := err.(*json.UnmarshalTypeError); ok {
			return fmt.Errorf("invalid type for field %s: expected %s", jsonErr.Field, jsonErr.Type)
		}
		return fmt.Errorf("failed to parse JSON: %v", err)
	}

	if decoder.More() {
		return fmt.Errorf("request body must contain only one JSON object")
	}

	return nil
}

// ParseJSONAllowUnknown parses JSON body allowing unknown fields
func (h *RequestHandler) ParseJSONAllowUnknown(v interface{}) error {
	if h.request.Body == nil {
		return fmt.Errorf("request body is empty")
	}

	decoder := json.NewDecoder(h.request.Body)

	if err := decoder.Decode(v); err != nil {
		if err == io.EOF {
			return fmt.Errorf("request body is empty")
		}
		return err
	}

	return nil
}

// Validate validates a struct using the handler's validator
func (h *RequestHandler) Validate(v interface{}) error {
	return h.validator.Struct(v)
}

// ValidateWithValidator validates a struct using a custom validator
func (h *RequestHandler) ValidateWithValidator(v interface{}, validate *validator.Validate) error {
	return validate.Struct(v)
}
