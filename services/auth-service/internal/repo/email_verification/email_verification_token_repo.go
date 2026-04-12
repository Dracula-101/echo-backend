package email_verification

import (
	"context"
	"database/sql"
	"errors"
	"time"

	authErrors "auth-service/internal/error"
	"shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
)

type EmailVerificationTokenRepositoryInterface interface {
	CreateVerificationToken(ctx context.Context, userID string, email string, tokenHash string, ipAddress string, userAgent string, expiresAt time.Time) pkgErrors.AppError
	GetVerificationTokenByHash(ctx context.Context, tokenHash string) (*models.EmailVerificationToken, pkgErrors.AppError)
	MarkTokenVerified(ctx context.Context, tokenHash string, userId string) pkgErrors.AppError
	GetLatestTokenByUserID(ctx context.Context, userID string) (*models.EmailVerificationToken, pkgErrors.AppError)
	CountRecentTokensByUserID(ctx context.Context, userID string, since time.Time) (int, pkgErrors.AppError)
	InvalidateUserTokens(ctx context.Context, userID string) pkgErrors.AppError
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
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get email verification token")
	}
	return &token, nil
}

func (r *EmailVerificationTokenRepository) MarkTokenVerified(ctx context.Context, tokenHash string, userId string) pkgErrors.AppError {
	// if the token is wrong add to attempt
	r.log.Info("Marking email verification token as verified",
		logger.String("token_hash", tokenHash),
	)

	// check if the token exists
	query := `SELECT * FROM auth.email_verification_tokens WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1`
	var token models.EmailVerificationToken
	err := r.db.QueryRow(ctx, query, userId).ScanModel(&token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return pkgErrors.New(authErrors.CodeVerificationNotFound, "verification token not found")
		}
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get email verification token")
	}

	if token.VerifiedAt != nil {
		return nil
	}
	if token.ExpiresAt.Before(time.Now()) {
		return pkgErrors.New(authErrors.CodeVerificationTokenExpired, "verification token expired")
	}

	if token.TokenHash != tokenHash {
		query := `UPDATE auth.email_verification_tokens SET attempts = attempts + 1 WHERE token_hash = $1 AND user_id = $2`
		_, err := r.db.Exec(ctx, query, tokenHash, userId)
		if err != nil {
			return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to update verification token attempts")
		}
		return pkgErrors.New(authErrors.CodeVerificationInvalid, "invalid verification token")
	}

	query = `UPDATE auth.email_verification_tokens SET verified_at = $1 WHERE token_hash = $2 AND user_id = $3`
	_, err = r.db.Exec(ctx, query, time.Now(), tokenHash, userId)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to mark email verification token as verified")
	}

	r.log.Info("Email verification token marked as verified",
		logger.String("token_hash", tokenHash),
	)
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

func (r *EmailVerificationTokenRepository) CountRecentTokensByUserID(ctx context.Context, userID string, since time.Time) (int, pkgErrors.AppError) {
	r.log.Debug("Counting recent verification tokens for user",
		logger.String("user_id", userID),
	)

	query := `SELECT COUNT(*) FROM auth.email_verification_tokens WHERE user_id = $1 AND created_at > $2`
	var count int
	err := r.db.QueryRow(ctx, query, userID, since).Scan(&count)
	if err != nil {
		return 0, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to count recent verification tokens").
			WithDetail("user_id", userID)
	}
	return count, nil
}

func (r *EmailVerificationTokenRepository) InvalidateUserTokens(ctx context.Context, userID string) pkgErrors.AppError {
	r.log.Info("Deleting old unverified tokens for user",
		logger.String("user_id", userID),
	)

	query := `DELETE FROM auth.email_verification_tokens WHERE user_id = $1 AND verified_at IS NULL`
	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to delete old verification tokens").
			WithDetail("user_id", userID)
	}
	return nil
}
