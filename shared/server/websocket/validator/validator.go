package validator

import (
	"fmt"
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

// Error returns the error message
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field '%s': %s", e.Field, e.Message)
}

// Validator validates messages
type Validator interface {
	Validate(data interface{}) error
}

// Rule represents a validation rule
type Rule func(value interface{}) error

// FieldValidator validates individual fields
type FieldValidator struct {
	rules map[string][]Rule
}

// NewFieldValidator creates a new field validator
func NewFieldValidator() *FieldValidator {
	return &FieldValidator{
		rules: make(map[string][]Rule),
	}
}

// AddRule adds a validation rule for a field
func (v *FieldValidator) AddRule(field string, rule Rule) {
	v.rules[field] = append(v.rules[field], rule)
}

// Validate validates a map of fields
func (v *FieldValidator) Validate(data interface{}) error {
	fields, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("data must be a map[string]interface{}")
	}

	for field, rules := range v.rules {
		value, exists := fields[field]
		if !exists {
			return &ValidationError{
				Field:   field,
				Message: "field is required",
			}
		}

		for _, rule := range rules {
			if err := rule(value); err != nil {
				return &ValidationError{
					Field:   field,
					Message: err.Error(),
				}
			}
		}
	}

	return nil
}

// Required creates a required field rule
func Required() Rule {
	return func(value interface{}) error {
		if value == nil {
			return fmt.Errorf("field is required")
		}
		return nil
	}
}

// MaxLength creates a max length rule for strings
func MaxLength(max int) Rule {
	return func(value interface{}) error {
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("value must be a string")
		}
		if len(str) > max {
			return fmt.Errorf("length exceeds maximum of %d", max)
		}
		return nil
	}
}

// MinLength creates a min length rule for strings
func MinLength(min int) Rule {
	return func(value interface{}) error {
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("value must be a string")
		}
		if len(str) < min {
			return fmt.Errorf("length is less than minimum of %d", min)
		}
		return nil
	}
}
