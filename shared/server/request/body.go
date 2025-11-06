package request

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-playground/validator/v10"
)

const (
	DefaultMaxBodySize = 1 << 20 // 1MB
)

var defaultValidator = validator.New()

// ParseJSON parses JSON body from request
func ParseJSON(r *http.Request, v interface{}) error {
	return ParseJSONWithMaxSize(r, v, DefaultMaxBodySize)
}

// ParseJSONWithMaxSize parses JSON body with custom max size
func ParseJSONWithMaxSize(r *http.Request, v interface{}, maxSize int64) error {
	if r.Body == nil {
		return fmt.Errorf("request body is empty")
	}

	r.Body = http.MaxBytesReader(nil, r.Body, maxSize)

	decoder := json.NewDecoder(r.Body)
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
func ParseJSONAllowUnknown(r *http.Request, v interface{}) error {
	if r.Body == nil {
		return fmt.Errorf("request body is empty")
	}

	r.Body = http.MaxBytesReader(nil, r.Body, DefaultMaxBodySize)
	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(v); err != nil {
		if err == io.EOF {
			return fmt.Errorf("request body is empty")
		}
		return err
	}

	return nil
}

// Validate validates a struct using the default validator
func Validate(v interface{}) error {
	return defaultValidator.Struct(v)
}

// ValidateWithValidator validates a struct using a custom validator
func ValidateWithValidator(v interface{}, validate *validator.Validate) error {
	return validate.Struct(v)
}
