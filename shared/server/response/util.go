package response

import (
	"shared/pkg/errors"
	"time"
)

// ErrorDetailsFromError converts a standard error to ErrorDetails
func ErrorDetailsFromError(err error, includeStackTrace bool) *ErrorDetails {
	if err == nil {
		return nil
	}

	details := &ErrorDetails{
		Message:   err.Error(),
		Timestamp: time.Now(),
		Context:   make(map[string]interface{}),
	}

	if appErr, ok := err.(errors.Error); ok {
		details.Code = appErr.Code()
		details.Message = appErr.Message()

		details.Type, details.Severity = MapErrorCodeToTypeAndSeverity(appErr.Code())
		details.Retryable = IsRetryable(appErr.Code())

		if includeStackTrace && appErr.StackTrace() != nil {
			stackTrace := make([]string, len(appErr.StackTrace()))
			for i, frame := range appErr.StackTrace() {
				stackTrace[i] = frame
			}
			details.StackTrace = stackTrace
		}

		if appErr.Details() != nil {
			for k, v := range appErr.Details() {
				details.Context[k] = v
			}
		}

		if innerErr := appErr.Unwrap(); innerErr != nil {
			details.InnerError = innerErr.Error()
		}
	} else {
		details.Code = errors.CodeInternal
		details.Type = ErrorTypeInternal
		details.Severity = SeverityHigh
		details.Retryable = false
	}

	return details
}

// MapErrorCodeToTypeAndSeverity maps error codes to types and severity
func MapErrorCodeToTypeAndSeverity(code string) (ErrorType, ErrorSeverity) {
	switch code {
	case errors.CodeValidationFailed:
		return ErrorTypeValidation, SeverityLow
	case errors.CodeInvalidArgument:
		return ErrorTypeBadRequest, SeverityLow
	case errors.CodeUnauthenticated, errors.CodeTokenExpired, errors.CodeTokenInvalid:
		return ErrorTypeAuthentication, SeverityMedium
	case errors.CodePermissionDenied:
		return ErrorTypeAuthorization, SeverityMedium
	case errors.CodeNotFound:
		return ErrorTypeNotFound, SeverityLow
	case errors.CodeAlreadyExists, errors.CodeConflict:
		return ErrorTypeConflict, SeverityMedium
	case errors.CodeRateLimitExceeded, errors.CodeResourceExhausted:
		return ErrorTypeRateLimit, SeverityMedium
	case errors.CodeUnavailable:
		return ErrorTypeUnavailable, SeverityHigh
	case errors.CodeDeadlineExceeded:
		return ErrorTypeTimeout, SeverityHigh
	case errors.CodeInternal, errors.CodeDataLoss:
		return ErrorTypeInternal, SeverityCritical
	default:
		return ErrorTypeInternal, SeverityHigh
	}
}

// IsRetryable determines if an error is retriable
func IsRetryable(code string) bool {
	switch code {
	case errors.CodeUnavailable, errors.CodeDeadlineExceeded, errors.CodeResourceExhausted:
		return true
	default:
		return false
	}
}

// GetRetryAfter calculates retry-after seconds for an error
func GetRetryAfter(code string) *int {
	if !IsRetryable(code) {
		return nil
	}

	var seconds int
	switch code {
	case errors.CodeRateLimitExceeded, errors.CodeResourceExhausted:
		seconds = 60 // 1 minute for rate limits
	case errors.CodeUnavailable:
		seconds = 30 // 30 seconds for unavailable services
	case errors.CodeDeadlineExceeded:
		seconds = 5 // 5 seconds for timeouts
	default:
		seconds = 10
	}

	return &seconds
}
