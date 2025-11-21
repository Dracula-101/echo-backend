package service

import (
	"auth-service/api/v1/dto"
	"auth-service/internal/model"
	serviceModels "auth-service/internal/service/models"
	"context"

	"shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/server/common/hashing"
	"shared/server/common/token"
	"shared/server/request"
)

// ============================================================================
// Service Interfaces
// ============================================================================

// AuthServiceInterface defines the contract for authentication service operations
type AuthServiceInterface interface {
	// Email validation
	IsEmailTaken(ctx context.Context, email string) (bool, pkgErrors.AppError)

	// User operations
	GetUserByEmail(ctx context.Context, email string) (*model.User, pkgErrors.AppError)
	RegisterUser(ctx context.Context, input serviceModels.RegisterUserInput) (*serviceModels.RegisterUserOutput, pkgErrors.AppError)
	Login(ctx context.Context, email, password string) (*dto.LoginResponse, pkgErrors.AppError)

	// Service accessors
	TokenService() token.JWTTokenService
	HashingService() hashing.HashingService
}

// SessionServiceInterface defines the contract for session service operations
type SessionServiceInterface interface {
	// Session management
	CreateSession(ctx context.Context, input serviceModels.CreateSessionInput) (*serviceModels.CreateSessionOutput, pkgErrors.AppError)
	GetSessionByUserId(ctx context.Context, userID string) (*models.AuthSession, pkgErrors.AppError)
	DeleteSessionByID(ctx context.Context, sessionID string) pkgErrors.AppError
}

// LocationServiceInterface defines the contract for location service operations
type LocationServiceInterface interface {
	// IP lookup
	Lookup(ip string) (*request.IpAddressInfo, pkgErrors.AppError)
}

// Compile-time interface compliance checks
var (
	_ AuthServiceInterface     = (*AuthService)(nil)
	_ SessionServiceInterface  = (*SessionService)(nil)
	_ LocationServiceInterface = (*LocationService)(nil)
)
