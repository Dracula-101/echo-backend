package service

import (
	"context"
	"user-service/internal/domain"
)

// ============================================================================
// Service Interface
// ============================================================================

// UserServiceInterface defines the contract for user service operations
type UserServiceInterface interface {
	// Profile operations
	GetProfile(ctx context.Context, userID string) (*domain.User, error)
	CreateProfile(ctx context.Context, profile *domain.CreateProfileInput) (*domain.User, error)
	AddUserDevice(ctx context.Context, input *domain.UserDevice) error
	GenerateUsername(ctx context.Context, displayName string) (string, error)
	AddProfileThumbnail(ctx context.Context, userID string, thumbnailURL string) error
}

// Compile-time interface compliance check
var _ UserServiceInterface = (*UserService)(nil)
