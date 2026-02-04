package protocol

type ErrorCode string

const (
	ErrCodeUnauthorized       ErrorCode = "unauthorized"
	ErrCodeNotImplemented     ErrorCode = "not_implemented"
	ErrCodeInvalidPayload     ErrorCode = "invalid_payload"
	ErrCodeInvalidJSON        ErrorCode = "invalid_json"
	ErrCodeRateLimited        ErrorCode = "rate_limited"
	ErrCodeSubscriptionDenied ErrorCode = "subscription_denied"
	ErrCodeRoutingFailed      ErrorCode = "routing_failed"
	ErrCodeInvalidMessageType ErrorCode = "invalid_message_type"
	ErrCodeInternalError      ErrorCode = "internal_error"
	ErrCodeConnectionClosed   ErrorCode = "connection_closed"
	ErrCodeMessageTooLarge    ErrorCode = "message_too_large"
)

func isRetryableError(code ErrorCode) bool {
	switch code {
	case ErrCodeRateLimited, ErrCodeInternalError:
		return true
	default:
		return false
	}
}
