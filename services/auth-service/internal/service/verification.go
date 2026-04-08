package service

import (
	"auth-service/internal/error"
	repository "auth-service/internal/repo"
	"context"
	"time"

	"shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
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

	// ── 1. Fetch the latest token for this user ──────────────────────────
	latestToken, dbErr := s.emailVerificationRepo.GetLatestTokenByUserID(ctx, userID)
	if dbErr != nil {
		return &error.AuthError{
			Code:    error.CodeDatabaseError,
			Message: "Failed to look up verification token",
			Error:   dbErr,
		}
	}
	if latestToken == nil {
		s.log.Warn("No verification token found for user",
			logger.String("service", error.ServiceName),
			logger.String("user_id", userID),
		)
		return &error.AuthError{
			Code:    error.CodeVerificationNotFound,
			Message: "Invalid or expired verification token",
			Error:   pkgErrors.New(error.CodeVerificationNotFound, "no verification token found for user"),
		}
	}

	// ── 2. Already verified ──────────────────────────────────────────────
	if latestToken.VerifiedAt != nil {
		return &error.AuthError{
			Code:    error.CodeEmailAlreadyVerified,
			Message: "Email is already verified",
			Error:   pkgErrors.New(error.CodeEmailAlreadyVerified, "email already verified"),
		}
	}

	// ── 3. Expired ───────────────────────────────────────────────────────
	if latestToken.ExpiresAt.Before(time.Now()) {
		return &error.AuthError{
			Code:    error.CodeVerificationTokenExpired,
			Message: "Verification token has expired. Please request a new one.",
			Error:   pkgErrors.New(error.CodeVerificationTokenExpired, "token expired"),
		}
	}

	// ── 4. Too many wrong attempts ───────────────────────────────────────
	if latestToken.Attempts >= maxVerificationAttempts {
		s.log.Warn("Verification attempt threshold exceeded",
			logger.String("service", error.ServiceName),
			logger.String("user_id", userID),
			logger.Int("attempts", latestToken.Attempts),
		)
		return &error.AuthError{
			Code:    error.CodeTooManyVerificationAttempts,
			Message: "Too many failed attempts. Please request a new verification email.",
			Error:   pkgErrors.New(error.CodeTooManyVerificationAttempts, "max attempts exceeded"),
		}
	}

	// ── 5. Hash mismatch → delegate to repo (increments attempts) ────────
	if latestToken.TokenHash != *tokenHash {
		_ = s.emailVerificationRepo.MarkTokenVerified(ctx, *tokenHash, userID)
		return &error.AuthError{
			Code:    error.CodeVerificationInvalid,
			Message: "Invalid verification token",
			Error:   pkgErrors.New(error.CodeVerificationInvalid, "token hash mismatch"),
		}
	}

	// ── 6. Mark the token as verified ────────────────────────────────────
	markErr := s.emailVerificationRepo.MarkTokenVerified(ctx, *tokenHash, userID)
	if markErr != nil {
		return &error.AuthError{
			Code:    error.CodeEmailVerificationFailed,
			Message: "Failed to verify email",
			Error:   markErr,
		}
	}

	// ── 7. Mark the user email as verified ───────────────────────────────
	verifyErr := s.repo.MarkEmailVerified(ctx, userID)
	if verifyErr != nil {
		return &error.AuthError{
			Code:    error.CodeDatabaseError,
			Message: "Failed to update user verification status",
			Error:   verifyErr,
		}
	}

	// ── 8. If account is pending, flip to active ─────────────────────────
	user, getUserErr := s.repo.GetUserByID(ctx, userID)
	if getUserErr == nil && user != nil && user.AccountStatus == models.AccountStatusPending {
		s.log.Info("Activating pending account after email verification",
			logger.String("service", error.ServiceName),
			logger.String("user_id", userID),
		)
		activateErr := s.repo.ActivatePendingUser(ctx, userID)
		if activateErr != nil {
			s.log.Warn("Failed to activate pending account (non-critical)",
				logger.String("service", error.ServiceName),
				logger.String("user_id", userID),
				logger.Error(activateErr),
			)
		}
	}

	// ── 9. Log security event ────────────────────────────────────────────
	if s.securityEventRepo != nil {
		_ = s.securityEventRepo.LogEvent(ctx, repository.SecurityEventInput{
			UserID:        userID,
			EventType:     models.SecurityEventEmailVerified,
			EventCategory: "account_management",
			Severity:      models.SecuritySeverityLow,
			Status:        "success",
			Description:   "Email address verified successfully",
			IPAddress:     ipAddress,
			UserAgent:     userAgent,
		})
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

	email = normalizeEmail(email)

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
		ctx, user.ID, email, rawToken, *tokenHash, ipAddress, userAgent, expiresAt,
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
	emailErr := s.emailService.SendEmailVerificationEmail(email, user.ID, rawToken)
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

	// ── 7. Log security event ────────────────────────────────────────────
	if s.securityEventRepo != nil {
		_ = s.securityEventRepo.LogEvent(ctx, repository.SecurityEventInput{
			UserID:        user.ID,
			EventType:     models.SecurityEventVerificationEmailResent,
			EventCategory: "account_management",
			Severity:      models.SecuritySeverityLow,
			Status:        "success",
			Description:   "Verification email resent",
			IPAddress:     ipAddress,
			UserAgent:     userAgent,
		})
	}

	s.log.Info("Verification email resent successfully",
		logger.String("service", error.ServiceName),
		logger.String("email", email),
		logger.String("user_id", user.ID),
	)

	return &rawToken, nil
}
