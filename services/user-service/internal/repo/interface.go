package repository

import (
	"context"

	"user-service/internal/domain"

	pkgErrors "shared/pkg/errors"
)

// ============================================================================
// Repository Interface
// ============================================================================

// UserRepositoryInterface defines the contract for user repository operations
type UserRepository interface {
	// Profile retrieval
	GetProfileByUserID(ctx context.Context, userID string) (*domain.Profile, pkgErrors.AppError)
	GetProfileByUsername(ctx context.Context, username string) (*domain.Profile, pkgErrors.AppError)
	GenerateUniqueUsername(ctx context.Context, baseUsername string) (*string, pkgErrors.AppError)

	// Profile management
	CreateProfile(ctx context.Context, userId string, profile CreateProfileInput) (*domain.Profile, pkgErrors.AppError)
	UpdateProfile(ctx context.Context, userId string, params UpdateProfileParams) (*domain.Profile, pkgErrors.AppError)

	AddUserDevice(ctx context.Context, input *domain.UserDevice, isCurrentDevice bool) pkgErrors.AppError
	GetUserDevices(ctx context.Context, userID string) ([]*domain.UserDevice, pkgErrors.AppError)
	UpdateUserDevice(ctx context.Context, input *domain.UpdateUserDevice, userID string, deviceID string) pkgErrors.AppError

	// Search and validation
	SearchProfiles(ctx context.Context, query string, limit, offset int) ([]*domain.Profile, int, pkgErrors.AppError)
	UsernameExists(ctx context.Context, username string) (bool, pkgErrors.AppError)
}

// Compile-time interface compliance check
var _ UserRepository = (*userRepository)(nil)
