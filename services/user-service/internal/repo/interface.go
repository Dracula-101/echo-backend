package repository

import (
	"context"

	repoModel "user-service/internal/repo/model"
)

// ============================================================================
// Repository Interface
// ============================================================================

// UserRepositoryInterface defines the contract for user repository operations
type UserRepositoryInterface interface {
	// Profile retrieval
	GetProfileByUserID(ctx context.Context, userID string) (*repoModel.Profile, error)
	GetProfileByUsername(ctx context.Context, username string) (*repoModel.Profile, error)
	GenerateUniqueUsername(ctx context.Context, baseUsername string) (*string, error)

	// Profile management
	CreateProfile(ctx context.Context, profile repoModel.Profile) (*repoModel.Profile, error)
	UpdateProfile(ctx context.Context, params UpdateProfileParams) (*repoModel.Profile, error)

	// Search and validation
	SearchProfiles(ctx context.Context, query string, limit, offset int) ([]*repoModel.Profile, int, error)
	UsernameExists(ctx context.Context, username string) (bool, error)
}

// Compile-time interface compliance check
var _ UserRepositoryInterface = (*UserRepository)(nil)
