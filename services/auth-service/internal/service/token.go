package service

import (
	"auth-service/internal/domain"
	"auth-service/internal/error"
	authErrors "auth-service/internal/error"
	"context"
	"shared/pkg/database/postgres"
	"shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/server/common/token"
	"time"
)

// ============================================================================
// Token Refresh
// ============================================================================

func (s *authService) RefreshToken(ctx context.Context, refreshToken string, deviceID string) (*domain.RefreshTokenResult, *error.AuthError) {
	s.log.Info("Refreshing token",
		logger.String("service", authErrors.ServiceName),
	)

	session, sessionErr := s.sessionRepo.GetSessionByRefreshToken(ctx, refreshToken)
	if sessionErr != nil && !postgres.IsNotFoundError(sessionErr) {
		return nil, &error.AuthError{
			Message: "Failed to validate refresh token",
			Code:    authErrors.CodeTokenValidationFailed,
			Error: pkgErrors.FromError(sessionErr, authErrors.CodeTokenValidationFailed, "failed to validate refresh token").
				WithService(authErrors.ServiceName),
		}
	}
	if session == nil {
		s.log.Warn("Refresh token not found in database",
			logger.String("service", authErrors.ServiceName),
		)
		return nil, &error.AuthError{
			Message: "Invalid or expired refresh token",
			Code:    authErrors.CodeInvalidToken,
			Error: pkgErrors.New(authErrors.CodeInvalidToken, "invalid or expired refresh token").
				WithService(authErrors.ServiceName),
		}
	}
	if session.ExpiresAt.Before(time.Now()) {
		s.log.Warn("Refresh token expired",
			logger.String("service", authErrors.ServiceName),
			logger.String("session_id", session.ID),
		)
		return nil, &error.AuthError{
			Message: "Invalid or expired refresh token",
			Code:    authErrors.CodeExpiredSessionToken,
			Error: pkgErrors.New(authErrors.CodeExpiredSessionToken, "invalid or expired refresh token").
				WithService(authErrors.ServiceName).
				WithDetail("session_id", session.ID),
		}
	}
	if session.RevokedAt != nil && session.RevokedAt.Before(time.Now()) {
		s.log.Warn("Refresh token has been invalidated",
			logger.String("service", authErrors.ServiceName),
			logger.String("session_id", session.ID),
		)
		return nil, &error.AuthError{
			Message: "Invalid or expired refresh token",
			Code:    authErrors.CodeSessionRevoked,
			Error: pkgErrors.New(authErrors.CodeSessionRevoked, "invalid or expired refresh token").
				WithService(authErrors.ServiceName).
				WithDetail("reason", session.RevokedReason).
				WithDetail("session_id", session.ID),
		}
	}
	if session.DeviceID != deviceID {
		s.log.Warn("Refresh token used from different device",
			logger.String("service", authErrors.ServiceName),
			logger.String("session_id", session.ID),
			logger.String("expected_device_id", session.DeviceID),
			logger.String("actual_device_id", deviceID),
		)
		return nil, &error.AuthError{
			Message: "Refresh token used from different device",
			Code:    authErrors.CodeSessionDeviceMismatch,
			Error: pkgErrors.New(authErrors.CodeSessionDeviceMismatch, "refresh token used from different device").
				WithService(authErrors.ServiceName).
				WithDetail("session_id", session.ID).
				WithDetail("expected_device_id", session.DeviceID).
				WithDetail("actual_device_id", deviceID),
		}
	}

	userStatus, statusErr := s.sessionCache.GetUserStatus(session.UserID)
	if statusErr != nil {
		return nil, &error.AuthError{
			Message: "Failed to get user account status",
			Code:    authErrors.CodeDatabaseError,
			Error: pkgErrors.FromError(statusErr, authErrors.CodeDatabaseError, "failed to get user account status").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", session.UserID),
		}
	}
	if userStatus != nil && userStatus.AccountStatus != models.AccountStatusActive.String() {
		s.log.Warn("Token refresh attempt for inactive or non-existent user",
			logger.String("service", authErrors.ServiceName),
			logger.String("user_id", session.UserID),
			logger.String("account_status", string(userStatus.AccountStatus)),
		)
		return nil, &error.AuthError{
			Message: "User account is not active",
			Code:    authErrors.CodeUserNotActive,
			Error: pkgErrors.New(authErrors.CodeUserNotActive, "user account is not active").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", session.UserID).
				WithDetail("account_status", string(userStatus.AccountStatus)),
		}
	}

	_, err := s.tokenService.Validate(ctx, refreshToken, token.TokenTypeRefresh)
	if err != nil {
		return nil, &error.AuthError{
			Message: "Invalid or expired refresh token",
			Code:    authErrors.CodeInvalidToken,
			Error: pkgErrors.FromError(err, authErrors.CodeInvalidToken, "invalid or expired refresh token").
				WithService(authErrors.ServiceName),
		}
	}

	accessToken, tokenErr := s.tokenService.IssueAccessToken(ctx, session.UserID, token.IssueOptions{
		ExpiresIn: s.cfg.JWT.AccessTokenTTL,
		Metadata: map[string]interface{}{
			"purpose": "access_token",
			"user_id": session.UserID,
		},
		Audience: []string{s.cfg.JWT.Audience},
	})
	if tokenErr != nil {
		return nil, &error.AuthError{
			Message: "Failed to generate token",
			Code:    authErrors.CodeTokenGenerationFailed,
			Error: pkgErrors.FromError(tokenErr, authErrors.CodeTokenGenerationFailed, "failed to generate access token").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", session.UserID).
				WithDetail("purpose", "access_token"),
		}
	}

	expiresAt := accessToken.Claims.IssuedAt.Add(s.cfg.JWT.AccessTokenTTL)
	newRefreshToken, refreshErr := s.tokenService.IssueRefreshToken(ctx, session.UserID, token.IssueOptions{
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
				WithDetail("user_id", session.UserID).
				WithDetail("purpose", "refresh_token"),
		}
	}

	s.sessionCache.SetRefreshUsed(ctx, refreshToken)
	s.sessionCache.DeleteSession(ctx, session.ID)
	session, err = s.sessionRepo.UpdateSession(ctx, &domain.UpdateAuthSession{
		ID:           session.ID,
		RefreshToken: &newRefreshToken.Token,
	})
	if err != nil {
		return nil, &error.AuthError{
			Message: "Failed to update session with new refresh token",
			Code:    authErrors.CodeDatabaseError,
			Error: pkgErrors.FromError(err, authErrors.CodeDatabaseError, "failed to update session with new refresh token").
				WithService(authErrors.ServiceName).
				WithDetail("session_id", session.ID),
		}
	}

	return &domain.RefreshTokenResult{
		AccessToken:  accessToken.Token,
		RefreshToken: newRefreshToken.Token,
		TokenType:    "Bearer",
		ExpiresAt:    expiresAt,
	}, nil
}
