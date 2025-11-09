package repository

import (
	"context"

	"shared/pkg/database/postgres/models"
)

// ============================================================================
// Repository Interface
// ============================================================================

// UserRepositoryInterface defines the contract for user repository operations
type UserRepositoryInterface interface {
	// Profile retrieval
	GetProfileByUserID(ctx context.Context, userID string) (*models.Profile, error)
	GetProfileByUsername(ctx context.Context, username string) (*models.Profile, error)

	// Profile management
	CreateProfile(ctx context.Context, profile models.Profile) (*models.Profile, error)
	UpdateProfile(ctx context.Context, params UpdateProfileParams) (*models.Profile, error)

	// Search and validation
	SearchProfiles(ctx context.Context, query string, limit, offset int) ([]*models.Profile, int, error)
	UsernameExists(ctx context.Context, username string) (bool, error)
}

// Compile-time interface compliance check
var _ UserRepositoryInterface = (*UserRepository)(nil)
