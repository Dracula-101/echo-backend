package service

import (
	"auth-service/internal/domain"
	"auth-service/internal/error"
	authErrors "auth-service/internal/error"
	"context"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/pkg/utils"
	"time"
)

// ============================================================================
// Email Validation
// ============================================================================

func (s *authService) IsEmailTaken(ctx context.Context, email string) (*domain.User, *error.AuthError) {
	s.log.Info("Checking if email is taken", logger.String("email", email))
	email = utils.NormalizeEmail(email)

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, &error.AuthError{
			Message: "Failed to check email existence",
			Code:    authErrors.CodeDatabaseError,
			Error: pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to check email existence").
				WithService(authErrors.ServiceName).
				WithDetail("email", email),
		}
	}
	return user, nil
}

func (s *authService) SendEmailVerification(ctx context.Context, userID string, email string, ipAddress string, userAgent string) *error.AuthError {
	s.log.Info("Sending email verification", logger.String("user_id", userID), logger.String("email", email))

	token := s.tokenService.GenerateVerificationToken()
	tokenHash, err := s.hashingService.SimpleHash(ctx, token)
	if err != nil {
		return &error.AuthError{
			Message: "Failed to generate email verification token",
			Code:    authErrors.CodeTokenGenerationFailed,
			Error: pkgErrors.FromError(err, authErrors.CodeTokenGenerationFailed, "failed to generate email verification token").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", userID),
		}
	}
	if tokenHash == nil {
		return &error.AuthError{
			Message: "Failed to generate email verification token",
			Code:    authErrors.CodeTokenGenerationFailed,
			Error: pkgErrors.New(authErrors.CodeTokenGenerationFailed, "failed to generate email verification token").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", userID),
		}
	}
	err = s.emailVerificationRepo.CreateVerificationToken(ctx, userID, email, *tokenHash, ipAddress, userAgent, time.Now().Add(s.cfg.EmailVerification.TokenTTL))
	if err != nil {
		return &error.AuthError{
			Message: "Failed to generate email verification token",
			Code:    authErrors.CodeDatabaseError,
			Error: pkgErrors.FromError(err, authErrors.CodeDatabaseError, "failed to generate email verification token").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", userID),
		}
	}

	emailErr := s.emailService.SendEmailVerificationEmail(email, token)
	if emailErr != nil {
		return &error.AuthError{
			Message: "Failed to send email verification",
			Code:    authErrors.CodeEmailSendFailed,
			Error: pkgErrors.FromError(emailErr, authErrors.CodeEmailSendFailed, "failed to send email verification").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", userID),
		}
	}

	s.log.Info("Email verification sent successfully", logger.String("user_id", userID), logger.String("email", email))
	return nil
}
