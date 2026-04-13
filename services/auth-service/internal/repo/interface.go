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
type AuthRepositoryInterface interface {
	// User management
	ExistsByEmail(ctx context.Context, email string) (*models.AccountStatus, pkgErrors.AppError)
	CreateUser(ctx context.Context, params model.CreateUserParams) (string, pkgErrors.AppError)
	UnlockUserAccount(ctx context.Context, userID string) pkgErrors.AppError
	GetUserByEmail(ctx context.Context, email string) (*domain.User, pkgErrors.AppError)
	GetUserByID(ctx context.Context, userID string) (*domain.User, pkgErrors.AppError)
	RecordFailedLogin(ctx context.Context, userID string) pkgErrors.AppError
	RecordSuccessfulLogin(ctx context.Context, userID string) pkgErrors.AppError

	// Password management
	UpdatePassword(ctx context.Context, userID string, hash string, salt string, algorithm string) pkgErrors.AppError

	// Email verification
	MarkEmailVerified(ctx context.Context, userID string, deviceInfo request.DeviceInfo, locationInfo request.IpAddressInfo) pkgErrors.AppError
	ActivatePendingUser(ctx context.Context, userID string) pkgErrors.AppError

	// Two-factor authentication
	Update2FASecret(ctx context.Context, userID string, secret string) pkgErrors.AppError
	Enable2FA(ctx context.Context, userID string) pkgErrors.AppError
	Disable2FA(ctx context.Context, userID string) pkgErrors.AppError

	// Account management
	SoftDeleteUser(ctx context.Context, userID string) pkgErrors.AppError
}

// Compile-time interface compliance checks
var _ AuthRepositoryInterface = (*AuthRepository)(nil)
