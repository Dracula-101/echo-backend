package service

import (
	"context"
	"user-service/internal/domain"

	pkgErrors "shared/pkg/errors"
)

// ============================================================================
// Service Interface
// ============================================================================

// UserServiceInterface defines the contract for user service operations
type UserService interface {
	// Profile operations
	GetProfile(ctx context.Context, userID string) (*domain.Profile, pkgErrors.AppError)
	GetProfileByUsername(ctx context.Context, currentUserID string, username string) (*domain.Profile, pkgErrors.AppError)
	CreateProfile(ctx context.Context, userID string, profile *domain.CreateProfileInput) (*domain.Profile, pkgErrors.AppError)
	UpdateProfile(ctx context.Context, userID string, input *domain.UpdateProfileInput) (*domain.Profile, pkgErrors.AppError)
	AddUserDevice(ctx context.Context, input *domain.UserDevice) pkgErrors.AppError
	GenerateUsername(ctx context.Context, displayName string) (string, pkgErrors.AppError)
	AddProfileThumbnail(ctx context.Context, userID string, thumbnailURL string) pkgErrors.AppError
}

// Compile-time interface compliance check
var _ UserService = (*userService)(nil)
