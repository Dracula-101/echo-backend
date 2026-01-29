package repository

import (
	"context"

	"user-service/internal/domain"
)

// ============================================================================
// Repository Interface
// ============================================================================

// UserRepositoryInterface defines the contract for user repository operations
type UserRepositoryInterface interface {
	// Profile retrieval
	GetProfileByUserID(ctx context.Context, userID string) (*domain.Profile, error)
	GetProfileByUsername(ctx context.Context, username string) (*domain.Profile, error)
	GenerateUniqueUsername(ctx context.Context, baseUsername string) (*string, error)

	// Profile management
	CreateProfile(ctx context.Context, profile domain.Profile) (*domain.Profile, error)
	UpdateProfile(ctx context.Context, params UpdateProfileParams) (*domain.Profile, error)

	AddUserDevice(ctx context.Context, input *domain.UserDevice, isCurrentDevice bool) error
	GetUserDevices(ctx context.Context, userID string) ([]*domain.UserDevice, error)
	UpdateUserDevice(ctx context.Context, input *domain.UpdateUserDevice, userID string, deviceID string) error

	// Search and validation
	SearchProfiles(ctx context.Context, query string, limit, offset int) ([]*domain.Profile, int, error)
	UsernameExists(ctx context.Context, username string) (bool, error)
}

// Compile-time interface compliance check
var _ UserRepositoryInterface = (*UserRepository)(nil)
