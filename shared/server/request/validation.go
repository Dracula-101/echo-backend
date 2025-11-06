package request

import (
	"github.com/go-playground/validator/v10"
)

// Config holds request parsing configuration
type Config struct {
	MaxBodySize        int64
	DisallowUnknown    bool
	RequireContentType bool
	AllowEmptyBody     bool
}

// Validator interface for request validation
type Validator interface {
	GetValue() interface{}
	ValidateErrors(validator.ValidationErrors) ([]ValidationErrorDetail, error)
}

// ValidationErrorDetail holds validation error details
type ValidationErrorDetail struct {
	Msg  string
	Code string
}

// Validation error codes
const (
	REQUIRED_FIELD   = "REQUIRED_FIELD"
	INVALID_FORMAT   = "INVALID_FORMAT"
	TOO_SHORT        = "TOO_SHORT"
	TOO_LONG         = "TOO_LONG"
	PATTERN_MISMATCH = "PATTERN_MISMATCH"
)
