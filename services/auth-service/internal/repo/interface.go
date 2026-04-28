package repository

import (
	"auth-service/internal/domain"
	"auth-service/internal/repo/model"
	"context"
	"shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/server/request"
)

// ============================================================================
// Repository Interfaces
// ============================================================================

// AuthRepositoryInterface defines the contract for authentication repository operations
type AuthRepository interface {
	// User management
	ExistsByEmail(ctx context.Context, email string) (*models.AccountStatus, pkgErrors.AppError)
	CreateUser(ctx context.Context, params model.CreateUserParams) (string, pkgErrors.AppError)
	LockUserAccount(ctx context.Context, userID string) pkgErrors.AppError
	UnlockUserAccount(ctx context.Context, userID string) pkgErrors.AppError
	GetUserByEmail(ctx context.Context, email string) (*domain.User, pkgErrors.AppError)
	GetUserByPhone(ctx context.Context, phone string) (*domain.User, pkgErrors.AppError)
	GetUserByID(ctx context.Context, userID string) (*domain.User, pkgErrors.AppError)
	GetLoginAttempts(ctx context.Context, userID string) (int, pkgErrors.AppError)

	// Device management
	GetUserActiveDevice(ctx context.Context, userID string) (*domain.UserDevice, pkgErrors.AppError)
	UpdateUserDevice(ctx context.Context, userID string, deviceID string, update domain.UpdateUserDevice) (*domain.UserDevice, pkgErrors.AppError)
	IsDeviceTrusted(ctx context.Context, userID string, deviceID string) (bool, pkgErrors.AppError)
	IsNewLocation(ctx context.Context, userID string, city string) (bool, pkgErrors.AppError)

	// Password management
	CheckPreviousPasswords(ctx context.Context, userID string, newHash string) (bool, pkgErrors.AppError)
	UpdatePassword(ctx context.Context, userID string, input domain.PasswordHistoryEntry) pkgErrors.AppError

	// Email verification
	MarkEmailVerified(ctx context.Context, userID string, deviceInfo request.DeviceInfo, locationInfo request.IpAddressInfo) pkgErrors.AppError
	ActivatePendingUser(ctx context.Context, userID string) pkgErrors.AppError

	// Phone verification
	CreateOTPVerification(ctx context.Context, userID string, phone string, otp string, otpHash string, ipAddress string, userAgent string) pkgErrors.AppError
	GetOTPVerification(ctx context.Context, userID string) ([]*domain.OTPVerification, pkgErrors.AppError)
	DeleteOTPVerification(ctx context.Context, userID string, phone string, otpCode string) pkgErrors.AppError
	LogOTPAttempt(ctx context.Context, userID string) pkgErrors.AppError
	MarkPhoneVerified(ctx context.Context, userID string) pkgErrors.AppError

	// Two-factor authentication
	Update2FASecret(ctx context.Context, userID string, secret string) pkgErrors.AppError
	Enable2FA(ctx context.Context, userID string) pkgErrors.AppError
	Disable2FA(ctx context.Context, userID string) pkgErrors.AppError

	// Account management
	SoftDeleteUser(ctx context.Context, userID string) pkgErrors.AppError
}
