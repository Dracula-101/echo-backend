package service

import (
	"context"
	"user-service/internal/model"
	svcmodel "user-service/internal/service/model"
)

// ============================================================================
// Service Interface
// ============================================================================

// UserServiceInterface defines the contract for user service operations
type UserServiceInterface interface {
	// Profile operations
	GetProfile(ctx context.Context, userID string) (*model.User, error)
	CreateProfile(ctx context.Context, profile *svcmodel.Profile) (*model.User, error)
	GenerateUsername(ctx context.Context, displayName string) (string, error)
}

// Compile-time interface compliance check
var _ UserServiceInterface = (*UserService)(nil)
