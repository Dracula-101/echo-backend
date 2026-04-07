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

type EmailVerificationTokenRepositoryInterface interface {
	CreateVerificationToken(ctx context.Context, userID string, email string, tokenHash string, ipAddress string, userAgent string, expiresAt time.Time) pkgErrors.AppError
	GetVerificationTokenByHash(ctx context.Context, tokenHash string) (*models.EmailVerificationToken, pkgErrors.AppError)
	MarkTokenVerified(ctx context.Context, tokenID string) pkgErrors.AppError
	GetLatestTokenByUserID(ctx context.Context, userID string) (*models.EmailVerificationToken, pkgErrors.AppError)
}

type EmailVerificationTokenRepo struct {
	db  database.Database
	log logger.Logger
}

func NewEmailVerificationTokenRepo(db database.Database, log logger.Logger) *EmailVerificationTokenRepo {
	if db == nil {
		panic("Database is required for EmailVerificationTokenRepo")
	}
	if log == nil {
		panic("Logger is required for EmailVerificationTokenRepo")
	}
	return &EmailVerificationTokenRepo{db: db, log: log}
}

func (r *EmailVerificationTokenRepo) CreateVerificationToken(ctx context.Context, userID string, email string, tokenHash string, ipAddress string, userAgent string, expiresAt time.Time) pkgErrors.AppError {
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

func (r *EmailVerificationTokenRepo) GetVerificationTokenByHash(ctx context.Context, tokenHash string) (*models.EmailVerificationToken, pkgErrors.AppError) {
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

func (r *EmailVerificationTokenRepo) MarkTokenVerified(ctx context.Context, tokenID string) pkgErrors.AppError {
	r.log.Info("Marking email verification token as verified",
		logger.String("token_id", tokenID),
	)

	query := `UPDATE auth.email_verification_tokens SET verified_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, tokenID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to mark token as verified").
			WithDetail("token_id", tokenID)
	}
	return nil
}

func (r *EmailVerificationTokenRepo) GetLatestTokenByUserID(ctx context.Context, userID string) (*models.EmailVerificationToken, pkgErrors.AppError) {
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
