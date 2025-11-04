package errors

import pkgErrors "shared/pkg/errors"

// ============================================================================
// Auth Service Error Codes
// ============================================================================

const (
	// Authentication Errors
	CodeInvalidCredentials      = "AUTH_INVALID_CREDENTIALS"
	CodeUserNotFound            = "AUTH_USER_NOT_FOUND"
	CodeEmailAlreadyExists      = "AUTH_EMAIL_EXISTS"
	CodePasswordHashingFailed   = "AUTH_PASSWORD_HASH_FAILED"
	CodeTokenGenerationFailed   = "AUTH_TOKEN_GEN_FAILED"
	CodeTokenValidationFailed   = "AUTH_TOKEN_VALIDATE_FAILED"
	CodeSessionNotFound         = "AUTH_SESSION_NOT_FOUND"
	CodeSessionExpired          = "AUTH_SESSION_EXPIRED"
	CodeAccountLocked           = "AUTH_ACCOUNT_LOCKED"
	CodeAccountDisabled         = "AUTH_ACCOUNT_DISABLED"
	CodePasswordExpired         = "AUTH_PASSWORD_EXPIRED"
	CodeTwoFactorRequired       = "AUTH_2FA_REQUIRED"
	CodeInvalidTwoFactorCode    = "AUTH_INVALID_2FA_CODE"
	CodeEmailVerificationFailed = "AUTH_EMAIL_VERIFY_FAILED"
	CodePhoneVerificationFailed = "AUTH_PHONE_VERIFY_FAILED"

	// Registration Errors
	CodeInvalidEmail        = "AUTH_INVALID_EMAIL"
	CodeInvalidPhoneNumber  = "AUTH_INVALID_PHONE"
	CodePasswordTooWeak     = "AUTH_PASSWORD_WEAK"
	CodeTermsNotAccepted    = "AUTH_TERMS_NOT_ACCEPTED"

	// Session Errors
	CodeSessionCreationFailed = "AUTH_SESSION_CREATE_FAILED"
	CodeSessionUpdateFailed   = "AUTH_SESSION_UPDATE_FAILED"
	CodeInvalidRefreshToken   = "AUTH_INVALID_REFRESH_TOKEN"
	CodeRefreshTokenExpired   = "AUTH_REFRESH_TOKEN_EXPIRED"

	// Security Errors
	CodeTooManyFailedAttempts = "AUTH_TOO_MANY_FAILED_ATTEMPTS"
	CodeSuspiciousActivity    = "AUTH_SUSPICIOUS_ACTIVITY"
	CodeIPBlocked             = "AUTH_IP_BLOCKED"
	CodeDeviceNotTrusted      = "AUTH_DEVICE_NOT_TRUSTED"
)

// ============================================================================
// Service Name
// ============================================================================

const ServiceName = "auth-service"

// ============================================================================
// HTTP Status Code Mapping
// ============================================================================

var HTTPStatusMap = map[string]int{
	// Authentication Errors
	CodeInvalidCredentials:      401,
	CodeUserNotFound:            404,
	CodeEmailAlreadyExists:      409,
	CodePasswordHashingFailed:   500,
	CodeTokenGenerationFailed:   500,
	CodeTokenValidationFailed:   401,
	CodeSessionNotFound:         404,
	CodeSessionExpired:          401,
	CodeAccountLocked:           423,
	CodeAccountDisabled:         403,
	CodePasswordExpired:         401,
	CodeTwoFactorRequired:       401,
	CodeInvalidTwoFactorCode:    401,
	CodeEmailVerificationFailed: 400,
	CodePhoneVerificationFailed: 400,

	// Registration Errors
	CodeInvalidEmail:       400,
	CodeInvalidPhoneNumber: 400,
	CodePasswordTooWeak:    400,
	CodeTermsNotAccepted:   400,

	// Session Errors
	CodeSessionCreationFailed: 500,
	CodeSessionUpdateFailed:   500,
	CodeInvalidRefreshToken:   401,
	CodeRefreshTokenExpired:   401,

	// Security Errors
	CodeTooManyFailedAttempts: 429,
	CodeSuspiciousActivity:    403,
	CodeIPBlocked:             403,
	CodeDeviceNotTrusted:      403,
}

// HTTPStatus returns the HTTP status code for an auth service error code
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

// NewAuthError creates a new auth service error with service context
func NewAuthError(code, message string) pkgErrors.Error {
	return pkgErrors.New(code, message).WithService(ServiceName)
}

// WrapAuthError wraps an error with auth service context
func WrapAuthError(err error, code, message string) pkgErrors.Error {
	if err == nil {
		return nil
	}
	return pkgErrors.Wrap(err, code, message).WithService(ServiceName)
}
