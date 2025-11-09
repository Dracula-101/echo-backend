package response

import (
	"context"
	"fmt"
	"net/http"

	"shared/pkg/errors"
	"shared/pkg/logger"
)

// JSON sends a success response with data
func JSON(w http.ResponseWriter, statusCode int, data any) error {
	return Success().
		WithData(data).
		Send(w, statusCode)
}

// JSONWithContext sends a success response with context
func JSONWithContext(ctx context.Context, r *http.Request, w http.ResponseWriter, statusCode int, data any) error {
	return Success().
		WithContext(ctx).
		WithRequest(r).
		WithData(data).
		Send(w, statusCode)
}

// JSONWithMessage sends a success response with data and message
func JSONWithMessage(ctx context.Context, r *http.Request, w http.ResponseWriter, statusCode int, message string, data any) error {
	return Success().
		WithContext(ctx).
		WithRequest(r).
		WithMessage(message).
		WithData(data).
		Send(w, statusCode)
}

// Paginated sends a paginated response
func Paginated(ctx context.Context, r *http.Request, w http.ResponseWriter, data any, pagination *PaginationInfo) error {
	return Success().
		WithContext(ctx).
		WithRequest(r).
		WithData(data).
		WithPagination(pagination).
		OK(w)
}

// WithHATEOAS sends a response with HATEOAS links
func WithHATEOAS(ctx context.Context, r *http.Request, w http.ResponseWriter, statusCode int, data any, links ...Link) error {
	return Success().
		WithContext(ctx).
		WithRequest(r).
		WithData(data).
		WithLinks(links...).
		Send(w, statusCode)
}

// ================ Error Response Helpers ================

// RespondWithError sends an error response
func RespondWithError(ctx context.Context, r *http.Request, w http.ResponseWriter, statusCode int, err error) error {
	config := GetGlobalConfig()
	errorDetails := ErrorDetailsFromError(err, config.ShouldIncludeStackTrace())

	return Error().
		WithContext(ctx).
		WithRequest(r).
		WithError(errorDetails).
		WithMessage(err.Error()).
		Send(w, statusCode)
}

// BadRequestError creates a 400 Bad Request error response
func BadRequestError(ctx context.Context, r *http.Request, w http.ResponseWriter, message string, reqError error) error {
	err := errors.New(errors.CodeInvalidArgument, message)
	errorDetails := ErrorDetailsFromError(err, false)
	errorDetails.Type = ErrorTypeBadRequest
	if reqError != nil {
		errorDetails.InnerError = reqError.Error()
	} else {
		errorDetails.InnerError = ""
	}
	errorDetails.Description = "The request could not be understood or was missing required parameters."

	return Error().
		WithContext(ctx).
		WithRequest(r).
		WithError(errorDetails).
		WithMessage(message).
		BadRequest(w)
}

// UnauthorizedError creates a 401 Unauthorized error response
func UnauthorizedError(ctx context.Context, r *http.Request, w http.ResponseWriter, message string, reqError error) error {
	err := errors.New(errors.CodeUnauthenticated, message)
	errorDetails := ErrorDetailsFromError(err, false)
	errorDetails.Type = ErrorTypeAuthentication
	errorDetails.Description = "Authentication is required and has failed or has not been provided."
	if reqError != nil {
		errorDetails.InnerError = reqError.Error()
	} else {
		errorDetails.InnerError = ""
	}

	return Error().
		WithContext(ctx).
		WithRequest(r).
		WithError(errorDetails).
		WithMessage(message).
		Unauthorized(w)
}

// ForbiddenError creates a 403 Forbidden error response
func ForbiddenError(ctx context.Context, r *http.Request, w http.ResponseWriter, message string, reqError error) error {
	err := errors.New(errors.CodePermissionDenied, message)
	errorDetails := ErrorDetailsFromError(err, false)
	errorDetails.Description = "You do not have permission to access this resource."
	errorDetails.InnerError = reqError.Error()

	return Error().
		WithContext(ctx).
		WithRequest(r).
		WithError(errorDetails).
		WithMessage(message).
		Forbidden(w)
}

// NotFoundError creates a 404 Not Found error response
func NotFoundError(ctx context.Context, r *http.Request, w http.ResponseWriter, resource string) error {
	message := fmt.Sprintf("%s not found", resource)
	err := errors.New(errors.CodeNotFound, message)
	errorDetails := ErrorDetailsFromError(err, false)
	errorDetails.Description = "The requested resource could not be found."
	errorDetails.Context = map[string]interface{}{
		"resource": resource,
		"path":     r.URL.Path,
	}

	return Error().
		WithContext(ctx).
		WithRequest(r).
		WithError(errorDetails).
		WithMessage(message).
		NotFound(w)
}

