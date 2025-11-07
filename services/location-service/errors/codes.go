package errors

import pkgErrors "shared/pkg/errors"

// ============================================================================
// Location Service Error Codes
// ============================================================================

const (
	// Lookup Errors
	CodeLookupFailed     = "LOC_LOOKUP_FAILED"
	CodeInvalidIP        = "LOC_INVALID_IP"
	CodeIPNotFound       = "LOC_IP_NOT_FOUND"
	CodePrivateIPAddress = "LOC_PRIVATE_IP"

	// Database Errors
	CodeDatabaseNotFound   = "LOC_DATABASE_NOT_FOUND"
	CodeDatabaseLoadFailed = "LOC_DATABASE_LOAD_FAILED"
	CodeDatabaseCorrupted  = "LOC_DATABASE_CORRUPTED"
	CodeDatabaseOutdated   = "LOC_DATABASE_OUTDATED"

	// Service Errors
	CodeServiceUnavailable = "LOC_SERVICE_UNAVAILABLE"
	CodeRateLimitExceeded  = "LOC_RATE_LIMIT_EXCEEDED"

	// Data Quality Errors
	CodeIncompleteData = "LOC_INCOMPLETE_DATA"
	CodeLowAccuracy    = "LOC_LOW_ACCURACY"
)

// ============================================================================
// Service Name
// ============================================================================

const ServiceName = "location-service"

// ============================================================================
// HTTP Status Code Mapping
// ============================================================================

var HTTPStatusMap = map[string]int{
	// Lookup Errors
	CodeLookupFailed:     500,
	CodeInvalidIP:        400,
	CodeIPNotFound:       404,
	CodePrivateIPAddress: 400,

	// Database Errors
	CodeDatabaseNotFound:   503,
	CodeDatabaseLoadFailed: 500,
	CodeDatabaseCorrupted:  500,
	CodeDatabaseOutdated:   503,

	// Service Errors
	CodeServiceUnavailable: 503,
	CodeRateLimitExceeded:  429,

	// Data Quality Errors
	CodeIncompleteData: 200, // Still return data but with warning
	CodeLowAccuracy:    200, // Still return data but with warning
}

// HTTPStatus returns the HTTP status code for a location service error code
func HTTPStatus(code string) int {
	if status, ok := HTTPStatusMap[code]; ok {
		return status
	}
	// Fallback to shared error codes
	return pkgErrors.HTTPStatus(code)
}

// ============================================================================
// Error Constructor Helpers
// ============================================================================

// NewLocationError creates a new location service error with service context
func NewLocationError(code, message string) pkgErrors.Error {
	return pkgErrors.New(code, message).WithService(ServiceName)
}

// WrapLocationError wraps an error with location service context
func WrapLocationError(err error, code, message string) pkgErrors.Error {
	if err == nil {
		return nil
	}
	return pkgErrors.Wrap(err, code, message).WithService(ServiceName)
}
