package service

import (
	"context"
	"user-service/internal/model"
	"user-service/internal/service/models"
)

// ============================================================================
// Service Interface
// ============================================================================

// UserServiceInterface defines the contract for user service operations
type UserServiceInterface interface {
	// Profile operations
	GetProfile(ctx context.Context, userID string) (*model.User, error)
	CreateProfile(ctx context.Context, profile *models.Profile) (*model.User, error)
}

// Compile-time interface compliance check
var _ UserServiceInterface = (*UserService)(nil)