// ConflictError creates a 409 Conflict error response
func ConflictError(ctx context.Context, r *http.Request, w http.ResponseWriter, message string, reqError error) error {
	err := errors.New(errors.CodeConflict, message)
	errorDetails := ErrorDetailsFromError(err, false)
	errorDetails.Description = "The request could not be completed due to a conflict."
	errorDetails.Type = ErrorTypeConflict
	if reqError != nil {
		errorDetails.InnerError = reqError.Error()
	} else {
		errorDetails.InnerError = ""
	}

	return Error().
		WithContext(ctx).
		WithRequest(r).
		WithError(errorDetails).
		WithMessage(message).
		Conflict(w)
}

// ValidationError creates a 422 Unprocessable Entity error with field errors
func ValidationError(ctx context.Context, r *http.Request, w http.ResponseWriter, fieldErrors []FieldError) error {
	err := errors.New(errors.CodeValidationFailed, "Validation failed")
	errorDetails := ErrorDetailsFromError(err, false)
	errorDetails.Description = "One or more fields failed validation."
	errorDetails.Type = ErrorTypeValidation
	errorDetails.Fields = fieldErrors

	return Error().
		WithContext(ctx).
		WithRequest(r).
		WithError(errorDetails).
		WithMessage("Validation failed").
		UnprocessableEntity(w)
}

// RateLimitError creates a 429 Too Many Requests error response
func RateLimitError(ctx context.Context, r *http.Request, w http.ResponseWriter, retryAfter int) error {
	err := errors.New(errors.CodeRateLimitExceeded, "Rate limit exceeded")
	errorDetails := ErrorDetailsFromError(err, false)
	errorDetails.Description = "You have exceeded the rate limit. Please try again later."
	errorDetails.Type = ErrorTypeRateLimit

	w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))

	return Error().
		WithContext(ctx).
		WithRequest(r).
		WithError(errorDetails).
		WithMessage("Rate limit exceeded").
		TooManyRequests(w)
}

// InternalServerError creates a 500 Internal Server Error response
func InternalServerError(ctx context.Context, r *http.Request, w http.ResponseWriter, message string, err error) error {
	config := GetGlobalConfig()

	var appErr errors.Error
	if err != nil {
		if e, ok := err.(errors.Error); ok {
			appErr = e
		} else {
			appErr = errors.New(errors.CodeInternal, err.Error())
		}
	} else {
		appErr = errors.New(errors.CodeInternal, "Internal server error")
	}

	errorDetails := ErrorDetailsFromError(appErr, config.ShouldIncludeStackTrace())
	errorDetails.Description = "An unexpected error occurred. Please try again later."
	errorDetails.Type = ErrorTypeInternal

	// Don't expose internal details in production
	if config.IsProduction() {
		errorDetails.InnerError = ""
		errorDetails.Context = nil
		errorDetails.StackTrace = nil
	}

	return Error().
		WithContext(ctx).
		WithRequest(r).
		WithError(errorDetails).
		WithMessage(fmt.Sprintf("Internal Server Error - %s", message)).
		InternalServerError(w)
}

// ServiceUnavailableError creates a 503 Service Unavailable error response
func ServiceUnavailableError(ctx context.Context, r *http.Request, w http.ResponseWriter, service string, retryAfter int) error {
	message := fmt.Sprintf("%s is unavailable", service)
	err := errors.New(errors.CodeUnavailable, message)
	errorDetails := ErrorDetailsFromError(err, false)
	errorDetails.Type = ErrorTypeServiceUnavailable
	errorDetails.Description = "The service is temporarily unavailable. Please try again later."
	errorDetails.Context = map[string]interface{}{
		"service": service,
	}

	w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))

	return Error().
		WithContext(ctx).
		WithRequest(r).
		WithError(errorDetails).
		WithMessage(message).
		ServiceUnavailable(w)
}

// GatewayTimeoutError creates a 504 Gateway Timeout error response
func GatewayTimeoutError(ctx context.Context, r *http.Request, w http.ResponseWriter, service string) error {
	message := fmt.Sprintf("%s service timeout", service)
	err := errors.New(errors.CodeDeadlineExceeded, message)
	errorDetails := ErrorDetailsFromError(err, false)
	errorDetails.Description = "The upstream service did not respond in time."
	errorDetails.Type = ErrorTypeGatewayTimeout
	errorDetails.Context = map[string]interface{}{
		"service": service,
	}

	return Error().
		WithContext(ctx).
		WithRequest(r).
		WithError(errorDetails).
		WithMessage(message).
		GatewayTimeout(w)
}

