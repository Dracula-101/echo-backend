package service

import (
	"auth-service/internal/domain"
	"auth-service/internal/error"
	authErrors "auth-service/internal/error"
	"context"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
)

// ============================================================================
// User Retrieval
// ============================================================================

func (s *authService) GetUserByEmail(ctx context.Context, email string) (*domain.User, *error.AuthError) {
	s.log.Info("Fetching user by email",
		logger.String("service", authErrors.ServiceName),
		logger.String("email", email),
	)

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, &error.AuthError{
			Message: "Failed to get user by email",
			Code:    authErrors.CodeDatabaseError,
			Error: pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get user by email").
				WithService(authErrors.ServiceName).
				WithDetail("email", email),
		}
	}
	if user == nil {
		s.log.Info("User not found",
			logger.String("service", authErrors.ServiceName),
			logger.String("email", email),
		)
		return nil, nil
	}
	s.log.Debug("User fetched successfully",
		logger.String("service", authErrors.ServiceName),
		logger.String("email", email),
		logger.String("user_id", user.ID),
	)

	return &domain.User{
		ID:                     user.ID,
		Email:                  user.Email,
		PhoneNumber:            user.PhoneNumber,
		PhoneCountryCode:       user.PhoneCountryCode,
		EmailVerified:          user.EmailVerified,
		PhoneVerified:          user.PhoneVerified,
		AccountStatus:          user.AccountStatus,
		TwoFactorEnabled:       user.TwoFactorEnabled,
		PasswordHash:           user.PasswordHash,
		PasswordLastChanged:    user.PasswordLastChanged,
		AccountLockedUntil:     user.AccountLockedUntil,
		FailedLoginAttempts:    user.FailedLoginAttempts,
		RequiresPasswordChange: user.RequiresPasswordChange,
		CreatedAt:              user.CreatedAt,
		DeletedAt:              user.DeletedAt,
		UpdatedAt:              user.UpdatedAt,
	}, nil
}

func (s *authService) GetUserByID(ctx context.Context, userID string) (*domain.User, *error.AuthError) {
	s.log.Info("Fetching user by ID",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
	)

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, &error.AuthError{
			Message: "Failed to get user by ID",
			Code:    authErrors.CodeDatabaseError,
			Error: pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get user by ID").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", userID),
		}
	}
	if user == nil {
		s.log.Info("User not found",
			logger.String("service", authErrors.ServiceName),
			logger.String("user_id", userID),
		)
		return nil, nil
	}
	s.log.Debug("User fetched successfully",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", user.ID),
	)

	return &domain.User{
		ID:                     user.ID,
		Email:                  user.Email,
		PhoneNumber:            user.PhoneNumber,
		PhoneCountryCode:       user.PhoneCountryCode,
		EmailVerified:          user.EmailVerified,
		PhoneVerified:          user.PhoneVerified,
		AccountStatus:          user.AccountStatus,
		TwoFactorEnabled:       user.TwoFactorEnabled,
		PasswordHash:           user.PasswordHash,
		PasswordLastChanged:    user.PasswordLastChanged,
		AccountLockedUntil:     user.AccountLockedUntil,
		FailedLoginAttempts:    user.FailedLoginAttempts,
		RequiresPasswordChange: user.RequiresPasswordChange,
		CreatedAt:              user.CreatedAt,
		DeletedAt:              user.DeletedAt,
		UpdatedAt:              user.UpdatedAt,
	}, nil
}
