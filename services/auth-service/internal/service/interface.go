package service

import (
	"auth-service/api/dto"
	"auth-service/internal/model"
	serviceModels "auth-service/internal/service/models"
	"context"

	"shared/pkg/database/postgres/models"
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
	IsEmailTaken(ctx context.Context, email string) (bool, error)

	// User operations
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	RegisterUser(ctx context.Context, input serviceModels.RegisterUserInput) (*serviceModels.RegisterUserOutput, error)
	Login(ctx context.Context, email, password string) (*dto.LoginResponse, error)

	// Service accessors
	TokenService() token.JWTTokenService
	HashingService() hashing.HashingService
}

// SessionServiceInterface defines the contract for session service operations
type SessionServiceInterface interface {
	// Session management
	CreateSession(ctx context.Context, input serviceModels.CreateSessionInput) (*serviceModels.CreateSessionOutput, error)
	GetSessionByUserId(ctx context.Context, userID string) (*models.AuthSession, error)
	DeleteSessionByID(ctx context.Context, sessionID string) error
	GenerateDeviceFingerprint(deviceID string, deviceOS string, deviceName string) string
}

// LocationServiceInterface defines the contract for location service operations
type LocationServiceInterface interface {
	// IP lookup
	Lookup(ip string) (*request.IpAddressInfo, error)
}

// Compile-time interface compliance checks
var (
	_ AuthServiceInterface     = (*AuthService)(nil)
	_ SessionServiceInterface  = (*SessionService)(nil)
	_ LocationServiceInterface = (*LocationService)(nil)
)
