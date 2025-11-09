package errors

import pkgErrors "shared/pkg/errors"

// ============================================================================
// API Gateway Error Codes
// ============================================================================

const (
	// Service Discovery Errors
	CodeServiceNotFound    = "GW_SERVICE_NOT_FOUND"
	CodeServiceUnavailable = "GW_SERVICE_UNAVAILABLE"
	CodeNoHealthyInstances = "GW_NO_HEALTHY_INSTANCES"

	// Routing Errors
	CodeRoutingFailed = "GW_ROUTING_FAILED"
	CodeInvalidRoute  = "GW_INVALID_ROUTE"
	CodeRouteNotFound = "GW_ROUTE_NOT_FOUND"

	// Proxy Errors
	CodeProxyError       = "GW_PROXY_ERROR"
	CodeUpstreamTimeout  = "GW_UPSTREAM_TIMEOUT"
	CodeUpstreamError    = "GW_UPSTREAM_ERROR"
	CodeConnectionFailed = "GW_CONNECTION_FAILED"

	// Rate Limiting Errors
	CodeRateLimitExceeded = "GW_RATE_LIMIT_EXCEEDED"
	CodeQuotaExceeded     = "GW_QUOTA_EXCEEDED"

	// Circuit Breaker Errors
	CodeCircuitBreakerOpen  = "GW_CIRCUIT_BREAKER_OPEN"
	CodeCircuitBreakerError = "GW_CIRCUIT_BREAKER_ERROR"

	// Authentication/Authorization Errors
	CodeMissingAuthHeader    = "GW_MISSING_AUTH_HEADER"
	CodeInvalidAuthToken     = "GW_INVALID_AUTH_TOKEN"
	CodeAuthenticationFailed = "GW_AUTH_FAILED"

	// Request Errors
	CodeInvalidRequestFormat = "GW_INVALID_REQUEST"
	CodeRequestTooLarge      = "GW_REQUEST_TOO_LARGE"
	CodeInvalidContentType   = "GW_INVALID_CONTENT_TYPE"

	// Configuration Errors
	CodeConfigurationError   = "GW_CONFIG_ERROR"
	CodeInvalidConfiguration = "GW_INVALID_CONFIG"
)

// ============================================================================
// Service Name
// ============================================================================

const ServiceName = "api-gateway"

// ============================================================================
// HTTP Status Code Mapping
// ============================================================================

var HTTPStatusMap = map[string]int{
	// Service Discovery Errors
	CodeServiceNotFound:    503,
	CodeServiceUnavailable: 503,
	CodeNoHealthyInstances: 503,

	// Routing Errors
	CodeRoutingFailed: 502,
	CodeInvalidRoute:  400,
	CodeRouteNotFound: 404,

	// Proxy Errors
	CodeProxyError:       502,
	CodeUpstreamTimeout:  504,
	CodeUpstreamError:    502,
	CodeConnectionFailed: 503,

	// Rate Limiting Errors
	CodeRateLimitExceeded: 429,
	CodeQuotaExceeded:     429,

	// Circuit Breaker Errors
	CodeCircuitBreakerOpen:  503,
	CodeCircuitBreakerError: 500,

	// Authentication/Authorization Errors
	CodeMissingAuthHeader:    401,
	CodeInvalidAuthToken:     401,
	CodeAuthenticationFailed: 401,

	// Request Errors
	CodeInvalidRequestFormat: 400,
	CodeRequestTooLarge:      413,
	CodeInvalidContentType:   415,

	// Configuration Errors
	CodeConfigurationError:   500,
	CodeInvalidConfiguration: 500,
}

// HTTPStatus returns the HTTP status code for an api-gateway error code
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

// NewGatewayError creates a new api-gateway error with service context
func NewGatewayError(code, message string) pkgErrors.Error {
	return pkgErrors.New(code, message).WithService(ServiceName)
}