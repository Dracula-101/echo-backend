package email_verification

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
)

type EmailVerificationTokenRepositoryInterface interface {
	CreateVerificationToken(ctx context.Context, userID string, email string, tokenHash string, ipAddress string, userAgent string, expiresAt time.Time) pkgErrors.AppError
	GetVerificationTokenByHash(ctx context.Context, tokenHash string) (*models.EmailVerificationToken, pkgErrors.AppError)
	AttemptVerification(ctx context.Context, tokenHash string, userId string) pkgErrors.AppError
	GetLatestTokenByUserID(ctx context.Context, userID string) (*models.EmailVerificationToken, pkgErrors.AppError)
	ExpireTokensByUserID(ctx context.Context, userID string) pkgErrors.AppError
}

func (r *EmailVerificationTokenRepository) CreateVerificationToken(ctx context.Context, userID string, email string, tokenHash string, ipAddress string, userAgent string, expiresAt time.Time) pkgErrors.AppError {
	r.log.Info("Creating email verification token",
		logger.String("user_id", userID),
		logger.String("email", email),
	)

	_, err := r.db.Insert(ctx, &models.EmailVerificationToken{
		UserID:    userID,
		Email:     email,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		IPAddress: &ipAddress,
		UserAgent: &userAgent,
	})
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to create email verification token").
			WithDetail("user_id", userID).
			WithDetail("email", email)
	}

	r.log.Info("Email verification token created",
		logger.String("user_id", userID),
	)
	return nil
}

func (r *EmailVerificationTokenRepository) GetVerificationTokenByHash(ctx context.Context, tokenHash string) (*models.EmailVerificationToken, pkgErrors.AppError) {
	r.log.Debug("Fetching email verification token by hash")

	query := `SELECT * FROM auth.email_verification_tokens WHERE token_hash = $1 LIMIT 1`
	var token models.EmailVerificationToken
	err := r.db.QueryRow(ctx, query, tokenHash).ScanModel(&token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		r.log.Error("Failed to get email verification token",
			logger.String("token_hash", tokenHash),
			logger.Error(err),
		)
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get email verification token")
	}
	return &token, nil
}

func (r *EmailVerificationTokenRepository) AttemptVerification(ctx context.Context, tokenHash string, userId string) pkgErrors.AppError {
	r.log.Info("Recording email verification attempt",
		logger.String("token_hash", tokenHash),
	)

	query := `UPDATE auth.email_verification_tokens SET attempts = attempts + 1 WHERE token_hash = $1 AND user_id = $2`
	_, err := r.db.Exec(ctx, query, tokenHash, userId)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to record email verification attempt")
	}
	return nil
}

func (r *EmailVerificationTokenRepository) GetLatestTokenByUserID(ctx context.Context, userID string) (*models.EmailVerificationToken, pkgErrors.AppError) {
	r.log.Debug("Fetching latest email verification token for user",
		logger.String("user_id", userID),
	)

	query := `SELECT * FROM auth.email_verification_tokens WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1`
	var token models.EmailVerificationToken
	err := r.db.QueryRow(ctx, query, userID).ScanModel(&token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get latest email verification token").
			WithDetail("user_id", userID)
	}
	return &token, nil
}

func (r *EmailVerificationTokenRepository) ExpireTokensByUserID(ctx context.Context, userID string) pkgErrors.AppError {
	r.log.Info("Expiring existing email verification tokens for user",
		logger.String("user_id", userID),
	)
	// UPDATE auth.email_verification_tokens SET expires_at = NOW() WHERE user_id = $1 AND verified_at IS NULL. Marks all outstanding tokens as expired — prevents old links from working after a resend.
	query := `UPDATE auth.email_verification_tokens SET expires_at = NOW() WHERE user_id = $1 AND verified_at IS NULL`
	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to expire existing email verification tokens").
			WithDetail("user_id", userID)
	}
	return nil
}
