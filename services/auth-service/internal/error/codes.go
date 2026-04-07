package error

// ============================================================================
// Auth Service Error Codes
// ============================================================================

type AuthErrorCode string

const (
	// Authentication Errors
	CodeInvalidCredentials      = "AUTH_INVALID_CREDENTIALS"
	CodeUserNotFound            = "AUTH_USER_NOT_FOUND"
	CodeEmailAlreadyExists      = "AUTH_EMAIL_EXISTS"
	CodePasswordHashingFailed   = "AUTH_PASSWORD_HASH_FAILED"
	CodeTokenGenerationFailed   = "AUTH_TOKEN_GEN_FAILED"
	CodeInvalidToken            = "AUTH_INVALID_TOKEN"
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

	// Database Errors
	CodeDatabaseError = "AUTH_DATABASE_ERROR"

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

	// Password Reset Errors
	CodePasswordResetTokenExpired = "AUTH_PASSWORD_RESET_TOKEN_EXPIRED"
	CodePasswordResetTokenUsed    = "AUTH_PASSWORD_RESET_TOKEN_USED"
	CodeSamePassword              = "AUTH_SAME_PASSWORD"

	// Email Verification Errors
	CodeEmailAlreadyVerified      = "AUTH_EMAIL_ALREADY_VERIFIED"
	CodeVerificationTokenExpired  = "AUTH_VERIFICATION_TOKEN_EXPIRED"
	CodeVerificationCooldown      = "AUTH_VERIFICATION_COOLDOWN"

	// MFA Errors
	CodeMFAAlreadyEnabled  = "AUTH_MFA_ALREADY_ENABLED"
	CodeMFANotEnabled      = "AUTH_MFA_NOT_ENABLED"

	// Account Errors
	CodeAccountAlreadyDeleted    = "AUTH_ACCOUNT_ALREADY_DELETED"
	CodeSessionOwnershipMismatch = "AUTH_SESSION_OWNERSHIP_MISMATCH"

	// Email Sending Errors
	CodeEmailSendFailed = "AUTH_EMAIL_SEND_FAILED"
)
