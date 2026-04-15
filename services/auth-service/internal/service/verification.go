package service

import (
	ratelimiter "auth-service/api/v1/rate_limiter"
	"auth-service/internal/error"
	"context"
	"fmt"
	"time"

	"shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/pkg/utils"
	"shared/server/request"
)

const (
	maxVerificationAttempts  = 5
	maxResendTokensPerWindow = 5
)

func (s *authService) VerifyEmail(ctx context.Context, rawToken string, deviceInfo request.DeviceInfo, location request.IpAddressInfo) (email *string, authErr *error.AuthError) {
	s.log.Info("Processing email verification",
		logger.String("service", error.ServiceName),
		logger.String("ip_address", location.IP),
		logger.String("user_agent", deviceInfo.UserAgent),
	)

	tokenHash, hashErr := s.hashingService.SimpleHash(ctx, rawToken)
	if hashErr != nil {
		s.log.Error("Failed to hash verification token",
			logger.String("ip_address", location.IP),
			logger.String("user_agent", deviceInfo.UserAgent),
			logger.Error(hashErr),
		)
		return nil, &error.AuthError{
			Code:    error.CodeTokenGenerationFailed,
			Message: "Failed to create verification token",
			Error:   pkgErrors.FromError(hashErr, error.CodeTokenGenerationFailed, "failed to hash verification token"),
		}
	}

	verificationRecord, getErr := s.emailVerificationRepo.GetVerificationTokenByHash(ctx, *tokenHash)
	if getErr != nil {
		s.log.Error("Failed to retrieve verification token",
			logger.String("service", error.ServiceName),
			logger.String("ip_address", location.IP),
			logger.String("user_agent", deviceInfo.UserAgent),
			logger.Error(getErr),
		)
		return nil, &error.AuthError{
			Code:    error.CodeVerificationNotFound,
			Message: "Verification token not found",
			Error:   pkgErrors.FromError(getErr, error.CodeVerificationNotFound, "failed to retrieve verification token"),
		}
	}

	if verificationRecord == nil {
		s.log.Warn("Invalid verification token",
			logger.String("service", error.ServiceName),
			logger.String("ip_address", location.IP),
			logger.String("user_agent", deviceInfo.UserAgent),
		)
		return nil, &error.AuthError{
			Code:    error.CodeInvalidVerificationToken,
			Message: "Invalid verification token",
			Error:   pkgErrors.New(error.CodeInvalidVerificationToken, "token not found or does not belong to user"),
		}
	}

	user, userErr := s.repo.GetUserByID(ctx, verificationRecord.UserID)
	if userErr != nil {
		s.log.Error("Failed to retrieve user for email verification",
			logger.String("service", error.ServiceName),
			logger.String("user_id", verificationRecord.UserID),
			logger.Error(userErr),
		)
		return nil, &error.AuthError{
			Code:    error.CodeDatabaseError,
			Message: "Failed to process request",
			Error:   pkgErrors.FromError(userErr, error.CodeDatabaseError, "failed to retrieve user"),
		}
	}

	if user == nil {
		s.log.Warn("User not found for email verification",
			logger.String("service", error.ServiceName),
			logger.String("user_id", verificationRecord.UserID),
		)
		return nil, &error.AuthError{
			Code:    error.CodeUserNotFound,
			Message: "User not found",
			Error:   pkgErrors.New(error.CodeUserNotFound, "user not found"),
		}
	}

	if user.EmailVerified {
		s.log.Warn("Email already verified",
			logger.String("service", error.ServiceName),
			logger.String("user_id", verificationRecord.UserID),
		)
		return nil, &error.AuthError{
			Code:    error.CodeEmailAlreadyVerified,
			Message: "Email is already verified",
			Error:   pkgErrors.New(error.CodeEmailAlreadyVerified, "email already verified"),
		}
	}

	if user.AccountStatus == models.AccountStatusDeleted ||
		user.AccountStatus == models.AccountStatusSuspended ||
		user.AccountStatus == models.AccountStatusDeactivated {
		s.log.Warn("Account ineligible for email verification",
			logger.String("service", error.ServiceName),
			logger.String("user_id", verificationRecord.UserID),
			logger.String("account_status", string(user.AccountStatus)),
		)
		return nil, &error.AuthError{
			Code:    error.CodeUserNotFound,
			Message: "User not found",
			Error:   pkgErrors.New(error.CodeUserNotFound, "account ineligible"),
		}
	}

	if verificationRecord.ExpiresAt.Before(time.Now()) {
		s.log.Warn("Expired verification token",
			logger.String("service", error.ServiceName),
			logger.String("user_id", verificationRecord.UserID),
		)
		return nil, &error.AuthError{
			Code:    error.CodeVerificationTokenExpired,
			Message: "Verification token has expired",
			Error:   pkgErrors.New(error.CodeVerificationTokenExpired, "verification token expired"),
		}
	}

	if verificationRecord.Attempts >= maxVerificationAttempts {
		s.log.Warn("Too many verification attempts",
			logger.String("service", error.ServiceName),
			logger.String("user_id", verificationRecord.UserID),
		)
		return nil, &error.AuthError{
			Code:    error.CodeTooManyVerificationAttempts,
			Message: "Too many verification attempts. Please request a new verification email.",
			Error:   pkgErrors.New(error.CodeTooManyVerificationAttempts, "too many verification attempts"),
		}
	}

	// add attempt before updating to prevent race conditions
	err := s.emailVerificationRepo.AttemptVerification(ctx, *tokenHash, verificationRecord.UserID)
	if err != nil {
		s.log.Warn("Failed to record verification attempt (proceeding anyway)",
			logger.String("service", error.ServiceName),
			logger.String("user_id", verificationRecord.UserID),
			logger.Error(err),
		)
		return nil, &error.AuthError{
			Code:    error.CodeDatabaseError,
			Message: "Failed to process verification attempt",
			Error:   pkgErrors.FromError(err, error.CodeDatabaseError, "failed to record verification attempt"),
		}
	}

	updateErr := s.repo.MarkEmailVerified(ctx, verificationRecord.UserID, deviceInfo, location)
	if updateErr != nil {
		s.log.Error("Failed to mark user email as verified",
			logger.String("service", error.ServiceName),
			logger.String("user_id", verificationRecord.UserID),
			logger.Error(updateErr),
		)
		return nil, &error.AuthError{
			Code:    error.CodeDatabaseError,
			Message: "Failed to process verification attempt",
			Error:   pkgErrors.FromError(updateErr, error.CodeDatabaseError, "failed to mark user email as verified"),
		}
	}

	s.log.Info("Email verified successfully",
		logger.String("service", error.ServiceName),
		logger.String("user_id", verificationRecord.UserID),
	)

	return &user.Email, nil
}

