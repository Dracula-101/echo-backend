package repository

import (
	authErrors "auth-service/internal/errors"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"shared/pkg/database"
	"shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
)

// ============================================================================
// Repository Definition
// ============================================================================

type AuthRepository struct {
	db  database.Database
	log logger.Logger
}

func NewAuthRepository(db database.Database, log logger.Logger) *AuthRepository {
	if db == nil {
		panic("Database is required for AuthRepository")
	}
	if log == nil {
		panic("Logger is required for AuthRepository")
	}

	log.Info("Initializing AuthRepository",
		logger.String("service", authErrors.ServiceName),
	)

	return &AuthRepository{
		db:  db,
		log: log,
	}
}

// ============================================================================
// Email Operations
// ============================================================================

func (r *AuthRepository) ExistsByEmail(ctx context.Context, email string) (bool, pkgErrors.AppError) {
	r.log.Debug("Checking if email exists",
		logger.String("service", authErrors.ServiceName),
		logger.String("email", email),
	)

	query := `SELECT EXISTS(SELECT 1 FROM auth.users WHERE email = $1)`
	var exists bool
	err := r.db.QueryRow(ctx, query, email).Scan(&exists)
	if err != nil {
		return false, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to check email existence").
			WithDetail("email", email)
	}

	r.log.Debug("Email existence check completed",
		logger.String("service", authErrors.ServiceName),
		logger.String("email", email),
		logger.Bool("exists", exists),
	)

	return exists, nil
}

// ============================================================================
// User Creation
// ============================================================================

type CreateUserParams struct {
	Email             string
	PasswordHash      string
	PasswordSalt      string
	PasswordAlgorithm string
	PhoneNumber       string
	PhoneCountryCode  string
	IPAddress         string
	UserAgent         string
}

func (r *AuthRepository) CreateUser(ctx context.Context, params CreateUserParams) (string, pkgErrors.AppError) {
	if params.Email == "" {
		return "", pkgErrors.New(pkgErrors.CodeInvalidArgument, "email is required for user creation")
	}
	if params.PasswordHash == "" {
		return "", pkgErrors.New(pkgErrors.CodeInvalidArgument, "password hash is required for user creation").
			WithDetail("email", params.Email)
	}

	r.log.Info("Creating user",
		logger.String("service", authErrors.ServiceName),
		logger.String("email", params.Email),
		logger.String("password_algorithm", params.PasswordAlgorithm),
		logger.String("ip_address", params.IPAddress),
	)

	now := time.Now()
	passwordHistory := fmt.Sprintf(`[{"hash":"%s","salt":"%s","algorithm":"%s","changed_at":"%s"}]`,
		params.PasswordHash,
		params.PasswordSalt,
		params.PasswordAlgorithm,
		now.Format(time.RFC3339),
	)
	passwordHistoryJson := json.RawMessage(passwordHistory)

	r.log.Debug("Inserting user record",
		logger.String("service", authErrors.ServiceName),
		logger.String("email", params.Email),
	)

	id, err := r.db.Create(ctx, &models.AuthUser{
		Email:                  params.Email,
		PhoneNumber:            &params.PhoneNumber,
		PhoneCountryCode:       &params.PhoneCountryCode,
		PasswordHash:           params.PasswordHash,
		PasswordSalt:           params.PasswordSalt,
		PasswordAlgorithm:      params.PasswordAlgorithm,
		EmailVerified:          false,
		PhoneVerified:          false,
		PasswordLastChangedAt:  nil,
		TwoFactorEnabled:       false,
		TwoFactorSecret:        nil,
		TwoFactorBackupCodes:   nil,
		AccountStatus:          models.AccountStatusActive,
		AccountLockedUntil:     nil,
		FailedLoginAttempts:    0,
		LastFailedLoginAt:      nil,
		RequiresPasswordChange: false,
		DeletedAt:              nil,
		UpdatedAt:              now,
		PasswordHistory:        passwordHistoryJson,
		CreatedByIP:            &params.IPAddress,
		CreatedByUserAgent:     &params.UserAgent,
	})
	if err != nil {
		return "", pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to create user").
			WithDetail("email", params.Email)
	}

	r.log.Info("User created successfully",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", *id),
		logger.String("email", params.Email),
	)

	return *id, nil
}

// ============================================================================
// User Retrieval
// ============================================================================

func (r *AuthRepository) GetUserByEmail(ctx context.Context, email string) (*models.AuthUser, pkgErrors.AppError) {
	r.log.Debug("Fetching user by email",
		logger.String("service", authErrors.ServiceName),
		logger.String("email", email),
	)

	query := `SELECT  * FROM auth.users WHERE email = $1 LIMIT 1`
	row := r.db.QueryRow(ctx, query, email)
	var user models.AuthUser
	err := row.ScanOne(&user)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.log.Debug("User not found by email",
				logger.String("service", authErrors.ServiceName),
				logger.String("email", email),
			)
			return nil, nil
		}
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get user by email").
			WithDetail("email", email)
	}

	r.log.Debug("User fetched successfully",
		logger.String("service", authErrors.ServiceName),
		logger.String("email", email),
		logger.String("user_id", user.ID),
		logger.Any("account_status", user.AccountStatus),
	)

	return &user, nil
}

// ============================================================================
// Login Tracking
// ============================================================================

func (r *AuthRepository) RecordFailedLogin(ctx context.Context, userID string) pkgErrors.AppError {
	r.log.Info("Recording failed login attempt",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
	)

	query := `UPDATE auth.users
		SET failed_login_attempts = failed_login_attempts + 1,
		    last_failed_login_at = NOW(),
		    updated_at = NOW()
		WHERE id = $1`
	result, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to record failed login").
			WithDetail("user_id", userID)
	}

	rowsAffected, _ := result.RowsAffected()
	r.log.Info("Failed login recorded successfully",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
		logger.Int64("rows_affected", rowsAffected),
	)
	return nil
}

func (r *AuthRepository) RecordSuccessfulLogin(ctx context.Context, userID string) pkgErrors.AppError {
	r.log.Info("Recording successful login",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
	)

	query := `UPDATE auth.users
		SET failed_login_attempts = 0,
		    last_failed_login_at = NULL,
			last_successful_login_at = NOW(),
		    updated_at = NOW()
		WHERE id = $1`
	result, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to record successful login").
			WithDetail("user_id", userID)
	}

	rowsAffected, _ := result.RowsAffected()
	r.log.Info("Successful login recorded",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
		logger.Int64("rows_affected", rowsAffected),
	)
	return nil
}
