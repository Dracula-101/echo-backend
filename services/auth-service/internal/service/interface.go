package service

import (
	"auth-service/internal/domain"
	"context"

	"auth-service/internal/error"
	"shared/pkg/email"
	"shared/server/common/hashing"
	"shared/server/common/token"
	"shared/server/request"
)

// AuthServiceInterface defines the contract for authentication service operations
type AuthServiceInterface interface {
	// Email validation
	IsEmailTaken(ctx context.Context, email string) (*domain.User, *error.AuthError)
	// User operations
	GetUserByEmail(ctx context.Context, email string) (*domain.User, *error.AuthError)
	GetUserByID(ctx context.Context, userID string) (*domain.User, *error.AuthError)
	RegisterUser(ctx context.Context, input domain.RegisterUserInput) (*domain.RegisterUserOutput, *error.AuthError)
	Login(ctx context.Context, input domain.LoginInput) (*domain.LoginResult, *error.AuthError)
	RecordFailedLoginAttempt(ctx context.Context, input domain.FailedLoginAttemptInput) *error.AuthError
	RefreshToken(ctx context.Context, refreshToken string) (*domain.LoginResult, *error.AuthError)

	// Session management
	Logout(ctx context.Context, sessionID string, userID string, ipAddress string, userAgent string) *error.AuthError

	// Password management
	ForgotPassword(ctx context.Context, email string, ipAddress string, userAgent string) *error.AuthError
	ResetPassword(ctx context.Context, token string, newPassword string, ipAddress string, userAgent string) *error.AuthError
	ChangePassword(ctx context.Context, userID string, currentPassword string, newPassword string, ipAddress string, userAgent string) *error.AuthError

	// Email verification
	VerifyEmail(ctx context.Context, token string, deviceInfo request.DeviceInfo, locationInfo request.IpAddressInfo) (*string, *error.AuthError)
	SendEmailVerification(ctx context.Context, userID string, email string, ipAddress string, userAgent string) *error.AuthError
	ResendVerification(ctx context.Context, email string, ipAddress string, userAgent string) (*string, *error.AuthError)

	// MFA
	EnableMFA(ctx context.Context, userID string, email string) (*domain.MFAEnableOutput, *error.AuthError)
	DisableMFA(ctx context.Context, userID string, password string) *error.AuthError
	VerifyMFA(ctx context.Context, userID string, code string) (bool, *error.AuthError)

	// Account management
	DeleteAccount(ctx context.Context, userID string, password string, ipAddress string, userAgent string) *error.AuthError

	// Service accessors
	TokenService() token.JWTTokenService
	HashingService() hashing.HashingService
	EmailService() email.EmailService
}

// Compile-time interface compliance checks
var _ AuthServiceInterface = (*AuthService)(nil)
