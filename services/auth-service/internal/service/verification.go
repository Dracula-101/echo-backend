package service

import (
	"auth-service/internal/error"
	"context"
	"time"

	"shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/pkg/utils"
)

const (
	maxVerificationAttempts  = 5
	verificationTokenTTL     = 24 * time.Hour
	resendRateLimitWindow    = 1 * time.Hour
	maxResendTokensPerWindow = 3
)

func (s *AuthService) VerifyEmail(ctx context.Context, rawToken string, userID string, ipAddress string, userAgent string) *error.AuthError {
	s.log.Info("Processing email verification",
		logger.String("service", error.ServiceName),
		logger.String("user_id", userID),
	)

	tokenHash, hashErr := s.hashingService.SimpleHash(ctx, rawToken)
	if hashErr != nil {
		s.log.Error("Failed to hash verification token",
			logger.String("service", error.ServiceName),
			logger.String("user_id", userID),
			logger.Error(hashErr),
		)
		return &error.AuthError{
			Code:    error.CodeTokenGenerationFailed,
			Message: "Failed to create verification token",
			Error:   pkgErrors.FromError(hashErr, error.CodeTokenGenerationFailed, "failed to hash verification token"),
		}
	}

	verificationRecord, getErr := s.emailVerificationRepo.GetVerificationTokenByHash(ctx, *tokenHash)
	if getErr != nil {
		s.log.Error("Failed to retrieve verification token",
			logger.String("service", error.ServiceName),
			logger.String("user_id", userID),
			logger.Error(getErr),
		)
		return &error.AuthError{
			Code:    error.CodeVerificationNotFound,
			Message: "Verification token not found",
			Error:   pkgErrors.FromError(getErr, error.CodeVerificationNotFound, "failed to retrieve verification token"),
		}
	}

	if verificationRecord == nil || verificationRecord.UserID != userID {
		s.log.Warn("Invalid verification token",
			logger.String("service", error.ServiceName),
			logger.String("user_id", userID),
		)
		return &error.AuthError{
			Code:    error.CodeInvalidVerificationToken,
			Message: "Invalid verification token",
			Error:   pkgErrors.New(error.CodeInvalidVerificationToken, "token not found or does not belong to user"),
		}
	}

	user, userErr := s.repo.GetUserByID(ctx, userID)
	if userErr != nil {
		s.log.Error("Failed to retrieve user for email verification",
			logger.String("service", error.ServiceName),
			logger.String("user_id", userID),
			logger.Error(userErr),
		)
		return &error.AuthError{
			Code:    error.CodeDatabaseError,
			Message: "Failed to process request",
			Error:   pkgErrors.FromError(userErr, error.CodeDatabaseError, "failed to retrieve user"),
		}
	}

	if user == nil {
		s.log.Warn("User not found for email verification",
			logger.String("service", error.ServiceName),
			logger.String("user_id", userID),
		)
		return &error.AuthError{
			Code:    error.CodeUserNotFound,
			Message: "User not found",
			Error:   pkgErrors.New(error.CodeUserNotFound, "user not found"),
		}
	}

	if user.EmailVerified {
		s.log.Warn("Email already verified",
			logger.String("service", error.ServiceName),
			logger.String("user_id", userID),
		)
		return &error.AuthError{
			Code:    error.CodeEmailAlreadyVerified,
			Message: "Email is already verified",
			Error:   pkgErrors.New(error.CodeEmailAlreadyVerified, "email already verified"),
		}
	}

	if verificationRecord.ExpiresAt.Before(time.Now()) {
		s.log.Warn("Expired verification token",
			logger.String("service", error.ServiceName),
			logger.String("user_id", userID),
		)
		return &error.AuthError{
			Code:    error.CodeVerificationTokenExpired,
			Message: "Verification token has expired",
			Error:   pkgErrors.New(error.CodeVerificationTokenExpired, "verification token expired"),
		}
	}

	if verificationRecord.Attempts >= maxVerificationAttempts {
		s.log.Warn("Too many verification attempts",
			logger.String("service", error.ServiceName),
			logger.String("user_id", userID),
		)
		return &error.AuthError{
			Code:    error.CodeTooManyVerificationAttempts,
			Message: "Too many verification attempts. Please request a new verification email.",
			Error:   pkgErrors.New(error.CodeTooManyVerificationAttempts, "too many verification attempts"),
		}
	}

	// add attempt before updating to prevent
	err := s.emailVerificationRepo.AttemptVerification(ctx, *tokenHash, userID)
	if err != nil {
		s.log.Warn("Failed to record verification attempt (proceeding anyway)",
			logger.String("service", error.ServiceName),
			logger.String("user_id", userID),
			logger.Error(err),
		)
		return &error.AuthError{
			Code:    error.CodeDatabaseError,
			Message: "Failed to process verification attempt",
			Error:   pkgErrors.FromError(err, error.CodeDatabaseError, "failed to record verification attempt"),
		}
	}

	updateErr := s.repo.MarkEmailVerified(ctx, userID, ipAddress, userAgent)
	if updateErr != nil {
		s.log.Error("Failed to mark user email as verified",
			logger.String("service", error.ServiceName),
			logger.String("user_id", userID),
			logger.Error(updateErr),
		)
		return &error.AuthError{
			Code:    error.CodeDatabaseError,
			Message: "Failed to process verification attempt",
			Error:   pkgErrors.FromError(updateErr, error.CodeDatabaseError, "failed to mark user email as verified"),
		}
	}

	s.log.Info("Email verified successfully",
		logger.String("service", error.ServiceName),
		logger.String("user_id", userID),
	)

	return nil
}

