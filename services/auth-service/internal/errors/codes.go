package errors

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
	CodeInvalidEmail       = "AUTH_INVALID_EMAIL"
	CodeInvalidPhoneNumber = "AUTH_INVALID_PHONE"
	CodePasswordTooWeak    = "AUTH_PASSWORD_WEAK"
	CodeTermsNotAccepted   = "AUTH_TERMS_NOT_ACCEPTED"

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
