package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"time"

	authErrors "auth-service/internal/error"

	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
)

func (s *AuthService) ForgotPassword(ctx context.Context, email string, ipAddress string, userAgent string) *authErrors.AuthError {
	s.log.Info("Processing forgot password request",
		logger.String("service", authErrors.ServiceName),
		logger.String("email", email),
	)
	email = normalizeEmail(email)

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		s.log.Error("Failed to get user by email",
			logger.String("service", authErrors.ServiceName),
			logger.Error(err),
		)
		// Always return nil to prevent email enumeration
		return nil
	}
	if user == nil {
		// Don't reveal whether user exists
		s.log.Debug("Forgot password for non-existent user",
			logger.String("service", authErrors.ServiceName),
			logger.String("email", email),
		)
		return nil
	}

	// Generate random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return &authErrors.AuthError{
			Message: "Failed to generate reset token",
			Code:    authErrors.CodeTokenGenerationFailed,
			Error: pkgErrors.FromError(err, authErrors.CodeTokenGenerationFailed, "failed to generate random token").
				WithService(authErrors.ServiceName),
		}
	}
	rawToken := base64.RawURLEncoding.EncodeToString(tokenBytes)
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := base64.RawURLEncoding.EncodeToString(hash[:])

	expiresAt := time.Now().Add(1 * time.Hour)

	// Invalidate existing tokens
	_ = s.passwordResetRepo.InvalidateUserTokens(ctx, user.ID)
	dbErr := s.passwordResetRepo.CreateResetToken(ctx, user.ID, tokenHash, ipAddress, userAgent, expiresAt)
	if dbErr != nil {
		s.log.Error("Failed to create password reset token",
			logger.String("service", authErrors.ServiceName),
			logger.Error(dbErr),
		)
		// Still return nil to prevent information leakage
		return nil
	}

	// Send email with reset instructions
	if s.emailService != nil {
		emailErr := s.emailService.SendPasswordResetEmail(email, user.ID, rawToken)
		if emailErr != nil {
			s.log.Error("Failed to send password reset email",
				logger.String("service", authErrors.ServiceName),
				logger.Error(emailErr),
			)
			return &authErrors.AuthError{
				Message: "Failed to send password reset email",
				Code:    authErrors.CodeEmailSendFailed,
				Error: pkgErrors.FromError(emailErr, authErrors.CodeEmailSendFailed, "failed to send password reset email").
					WithService(authErrors.ServiceName).
					WithDetail("email", email),
			}
		} else {
			s.log.Info("Password reset email sent",
				logger.String("service", authErrors.ServiceName),
				logger.String("email", email),
				logger.String("user_id", user.ID),
				logger.String("ip_address", ipAddress),
				logger.String("user_agent", userAgent),
				logger.String("expires_at", expiresAt.Format(time.RFC3339)),
			)
		}
	}

	s.log.Info("Password reset token created",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", user.ID),
	)
	return nil
}