func (s *AuthService) ResendVerification(ctx context.Context, email string, ipAddress string, userAgent string) (*string, *error.AuthError) {
	s.log.Info("Processing resend verification email",
		logger.String("service", error.ServiceName),
		logger.String("email", email),
	)

	email = utils.NormalizeEmail(email)

	// ── 1. Look up user ──────────────────────────────────────────────────
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		s.log.Error("Failed to look up user for resend verification",
			logger.String("service", error.ServiceName),
			logger.String("email", email),
			logger.Error(err),
		)
		return nil, &error.AuthError{
			Code:    error.CodeDatabaseError,
			Message: "Failed to process request",
			Error:   err,
		}
	}

	// ── 2. Reject if ineligible ──────────────────────────────────────────
	if user == nil {
		return nil, &error.AuthError{
			Code:    error.CodeUserNotFound,
			Message: "No account found with this email",
			Error:   pkgErrors.New(error.CodeUserNotFound, "user not found"),
		}
	}
	if user.EmailVerified {
		return nil, &error.AuthError{
			Code:    error.CodeEmailAlreadyVerified,
			Message: "Email is already verified",
			Error:   pkgErrors.New(error.CodeEmailAlreadyVerified, "email already verified"),
		}
	}
	if user.AccountStatus == models.AccountStatusDeleted ||
		user.AccountStatus == models.AccountStatusSuspended ||
		user.AccountStatus == models.AccountStatusDeactivated {
		return nil, &error.AuthError{
			Code:    error.CodeUserNotFound,
			Message: "No account found with this email",
			Error:   pkgErrors.New(error.CodeUserNotFound, "account ineligible"),
		}
	}

	// ── 3. Rate-limit: count recent tokens ───────────────────────────────
	since := time.Now().Add(-resendRateLimitWindow)
	count, countErr := s.emailVerificationRepo.CountRecentTokensByUserID(ctx, user.ID, since)
	if countErr != nil {
		s.log.Warn("Failed to count recent verification tokens (proceeding anyway)",
			logger.String("service", error.ServiceName),
			logger.String("user_id", user.ID),
			logger.Error(countErr),
		)
	}
	if count >= maxResendTokensPerWindow {
		s.log.Warn("Resend verification rate limit hit",
			logger.String("service", error.ServiceName),
			logger.String("user_id", user.ID),
			logger.Int("recent_tokens", count),
		)
		return nil, &error.AuthError{
			Code:    error.CodeVerificationEmailRecentlySent,
			Message: "Verification email was recently sent. Please check your inbox or try again later.",
			Error:   pkgErrors.New(error.CodeVerificationEmailRecentlySent, "rate limit exceeded"),
		}
	}

	// ── 4. Invalidate old unverified tokens ──────────────────────────────
	invalidateErr := s.emailVerificationRepo.InvalidateUserTokens(ctx, user.ID)
	if invalidateErr != nil {
		s.log.Warn("Failed to invalidate old verification tokens (non-critical)",
			logger.String("service", error.ServiceName),
			logger.String("user_id", user.ID),
			logger.Error(invalidateErr),
		)
	}

	// ── 5. Generate new token ────────────────────────────────────────────
	rawToken := s.tokenService.GenerateVerificationToken()
	tokenHash, hashErr := s.hashingService.SimpleHash(ctx, rawToken)
	if hashErr != nil {
		s.log.Error("Failed to hash verification token",
			logger.String("service", error.ServiceName),
			logger.String("user_id", user.ID),
			logger.Error(hashErr),
		)
		return nil, &error.AuthError{
			Code:    error.CodeTokenGenerationFailed,
			Message: "Failed to create verification token",
			Error:   pkgErrors.FromError(hashErr, error.CodeTokenGenerationFailed, "failed to hash verification token"),
		}
	}

	expiresAt := time.Now().Add(verificationTokenTTL)

	createErr := s.emailVerificationRepo.CreateVerificationToken(
		ctx, user.ID, email, *tokenHash, ipAddress, userAgent, expiresAt,
	)
	if createErr != nil {
		s.log.Error("Failed to create verification token",
			logger.String("service", error.ServiceName),
			logger.String("user_id", user.ID),
			logger.Error(createErr),
		)
		return nil, &error.AuthError{
			Code:    error.CodeTokenGenerationFailed,
			Message: "Failed to create verification token",
			Error:   createErr,
		}
	}

	// ── 6. Send the email (only after DB insert succeeds) ────────────────
	emailErr := s.emailService.SendEmailVerificationEmail(email, rawToken)
	if emailErr != nil {
		s.log.Error("Failed to send verification email",
			logger.String("service", error.ServiceName),
			logger.String("email", email),
			logger.Error(emailErr),
		)
		return nil, &error.AuthError{
			Code:    error.CodeEmailSendFailed,
			Message: "Failed to send verification email. Please try again.",
			Error:   pkgErrors.FromError(emailErr, error.CodeEmailSendFailed, "failed to send verification email"),
		}
	}

	s.log.Info("Verification email resent successfully",
		logger.String("service", error.ServiceName),
		logger.String("email", email),
		logger.String("user_id", user.ID),
	)

	return &rawToken, nil
}
