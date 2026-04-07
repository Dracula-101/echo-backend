package error

import (
	"errors"
	pkgErrors "shared/pkg/errors"
)

type AuthError struct {
	Message string
	Code    AuthErrorCode
	Error   pkgErrors.AppError
}

const ServiceName = "auth-service"

var (
	// User account errors
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailAlreadyExists = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")

	// Account status errors
	ErrAccountLocked        = errors.New("account is locked due to multiple failed login attempts")
	ErrAccountDeactivated   = errors.New("account is deactivated")
	ErrAccountSuspended     = errors.New("account is suspended")
	ErrAccountPending       = errors.New("account is pending verification")
	ErrAccountDeleted       = errors.New("account has been deleted")
	ErrUnknownAccountStatus = errors.New("unknown account status")

	// Password errors
	ErrPasswordTooWeak      = errors.New("password does not meet security requirements")
	ErrPasswordHashFailed   = errors.New("failed to hash password")
	ErrPasswordVerifyFailed = errors.New("failed to verify password")

	// Token errors
	ErrTokenGenerationFailed = errors.New("failed to generate authentication token")
	ErrInvalidToken          = errors.New("invalid or expired token")

	// Session errors
	ErrSessionNotFound       = errors.New("session not found")
	ErrSessionExpired        = errors.New("session has expired")
	ErrSessionCreationFailed = errors.New("failed to create session")

	// Validation errors
	ErrInvalidEmail = errors.New("invalid email format")
	ErrInvalidInput = errors.New("invalid input parameters")

	// Password reset errors
	ErrPasswordResetTokenExpired = errors.New("password reset token has expired")
	ErrPasswordResetTokenUsed    = errors.New("password reset token has already been used")
	ErrSamePassword              = errors.New("new password must be different from the current password")

	// Email verification errors
	ErrEmailAlreadyVerified     = errors.New("email is already verified")
	ErrVerificationTokenExpired = errors.New("verification token has expired")
	ErrVerificationCooldown     = errors.New("please wait before requesting another verification email")

	// MFA errors
	ErrMFAAlreadyEnabled = errors.New("MFA is already enabled")
	ErrMFANotEnabled     = errors.New("MFA is not enabled on this account")

	// Account errors
	ErrAccountAlreadyDeleted    = errors.New("account has already been deleted")
	ErrSessionOwnershipMismatch = errors.New("session does not belong to this user")
)
