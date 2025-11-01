package validator

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"unicode"

	"github.com/golang-jwt/jwt/v5"
)

// Validator interface for custom validators
type Validator interface {
	Validate(value any) error
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
	Tag     string
}

func (v ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", v.Field, v.Message)
}

// ValidationErrors is a collection of validation errors
type ValidationErrors []ValidationError

func (v ValidationErrors) Error() string {
	if len(v) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("validation failed: ")
	for i, err := range v {
		if i > 0 {
			sb.WriteString("; ")
		}
		sb.WriteString(err.Error())
	}
	return sb.String()
}

// Common regex patterns
var (
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	phoneRegex    = regexp.MustCompile(`^\+?[1-9]\d{1,14}$`) // E.164 format
	uuidRegex     = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	alphaNumRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	slugRegex     = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
)

// String validators

// IsRequired checks if a string is not empty
func IsRequired(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return ValidationError{
			Field:   field,
			Message: "is required",
			Tag:     "required",
		}
	}
	return nil
}

// IsEmail validates email format
func IsEmail(field, value string) error {
	if !emailRegex.MatchString(value) {
		return ValidationError{
			Field:   field,
			Message: "must be a valid email address",
			Tag:     "email",
		}
	}
	// Additional validation using net/mail
	if _, err := mail.ParseAddress(value); err != nil {
		return ValidationError{
			Field:   field,
			Message: "must be a valid email address",
			Tag:     "email",
		}
	}
	return nil
}

// IsPhone validates phone number in E.164 format
func IsPhone(field, value string) error {
	if !phoneRegex.MatchString(value) {
		return ValidationError{
			Field:   field,
			Message: "must be a valid phone number in E.164 format",
			Tag:     "phone",
		}
	}
	return nil
}

// IsUUID validates UUID format
func IsUUID(field, value string) error {
	if !uuidRegex.MatchString(strings.ToLower(value)) {
		return ValidationError{
			Field:   field,
			Message: "must be a valid UUID",
			Tag:     "uuid",
		}
	}
	return nil
}

// IsAlphanumeric checks if string contains only letters and numbers
func IsAlphanumeric(field, value string) error {
	if !alphaNumRegex.MatchString(value) {
		return ValidationError{
			Field:   field,
			Message: "must contain only letters and numbers",
			Tag:     "alphanumeric",
		}
	}
	return nil
}

// IsSlug validates URL-friendly slug format
func IsSlug(field, value string) error {
	if !slugRegex.MatchString(value) {
		return ValidationError{
			Field:   field,
			Message: "must be a valid slug (lowercase letters, numbers, and hyphens)",
			Tag:     "slug",
		}
	}
	return nil
}

// MinLength checks minimum string length
func MinLength(field, value string, min int) error {
	if len(value) < min {
		return ValidationError{
			Field:   field,
			Message: fmt.Sprintf("must be at least %d characters long", min),
			Tag:     "minlength",
		}
	}
	return nil
}

// MaxLength checks maximum string length
func MaxLength(field, value string, max int) error {
	if len(value) > max {
		return ValidationError{
			Field:   field,
			Message: fmt.Sprintf("must be at most %d characters long", max),
			Tag:     "maxlength",
		}
	}
	return nil
}

// LengthBetween checks if string length is within range
func LengthBetween(field, value string, min, max int) error {
	length := len(value)
	if length < min || length > max {
		return ValidationError{
			Field:   field,
			Message: fmt.Sprintf("must be between %d and %d characters long", min, max),
			Tag:     "lengthbetween",
		}
	}
	return nil
}

// Password validators