func (s *authService) ResendVerification(ctx context.Context, email string, ipAddress string, userAgent string) (*string, *error.AuthError) {
	s.log.Info("Processing resend verification email",
		logger.String("service", error.ServiceName),
		logger.String("email", email),
	)

	email = utils.NormalizeEmail(email)

	// ── Look up user ──────────────────────────────────────────────────
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

	// ── Reject if ineligible ──────────────────────────────────────────
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

	// ── Rate limit ───────────────────────────────────────────────────
	allowed, remaining, rlErr := s.rateLimiter.AllowResendVerify(ctx, user.ID, maxResendTokensPerWindow)
	if rlErr != nil {
		s.log.Error("Failed to check resend verification rate limit",
			logger.String("service", error.ServiceName),
			logger.String("user_id", user.ID),
			logger.Error(rlErr),
		)
		return nil, &error.AuthError{
			Code:    error.CodeCacheError,
			Message: "Failed to process request",
			Error:   rlErr,
		}
	}
	if !allowed {
		s.log.Warn("Resend verification rate limit exceeded",
			logger.String("service", error.ServiceName),
			logger.String("user_id", user.ID),
			logger.Int("remaining_seconds", int(remaining)),
		)
		return nil, &error.AuthError{
			Code:    error.CodeVerificationCooldown,
			Message: "Too many resend attempts. Please try again later.",
			Error:   pkgErrors.New(error.CodeVerificationCooldown, "resend verification rate limit exceeded"),
		}
	}

	// Check Redis pwd_reset_sent:{user_id} pattern — equivalent key resend_verify_sent:{user_id}. If present and age < 60 seconds: return 429 with retry_after_seconds = 60 - age.
	allowed, retryAfter, cacheErr := s.rateLimiter.CanSendPwdReset(ctx, user.ID)
	if cacheErr != nil {
		s.log.Error("Failed to check resend verification cooldown",
			logger.String("service", error.ServiceName),
			logger.String("user_id", user.ID),
			logger.Error(cacheErr),
		)
		return nil, &error.AuthError{
			Code:    error.CodeCacheError,
			Message: "Failed to process request",
			Error:   cacheErr,
		}
	}
	if !allowed {
		s.log.Warn("Resend verification cooldown active",
			logger.String("service", error.ServiceName),
			logger.String("user_id", user.ID),
			logger.Int("retry_after_seconds", int(retryAfter)),
		)
		return nil, &error.AuthError{
			Code:    error.CodeVerificationCooldown,
			Message: fmt.Sprintf("Verification email recently sent. Try again after %d seconds.", int(retryAfter)),
			Error:   pkgErrors.New(error.CodeVerificationCooldown, "resend verification cooldown active"),
			Detail: map[string]interface{}{
				"retry_after_seconds": int(retryAfter),
			},
		}
	}

	err = s.emailVerificationRepo.ExpireTokensByUserID(ctx, user.ID)
	if err != nil {
		s.log.Error("Failed to expire existing email verification tokens",
			logger.String("service", error.ServiceName),
			logger.String("user_id", user.ID),
			logger.Error(err),
		)
		return nil, &error.AuthError{
			Code:    error.CodeDatabaseError,
			Message: "Failed to process request",
			Error:   err,
		}
	}

	// ── Generate new token ────────────────────────────────────────────
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
	expiresAt := time.Now().Add(ratelimiter.TTLResendVerify)
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

	// ── Send the email (only after DB insert succeeds) ────────────────
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

	// -─ Set cooldown in Redis ───────────────────────────────────────────
	err = s.rateLimiter.SetPwdResetCooldown(ctx, user.ID)
	if err != nil {
		s.log.Error("Failed to set resend verification cooldown",
			logger.String("service", error.ServiceName),
			logger.String("user_id", user.ID),
			logger.Error(err),
		)
	}

	s.log.Info("Verification email resent successfully",
		logger.String("service", error.ServiceName),
		logger.String("email", email),
		logger.String("user_id", user.ID),
	)

	// TODO:
	// add into auth.outbox for auth.password.reset_requested.

	return &rawToken, nil
}
