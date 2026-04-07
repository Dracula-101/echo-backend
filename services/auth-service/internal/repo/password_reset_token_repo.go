package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"shared/pkg/database"
	"shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
)

type PasswordResetTokenRepositoryInterface interface {
	CreateResetToken(ctx context.Context, userID string, tokenHash string, ipAddress string, userAgent string, expiresAt time.Time) pkgErrors.AppError
	GetResetTokenByHash(ctx context.Context, tokenHash string) (*models.PasswordResetToken, pkgErrors.AppError)
	MarkTokenUsed(ctx context.Context, tokenID string) pkgErrors.AppError
	InvalidateUserTokens(ctx context.Context, userID string) pkgErrors.AppError
}

type PasswordResetTokenRepo struct {
	db  database.Database
	log logger.Logger
}

func NewPasswordResetTokenRepo(db database.Database, log logger.Logger) *PasswordResetTokenRepo {
	if db == nil {
		panic("Database is required for PasswordResetTokenRepo")
	}
	if log == nil {
		panic("Logger is required for PasswordResetTokenRepo")
	}
	return &PasswordResetTokenRepo{db: db, log: log}
}

func (r *PasswordResetTokenRepo) CreateResetToken(ctx context.Context, userID string, tokenHash string, ipAddress string, userAgent string, expiresAt time.Time) pkgErrors.AppError {
	r.log.Info("Creating password reset token",
		logger.String("user_id", userID),
	)

	now := time.Now()
	_, err := r.db.Insert(ctx, &models.PasswordResetToken{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		IPAddress: &ipAddress,
		UserAgent: &userAgent,
		EmailSentAt: &now,
	})
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to create password reset token").
			WithDetail("user_id", userID)
	}

	r.log.Info("Password reset token created",
		logger.String("user_id", userID),
	)
	return nil
}

func (r *PasswordResetTokenRepo) GetResetTokenByHash(ctx context.Context, tokenHash string) (*models.PasswordResetToken, pkgErrors.AppError) {
	r.log.Debug("Fetching password reset token by hash")

	query := `SELECT * FROM auth.password_reset_tokens WHERE token_hash = $1 LIMIT 1`
	var token models.PasswordResetToken
	err := r.db.QueryRow(ctx, query, tokenHash).ScanModel(&token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get password reset token")
	}
	return &token, nil
}

func (r *PasswordResetTokenRepo) MarkTokenUsed(ctx context.Context, tokenID string) pkgErrors.AppError {
	r.log.Info("Marking password reset token as used",
		logger.String("token_id", tokenID),
	)

	query := `UPDATE auth.password_reset_tokens SET used_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, tokenID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to mark token as used").
			WithDetail("token_id", tokenID)
	}
	return nil
}

func (r *PasswordResetTokenRepo) InvalidateUserTokens(ctx context.Context, userID string) pkgErrors.AppError {
	r.log.Info("Invalidating all password reset tokens for user",
		logger.String("user_id", userID),
	)

	query := `UPDATE auth.password_reset_tokens SET used_at = NOW() WHERE user_id = $1 AND used_at IS NULL`
	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to invalidate user tokens").
			WithDetail("user_id", userID)
	}
	return nil
}