// IsStrongPassword validates password strength
// Requires: min 8 chars, at least 1 uppercase, 1 lowercase, 1 number, 1 special char
func IsStrongPassword(field, value string) error {
	if len(value) < 8 {
		return ValidationError{
			Field:   field,
			Message: "password must be at least 8 characters long",
			Tag:     "password",
		}
	}

	var (
		hasUpper   bool
		hasLower   bool
		hasNumber  bool
		hasSpecial bool
	)

	for _, char := range value {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if !hasUpper || !hasLower || !hasNumber || !hasSpecial {
		return ValidationError{
			Field:   field,
			Message: "password must contain at least one uppercase letter, one lowercase letter, one number, and one special character",
			Tag:     "password",
		}
	}

	return nil
}

// Numeric validators

// IsPositive checks if number is positive
func IsPositive(field string, value int) error {
	if value <= 0 {
		return ValidationError{
			Field:   field,
			Message: "must be a positive number",
			Tag:     "positive",
		}
	}
	return nil
}

// IsNonNegative checks if number is non-negative
func IsNonNegative(field string, value int) error {
	if value < 0 {
		return ValidationError{
			Field:   field,
			Message: "must be a non-negative number",
			Tag:     "nonnegative",
		}
	}
	return nil
}

// InRange checks if number is within range (inclusive)
func InRange(field string, value, min, max int) error {
	if value < min || value > max {
		return ValidationError{
			Field:   field,
			Message: fmt.Sprintf("must be between %d and %d", min, max),
			Tag:     "range",
		}
	}
	return nil
}

// Collection validators

// IsOneOf checks if value is in allowed list
func IsOneOf(field, value string, allowed []string) error {
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}
	return ValidationError{
		Field:   field,
		Message: fmt.Sprintf("must be one of: %s", strings.Join(allowed, ", ")),
		Tag:     "oneof",
	}
}

// IsNotEmpty checks if slice is not empty
func IsNotEmpty[T any](field string, values []T) error {
	if len(values) == 0 {
		return ValidationError{
			Field:   field,
			Message: "must not be empty",
			Tag:     "notempty",
		}
	}
	return nil
}

// HasMinItems checks minimum slice length
func HasMinItems[T any](field string, values []T, min int) error {
	if len(values) < min {
		return ValidationError{
			Field:   field,
			Message: fmt.Sprintf("must have at least %d items", min),
			Tag:     "minitems",
		}
	}
	return nil
}

// HasMaxItems checks maximum slice length
func HasMaxItems[T any](field string, values []T, max int) error {
	if len(values) > max {
		return ValidationError{
			Field:   field,
			Message: fmt.Sprintf("must have at most %d items", max),
			Tag:     "maxitems",
		}
	}
	return nil
}

// Custom pattern validator

// MatchesPattern validates string against custom regex pattern
func MatchesPattern(field, value, pattern, message string) error {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return ValidationError{
			Field:   field,
			Message: "invalid pattern for validation",
			Tag:     "pattern",
		}
	}
	if !regex.MatchString(value) {
		return ValidationError{
			Field:   field,
			Message: message,
			Tag:     "pattern",
		}
	}
	return nil
}

// Token validator
func ValidateJWTToken(token string, secretKey string) (user_id string, err error) {
	parsedToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secretKey), nil
	})
	if err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok || !parsedToken.Valid {
		return "", fmt.Errorf("invalid token claims")
	}
	uid, ok := claims["user_id"].(string)
	if !ok || uid == "" {
		return "", fmt.Errorf("user_id claim missing or invalid")
	}
	return uid, nil
}

// ChainValidator allows chaining multiple validators
type ChainValidator struct {
	errors ValidationErrors
}

// NewChainValidator creates a new chain validator
func NewChainValidator() *ChainValidator {
	return &ChainValidator{
		errors: make(ValidationErrors, 0),
	}
}

// Add adds a validation error if err is not nil
func (c *ChainValidator) Add(err error) *ChainValidator {
	if err != nil {
		if ve, ok := err.(ValidationError); ok {
			c.errors = append(c.errors, ve)
		} else {
			// Convert generic error to ValidationError
			c.errors = append(c.errors, ValidationError{
				Message: err.Error(),
			})
		}
	}
	return c
}

// Validate returns validation errors if any
func (c *ChainValidator) Validate() error {
	if len(c.errors) > 0 {
		return c.errors
	}
	return nil
}

// HasErrors checks if there are any validation errors
func (c *ChainValidator) HasErrors() bool {
	return len(c.errors) > 0
}

// Errors returns the validation errors
func (c *ChainValidator) Errors() ValidationErrors {
	return c.errors
}
