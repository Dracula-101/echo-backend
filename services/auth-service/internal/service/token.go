package service

import (
	"auth-service/internal/domain"
	"auth-service/internal/error"
	authErrors "auth-service/internal/error"
	"context"
	"shared/pkg/database/postgres"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/server/common/token"
)

// ============================================================================
// Token Refresh
// ============================================================================

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*domain.LoginResult, *error.AuthError) {
	s.log.Info("Refreshing token",
		logger.String("service", authErrors.ServiceName),
	)

	claims, err := s.tokenService.Validate(ctx, refreshToken, token.TokenTypeRefresh)
	if err != nil {
		return nil, &error.AuthError{
			Message: "Invalid or expired refresh token",
			Code:    authErrors.CodeInvalidToken,
			Error: pkgErrors.FromError(err, authErrors.CodeInvalidToken, "invalid or expired refresh token").
				WithService(authErrors.ServiceName),
		}
	}

	userID := claims.Subject
	s.log.Debug("Refresh token validated",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
	)

	user, err2 := s.repo.GetUserByID(ctx, userID)
	if err2 != nil && !postgres.IsNotFoundError(err2) {
		return nil, &error.AuthError{
			Message: "Failed to get user by ID",
			Code:    authErrors.CodeDatabaseError,
			Error: pkgErrors.FromError(err2, pkgErrors.CodeDatabaseError, "failed to get user by ID").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", userID),
		}
	}
	if user == nil {
		s.log.Warn("Token refresh attempt for non-existent user",
			logger.String("service", authErrors.ServiceName),
			logger.String("user_id", userID),
		)
		return nil, &error.AuthError{
			Message: "User not found",
			Code:    authErrors.CodeUserNotFound,
			Error: pkgErrors.New(authErrors.CodeUserNotFound, "user not found").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", userID),
		}
	}

	accessToken, tokenErr := s.tokenService.IssueAccessToken(ctx, user.ID, token.IssueOptions{
		ExpiresIn: s.cfg.JWT.AccessTokenTTL,
		Metadata: map[string]interface{}{
			"purpose": "access_token",
			"user_id": user.ID,
			"email":   user.Email,
		},
		Audience: []string{s.cfg.JWT.Audience},
	})
	if tokenErr != nil {
		return nil, &error.AuthError{
			Message: "Failed to generate token",
			Code:    authErrors.CodeTokenGenerationFailed,
			Error: pkgErrors.FromError(tokenErr, authErrors.CodeTokenGenerationFailed, "failed to generate access token").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", user.ID).
				WithDetail("email", user.Email).
				WithDetail("purpose", "access_token"),
		}
	}

	expiresAt := accessToken.Claims.IssuedAt.Add(s.cfg.JWT.AccessTokenTTL)
	newRefreshToken, refreshErr := s.tokenService.IssueRefreshToken(ctx, user.ID, token.IssueOptions{
		ExpiresIn: s.cfg.JWT.RefreshTokenTTL,
		Metadata: map[string]interface{}{
			"purpose": "refresh_token",
		},
		Audience: []string{s.cfg.JWT.Audience},
	})
	if refreshErr != nil {
		return nil, &error.AuthError{
			Message: "Failed to generate token",
			Code:    authErrors.CodeTokenGenerationFailed,
			Error: pkgErrors.FromError(refreshErr, authErrors.CodeTokenGenerationFailed, "failed to generate refresh token").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", user.ID).
				WithDetail("email", user.Email).
				WithDetail("purpose", "refresh_token"),
		}
	}

	return &domain.LoginResult{
		User: &domain.User{
			ID:                     user.ID,
			Email:                  user.Email,
			PhoneNumber:            user.PhoneNumber,
			PhoneCountryCode:       user.PhoneCountryCode,
			EmailVerified:          user.EmailVerified,
			PhoneVerified:          user.PhoneVerified,
			AccountStatus:          user.AccountStatus,
			TwoFactorEnabled:       user.TwoFactorEnabled,
			PasswordHash:           user.PasswordHash,
			PasswordSalt:           user.PasswordSalt,
			PasswordAlgorithm:      user.PasswordAlgorithm,
			PasswordLastChanged:    user.PasswordLastChanged,
			AccountLockedUntil:     user.AccountLockedUntil,
			FailedLoginAttempts:    user.FailedLoginAttempts,
			RequiresPasswordChange: user.RequiresPasswordChange,
			LastFailedLoginAt:      user.LastFailedLoginAt,
			LastSuccessfulLoginAt:  user.LastSuccessfulLoginAt,
			CreatedAt:              user.CreatedAt,
			UpdatedAt:              user.UpdatedAt,
			DeletedAt:              user.DeletedAt,
		},
		AccessToken:  accessToken.Token,
		RefreshToken: newRefreshToken.Token,
		ExpiresAt:    expiresAt,
	}, nil
}
