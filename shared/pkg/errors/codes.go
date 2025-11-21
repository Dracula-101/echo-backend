package errors

const (
	// --- 2xx: Success ---
	CodeOK        = "OK"
	CodeCreated   = "CREATED"
	CodeAccepted  = "ACCEPTED"
	CodeNoContent = "NO_CONTENT"

	// --- 3xx: Redirection ---
	CodeMovedPermanently  = "MOVED_PERMANENTLY"
	CodeFound             = "FOUND"
	CodeSeeOther          = "SEE_OTHER"
	CodeNotModified       = "NOT_MODIFIED"
	CodeTemporaryRedirect = "TEMPORARY_REDIRECT"
	CodePermanentRedirect = "PERMANENT_REDIRECT"

	// --- 4xx: Client Errors ---
	CodeBadRequest            = "BAD_REQUEST"
	CodeUnauthorized          = "UNAUTHORIZED"
	CodeInvalidArgument       = "INVALID_ARGUMENT"
	CodeAlreadyExists         = "ALREADY_EXISTS"
	CodePaymentRequired       = "PAYMENT_REQUIRED"
	CodeForbidden             = "FORBIDDEN"
	CodeNotFound              = "NOT_FOUND"
	CodeMethodNotAllowed      = "METHOD_NOT_ALLOWED"
	CodeNotAcceptable         = "NOT_ACCEPTABLE"
	CodeAccessDenied          = "ACCESS_DENIED"
	CodeProxyAuthRequired     = "PROXY_AUTHENTICATION_REQUIRED"
	CodeTimeout               = "TIMEOUT"
	CodeRequestTimeout        = "REQUEST_TIMEOUT"
	CodeConflict              = "CONFLICT"
	CodeGone                  = "GONE"
	CodeLengthRequired        = "LENGTH_REQUIRED"
	CodePreconditionFailed    = "PRECONDITION_FAILED"
	CodeRequestEntityTooLarge = "REQUEST_ENTITY_TOO_LARGE"
	CodeRequestURITooLong     = "REQUEST_URI_TOO_LONG"
	CodeUnsupportedMediaType  = "UNSUPPORTED_MEDIA_TYPE"
	CodeRangeNotSatisfiable   = "RANGE_NOT_SATISFIABLE"
	CodeExpectationFailed     = "EXPECTATION_FAILED"
	CodeTooManyRequests       = "TOO_MANY_REQUESTS"
	CodeUnprocessableEntity   = "UNPROCESSABLE_ENTITY"
	CodeRateLimitExceeded     = "RATE_LIMIT_EXCEEDED"
	CodeValidationFailed      = "VALIDATION_FAILED"

	// --- 4xx: Auth-related ---
	CodeUnauthenticated  = "UNAUTHENTICATED"
	CodeTokenExpired     = "TOKEN_EXPIRED"
	CodeTokenInvalid     = "TOKEN_INVALID"
	CodePermissionDenied = "PERMISSION_DENIED"
	CodeSessionExpired   = "SESSION_EXPIRED"
	CodeCSRFTokenInvalid = "CSRF_TOKEN_INVALID"

	// --- 5xx: Server Errors ---
	CodeInternal           = "INTERNAL_ERROR"
	CodeNotImplemented     = "NOT_IMPLEMENTED"
	CodeBadGateway         = "BAD_GATEWAY"
	CodeServiceUnavailable = "SERVICE_UNAVAILABLE"
	CodeGatewayTimeout     = "GATEWAY_TIMEOUT"
	CodeDataLoss           = "DATA_LOSS"
	CodeDatabaseError      = "DATABASE_ERROR"
	CodeCacheError         = "CACHE_ERROR"
	CodeQueueError         = "QUEUE_ERROR"
	CodeDeadlineExceeded   = "DEADLINE_EXCEEDED"
	CodeUnavailable        = "UNAVAILABLE"
	CodeAborted            = "ABORTED"
	CodeUnimplemented      = "UNIMPLEMENTED"
	CodeOutOfRange         = "OUT_OF_RANGE"
	CodeResourceExhausted  = "RESOURCE_EXHAUSTED"
	CodeFailedPrecondition = "FAILED_PRECONDITION"
	CodeInternalDependency = "INTERNAL_DEPENDENCY_FAILURE"

	// --- Cancellations ---
	CodeCancelled = "CANCELLED"
	CodeDeadlock  = "DEADLOCK_DETECTED"
)

var HTTPStatusMap = map[string]int{
	// --- 2xx ---
	CodeOK:        200,
	CodeCreated:   201,
	CodeAccepted:  202,
	CodeNoContent: 204,

	// --- 3xx ---
	CodeMovedPermanently:  301,
	CodeFound:             302,
	CodeSeeOther:          303,
	CodeNotModified:       304,
	CodeTemporaryRedirect: 307,
	CodePermanentRedirect: 308,

	// --- 4xx ---
	CodeBadRequest:            400,
	CodeInvalidArgument:       400,
	CodeFailedPrecondition:    400,
	CodeUnauthorized:          401,
	CodeUnauthenticated:       401,
	CodeTokenExpired:          401,
	CodeTokenInvalid:          401,
	CodeForbidden:             403,
	CodePermissionDenied:      403,
	CodeNotFound:              404,
	CodeMethodNotAllowed:      405,
	CodeNotAcceptable:         406,
	CodeProxyAuthRequired:     407,
	CodeRequestTimeout:        408,
	CodeConflict:              409,
	CodeAlreadyExists:         409,
	CodeGone:                  410,
	CodeLengthRequired:        411,
	CodePreconditionFailed:    412,
	CodeRequestEntityTooLarge: 413,
	CodeRequestURITooLong:     414,
	CodeUnsupportedMediaType:  415,
	CodeRangeNotSatisfiable:   416,
	CodeExpectationFailed:     417,
	CodeUnprocessableEntity:   422,
	CodeValidationFailed:      422,
	CodeTooManyRequests:       429,
	CodeRateLimitExceeded:     429,
	CodeResourceExhausted:     429,

	// --- 5xx ---
	CodeInternal:           500,
	CodeDatabaseError:      500,
	CodeCacheError:         500,
	CodeQueueError:         500,
	CodeDataLoss:           500,
	CodeNotImplemented:     501,
	CodeUnimplemented:      501,
	CodeBadGateway:         502,
	CodeServiceUnavailable: 503,
	CodeUnavailable:        503,
	CodeInternalDependency: 503,
	CodeGatewayTimeout:     504,
	CodeDeadlineExceeded:   504,
	CodeAborted:            409, // can also be 500 depending on use-case
	CodeCancelled:          499, // client closed connection
	CodeOutOfRange:         400,
}

func HTTPStatus(code string) int {
	if status, ok := HTTPStatusMap[code]; ok {
		return status
	}
	return 500
}