// CircuitOpenError creates a circuit breaker open error
func CircuitOpenError(ctx context.Context, r *http.Request, w http.ResponseWriter, service string) error {
	message := fmt.Sprintf("%s service circuit is open", service)
	err := errors.New(errors.CodeUnavailable, message)
	errorDetails := ErrorDetailsFromError(err, false)
	errorDetails.Type = ErrorTypeCircuitOpen
	errorDetails.Description = "The service is currently unavailable due to too many failures."
	errorDetails.Context = map[string]interface{}{
		"service":       service,
		"circuit_state": "open",
	}

	w.Header().Set("Retry-After", "30")

	return Error().
		WithContext(ctx).
		WithRequest(r).
		WithError(errorDetails).
		WithMessage(message).
		ServiceUnavailable(w)
}

// TooManyRequestsError creates a 429 Too Many Requests error response
func TooManyRequestsError(ctx context.Context, r *http.Request, w http.ResponseWriter, message string, retryAfter int) error {
	err := errors.New(errors.CodeRateLimitExceeded, message)
	errorDetails := ErrorDetailsFromError(err, false)
	errorDetails.Type = ErrorTypeRateLimit
	errorDetails.Description = "You have sent too many requests in a given amount of time."

	w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))

	return Error().
		WithContext(ctx).
		WithRequest(r).
		WithError(errorDetails).
		WithMessage(message).
		TooManyRequests(w)
}

// UnsupportedMediaTypeError creates a 415 Unsupported Media Type error response
func UnsupportedMediaTypeError(ctx context.Context, r *http.Request, w http.ResponseWriter, mediaType string) error {
	message := fmt.Sprintf("Unsupported media type: %s", mediaType)
	err := errors.New(errors.CodeUnsupportedMediaType, message)
	errorDetails := ErrorDetailsFromError(err, false)
	errorDetails.Type = ErrorTypeUnsupportedMediaType
	errorDetails.Description = "The request entity has a media type which the server or resource does not support."

	return Error().
		WithContext(ctx).
		WithRequest(r).
		WithError(errorDetails).
		WithMessage(message).
		UnsupportedMediaType(w)
}

func ConflictResourceError(ctx context.Context, r *http.Request, w http.ResponseWriter, resource string, reason string) error {
	message := fmt.Sprintf("Conflict with resource: %s", resource)
	err := errors.New(errors.CodeConflict, message)
	errorDetails := ErrorDetailsFromError(err, false)
	errorDetails.Description = reason
	errorDetails.Type = ErrorTypeConflictResource
	errorDetails.Context = map[string]interface{}{
		"resource": resource,
	}

	return Error().
		WithContext(ctx).
		WithRequest(r).
		WithError(errorDetails).
		WithMessage(message).
		Conflict(w)
}

func ServiceDependencyError(ctx context.Context, r *http.Request, w http.ResponseWriter, service string, reason string) error {
	message := fmt.Sprintf("Service dependency error: %s", service)
	err := errors.New(errors.CodeUnavailable, message)
	errorDetails := ErrorDetailsFromError(err, false)
	errorDetails.Description = reason
	errorDetails.Type = ErrorTypeServiceDependency
	errorDetails.Context = map[string]interface{}{
		"service": service,
	}
	// Return the error response
	return Error().
		WithContext(ctx).
		WithRequest(r).
		WithError(errorDetails).
		WithMessage(message).
		ServiceUnavailable(w)
}

func InvalidCredentialsError(ctx context.Context, r *http.Request, w http.ResponseWriter, message string) error {
	err := errors.New(errors.CodeUnauthenticated, message)
	errorDetails := ErrorDetailsFromError(err, false)
	errorDetails.Description = "The provided credentials are invalid."
	errorDetails.Type = ErrorTypeInvalidCredentials
	return Error().
		WithContext(ctx).
		WithRequest(r).
		WithError(errorDetails).
		WithMessage(message).
		Unauthorized(w)
}

func RouteNotFoundError(ctx context.Context, r *http.Request, w http.ResponseWriter, log logger.Logger) error {
	message := "Route not found"
	err := errors.New(errors.CodeNotFound, message)
	errorDetails := ErrorDetailsFromError(err, false)
	errorDetails.Description = "The requested route does not exist."
	errorDetails.Type = ErrorTypeNotFound
	errorDetails.Context = map[string]interface{}{
		"path":   r.URL.Path,
		"method": r.Method,
	}
	log.Warn("Route not found",
		logger.String("path", r.URL.Path),
		logger.String("method", r.Method),
	)
	return Error().
		WithContext(ctx).
		WithRequest(r).
		WithError(errorDetails).
		WithMessage(message).
		NotFound(w)
}