func (s *AuthService) ResetPassword(ctx context.Context, token string, newPassword string) *authErrors.AuthError {
	s.log.Info("Processing password reset",
		logger.String("service", authErrors.ServiceName),
	)

	hash := sha256.Sum256([]byte(token))
	tokenHash := base64.RawURLEncoding.EncodeToString(hash[:])

	resetToken, dbErr := s.passwordResetRepo.GetResetTokenByHash(ctx, tokenHash)
	if dbErr != nil {
		return &authErrors.AuthError{
			Message: "Failed to validate reset token",
			Code:    authErrors.CodeDatabaseError,
			Error: pkgErrors.FromError(dbErr, pkgErrors.CodeDatabaseError, "failed to get reset token").
				WithService(authErrors.ServiceName),
		}
	}
	if resetToken == nil {
		return &authErrors.AuthError{
			Message: "Invalid or expired reset token",
			Code:    authErrors.CodeInvalidToken,
			Error: pkgErrors.New(authErrors.CodeInvalidToken, "reset token not found").
				WithService(authErrors.ServiceName),
		}
	}

	if resetToken.UsedAt != nil {
		return &authErrors.AuthError{
			Message: "Reset token has already been used",
			Code:    authErrors.CodePasswordResetTokenUsed,
			Error: pkgErrors.New(authErrors.CodePasswordResetTokenUsed, "token already used").
				WithService(authErrors.ServiceName),
		}
	}

	if time.Now().After(resetToken.ExpiresAt) {
		return &authErrors.AuthError{
			Message: "Reset token has expired",
			Code:    authErrors.CodePasswordResetTokenExpired,
			Error: pkgErrors.New(authErrors.CodePasswordResetTokenExpired, "token expired").
				WithService(authErrors.ServiceName),
		}
	}

	// Hash the new password
	result, hashErr := s.hashingService.HashPassword(ctx, newPassword)
	if hashErr != nil {
		return &authErrors.AuthError{
			Message: "Failed to hash password",
			Code:    authErrors.CodePasswordHashingFailed,
			Error: pkgErrors.FromError(hashErr, authErrors.CodePasswordHashingFailed, "failed to hash password").
				WithService(authErrors.ServiceName),
		}
	}

	salt := base64.StdEncoding.EncodeToString(result.Salt)
	updateErr := s.repo.UpdatePassword(ctx, resetToken.UserID, result.Encoded, salt, string(result.Algorithm))
	if updateErr != nil {
		return &authErrors.AuthError{
			Message: "Failed to update password",
			Code:    authErrors.CodeDatabaseError,
			Error: pkgErrors.FromError(updateErr, pkgErrors.CodeDatabaseError, "failed to update password").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", resetToken.UserID),
		}
	}

	// Mark token as used
	_ = s.passwordResetRepo.MarkTokenUsed(ctx, resetToken.ID)

	// Revoke all sessions for security
	if s.sessionRepo != nil {
		_ = s.sessionRepo.RevokeAllUserSessions(ctx, resetToken.UserID, "password_reset")
	}

	s.log.Info("Password reset successful",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", resetToken.UserID),
	)
	return nil
}

func (s *AuthService) ChangePassword(ctx context.Context, userID string, currentPassword string, newPassword string) *authErrors.AuthError {
	s.log.Info("Processing password change",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
	)

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return &authErrors.AuthError{
			Message: "Failed to get user",
			Code:    authErrors.CodeDatabaseError,
			Error: pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get user").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", userID),
		}
	}
	if user == nil {
		return &authErrors.AuthError{
			Message: "User not found",
			Code:    authErrors.CodeUserNotFound,
			Error: pkgErrors.New(authErrors.CodeUserNotFound, "user not found").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", userID),
		}
	}

	// Verify current password
	success, _, verifyErr := s.hashingService.VerifyPassword(ctx, currentPassword, user.PasswordHash)
	if verifyErr != nil {
		return &authErrors.AuthError{
			Message: "Password verification failed",
			Code:    authErrors.CodeInvalidCredentials,
			Error: pkgErrors.FromError(verifyErr, authErrors.CodeInvalidCredentials, "password verification failed").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", userID),
		}
	}
	if !success {
		return &authErrors.AuthError{
			Message: "Current password is incorrect",
			Code:    authErrors.CodeInvalidCredentials,
			Error: pkgErrors.New(authErrors.CodeInvalidCredentials, "invalid current password").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", userID),
		}
	}

	// Check that new password is different
	sameCheck, _, _ := s.hashingService.VerifyPassword(ctx, newPassword, user.PasswordHash)
	if sameCheck {
		return &authErrors.AuthError{
			Message: "New password must be different from the current password",
			Code:    authErrors.CodeSamePassword,
			Error: pkgErrors.New(authErrors.CodeSamePassword, "same password").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", userID),
		}
	}

	// Hash the new password
	result, hashErr := s.hashingService.HashPassword(ctx, newPassword)
	if hashErr != nil {
		return &authErrors.AuthError{
			Message: "Failed to hash password",
			Code:    authErrors.CodePasswordHashingFailed,
			Error: pkgErrors.FromError(hashErr, authErrors.CodePasswordHashingFailed, "failed to hash password").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", userID),
		}
	}

	salt := base64.StdEncoding.EncodeToString(result.Salt)
	updateErr := s.repo.UpdatePassword(ctx, userID, result.Encoded, salt, string(result.Algorithm))
	if updateErr != nil {
		return &authErrors.AuthError{
			Message: "Failed to update password",
			Code:    authErrors.CodeDatabaseError,
			Error: pkgErrors.FromError(updateErr, pkgErrors.CodeDatabaseError, "failed to update password").
				WithService(authErrors.ServiceName).
				WithDetail("user_id", userID),
		}
	}

	s.log.Info("Password changed successfully",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
	)
	return nil
}
