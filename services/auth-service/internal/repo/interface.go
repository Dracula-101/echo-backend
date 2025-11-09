package repository

import (
	repoModels "auth-service/internal/repo/models"
	"context"

	"shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
)

// ============================================================================
// Repository Interfaces
// ============================================================================

// AuthRepositoryInterface defines the contract for authentication repository operations
type AuthRepositoryInterface interface {
	// User management
	ExistsByEmail(ctx context.Context, email string) (bool, pkgErrors.AppError)
	CreateUser(ctx context.Context, params CreateUserParams) (string, pkgErrors.AppError)
	GetUserByEmail(ctx context.Context, email string) (*models.AuthUser, pkgErrors.AppError)
	RecordFailedLogin(ctx context.Context, userID string) pkgErrors.AppError
	RecordSuccessfulLogin(ctx context.Context, userID string) pkgErrors.AppError
}

// LoginHistoryRepositoryInterface defines the contract for login history repository operations
type LoginHistoryRepositoryInterface interface {
	// Login history management
	CreateLoginHistory(ctx context.Context, input repoModels.CreateLoginHistoryInput) pkgErrors.AppError
	GetLoginHistoryByUserID(ctx context.Context, userID string, limit int) ([]*models.LoginHistory, error)
	GetLoginHistoryByID(ctx context.Context, id string) (*models.LoginHistory, error)
	GetFailedLoginAttempts(ctx context.Context, userID string, duration string) (int, error)
	DeleteLoginHistoryByUserID(ctx context.Context, userID string) pkgErrors.AppError
	DeleteLoginHistoryByID(ctx context.Context, id string) pkgErrors.AppError
}

// SessionRepositoryInterface defines the contract for session repository operations
type SessionRepositoryInterface interface {
	// Session management
	CreateSession(ctx context.Context, session *models.AuthSession) pkgErrors.AppError
	GetSessionByUserId(ctx context.Context, userID string) (*models.AuthSession, error)
	DeleteSessionByID(ctx context.Context, sessionID string) pkgErrors.AppError
}

// SecurityEventRepositoryInterface defines the contract for security event repository operations
type SecurityEventRepositoryInterface interface {
	// Security event management
	LogSecurityEvent(ctx context.Context, event *models.SecurityEvent) pkgErrors.AppError
	GetSecurityEventsByUserID(ctx context.Context, userID string, limit int) ([]*models.SecurityEvent, error)
	GetSecurityEventByID(ctx context.Context, id string) (*models.SecurityEvent, error)
	GetSuspiciousEvents(ctx context.Context, userID string, limit int) ([]*models.SecurityEvent, error)
	GetEventsByType(ctx context.Context, userID string, eventType string, limit int) ([]*models.SecurityEvent, error)
	CountEventsBySeverity(ctx context.Context, userID string, severity string, duration string) (int, error)
	DeleteSecurityEventsByUserID(ctx context.Context, userID string) pkgErrors.AppError
	DeleteSecurityEventByID(ctx context.Context, id string) pkgErrors.AppError
}

// Compile-time interface compliance checks
var (
	_ AuthRepositoryInterface          = (*AuthRepository)(nil)
	_ LoginHistoryRepositoryInterface  = (*LoginHistoryRepo)(nil)
	_ SessionRepositoryInterface       = (*SessionRepo)(nil)
	_ SecurityEventRepositoryInterface = (*SecurityEventRepo)(nil)
)