func MethodNotAllowedError(ctx context.Context, r *http.Request, w http.ResponseWriter) error {
	message := "Method not allowed"
	err := errors.New(errors.CodePermissionDenied, message)
	errorDetails := ErrorDetailsFromError(err, false)
	errorDetails.Description = "The HTTP method used is not allowed for this route."
	errorDetails.Type = ErrorTypeMethodNotAllowed
	errorDetails.Context = map[string]interface{}{
		"path":   r.URL.Path,
		"method": r.Method,
	}
	return Error().
		WithContext(ctx).
		WithRequest(r).
		WithError(errorDetails).
		WithMessage(message).
		MethodNotAllowed(w)
}

// ================ Field Error Builders ================

// RequiredFieldError creates a required field error
func RequiredFieldError(field string) FieldError {
	return FieldError{
		Field:   field,
		Message: fmt.Sprintf("%s is required", field),
		Code:    "required",
	}
}

// InvalidFieldError creates an invalid field error
func InvalidFieldError(field, reason string) FieldError {
	return FieldError{
		Field:   field,
		Message: fmt.Sprintf("%s is invalid: %s", field, reason),
		Code:    "invalid",
	}
}

// MinLengthFieldError creates a minimum length validation error
func MinLengthFieldError(field string, minLength int) FieldError {
	return FieldError{
		Field:       field,
		Message:     fmt.Sprintf("%s must be at least %d characters", field, minLength),
		Code:        "min_length",
		Constraints: fmt.Sprintf("min=%d", minLength),
	}
}

// MaxLengthFieldError creates a maximum length validation error
func MaxLengthFieldError(field string, maxLength int) FieldError {
	return FieldError{
		Field:       field,
		Message:     fmt.Sprintf("%s must be at most %d characters", field, maxLength),
		Code:        "max_length",
		Constraints: fmt.Sprintf("max=%d", maxLength),
	}
}

// ================ Pagination Builders ================

// NewOffsetPagination creates offset-based pagination info
func NewOffsetPagination(totalItems int64, currentPage, pageSize, itemsInPage int, hasNext, hasPrev bool) *PaginationInfo {
	totalPages := int((totalItems + int64(pageSize) - 1) / int64(pageSize))

	return &PaginationInfo{
		Type:        PaginationOffset,
		TotalItems:  &totalItems,
		TotalPages:  &totalPages,
		CurrentPage: &currentPage,
		PageSize:    pageSize,
		HasNext:     hasNext,
		HasPrevious: hasPrev,
		ItemsInPage: itemsInPage,
	}
}

// NewCursorPagination creates cursor-based pagination info
func NewCursorPagination(pageSize, itemsInPage int, hasNext, hasPrev bool, nextCursor, prevCursor *string) *PaginationInfo {
	return &PaginationInfo{
		Type:        PaginationCursor,
		PageSize:    pageSize,
		HasNext:     hasNext,
		HasPrevious: hasPrev,
		NextCursor:  nextCursor,
		PrevCursor:  prevCursor,
		ItemsInPage: itemsInPage,
	}
}

// ================ HATEOAS Link Builders ================

// SelfLink creates a self-referencing link
func SelfLink(href string) Link {
	return Link{
		Rel:         "self",
		Href:        href,
		Method:      "GET",
		Type:        "application/json",
		Description: "Current resource",
	}
}

// NextLink creates a next page link
func NextLink(href string) Link {
	return Link{
		Rel:         "next",
		Href:        href,
		Method:      "GET",
		Type:        "application/json",
		Description: "Next page of results",
	}
}

// PrevLink creates a previous page link
func PrevLink(href string) Link {
	return Link{
		Rel:         "prev",
		Href:        href,
		Method:      "GET",
		Type:        "application/json",
		Description: "Previous page of results",
	}
}

// CreateLink creates a resource creation link
func CreateLink(href string) Link {
	return Link{
		Rel:         "create",
		Href:        href,
		Method:      "POST",
		Type:        "application/json",
		Description: "Create new resource",
	}
}

// UpdateLink creates a resource update link
func UpdateLink(href string) Link {
	return Link{
		Rel:         "update",
		Href:        href,
		Method:      "PUT",
		Type:        "application/json",
		Description: "Update resource",
	}
}

// DeleteLink creates a resource deletion link
func DeleteLink(href string) Link {
	return Link{
		Rel:         "delete",
		Href:        href,
		Method:      "DELETE",
		Type:        "application/json",
		Description: "Delete resource",
	}
}
