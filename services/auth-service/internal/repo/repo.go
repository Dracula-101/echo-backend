package repository

import (
	"auth-service/internal/domain"
	authErrors "auth-service/internal/error"
	repoModels "auth-service/internal/repo/model"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"shared/pkg/database"
	"shared/pkg/database/postgres"
	"shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/pkg/utils"
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

	query := `SELECT EXISTS(SELECT 1 FROM auth.users WHERE LOWER(email) = LOWER($1) AND deleted_at IS NULL)`
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

func (r *AuthRepository) CreateUser(ctx context.Context, params repoModels.CreateUserParams) (string, pkgErrors.AppError) {
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

	id, err := r.db.Insert(ctx, &models.AuthUser{
		Email:                  params.Email,
		PhoneNumber:            params.PhoneNumber,
		PhoneCountryCode:       params.PhoneCountryCode,
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

func (r *AuthRepository) UnlockUserAccount(ctx context.Context, userID string) pkgErrors.AppError {
	r.log.Info("Unlocking user account",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
	)

	query := `UPDATE auth.users
		SET account_locked_until = NULL,
		    failed_login_attempts = 0,
			last_failed_login_at = NULL,
		    updated_at = NOW()
		WHERE id = $1`
	result, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to unlock user account").
			WithDetail("user_id", userID)
	}

	rowsAffected, _ := result.RowsAffected()
	r.log.Info("User account unlocked successfully",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
		logger.Int64("rows_affected", rowsAffected),
	)
	return nil
}

// ============================================================================
// User Retrieval
// ============================================================================

func (r *AuthRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, pkgErrors.AppError) {
	r.log.Debug("Fetching user by email",
		logger.String("service", authErrors.ServiceName),
		logger.String("email", email),
	)

	query := `SELECT  * FROM auth.users WHERE email = $1 LIMIT 1`
	row := r.db.QueryRow(ctx, query, email)
	var user models.AuthUser
	err := row.ScanModel(&user)
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

	return &domain.User{
		ID:                     user.ID,
		Email:                  user.Email,
		PhoneNumber:            utils.DerefString(user.PhoneNumber),
		PhoneCountryCode:       utils.DerefString(user.PhoneCountryCode),
		PasswordHash:           user.PasswordHash,
		PasswordSalt:           user.PasswordSalt,
		PasswordAlgorithm:      user.PasswordAlgorithm,
		EmailVerified:          user.EmailVerified,
		PhoneVerified:          user.PhoneVerified,
		TwoFactorEnabled:       user.TwoFactorEnabled,
		TwoFactorSecret:        utils.DerefString(user.TwoFactorSecret),
		AccountStatus:          user.AccountStatus,
		AccountLockedUntil:     user.AccountLockedUntil,
		FailedLoginAttempts:    user.FailedLoginAttempts,
		LastFailedLoginAt:      user.LastFailedLoginAt,
		RequiresPasswordChange: user.RequiresPasswordChange,
		PasswordLastChanged:    user.PasswordLastChangedAt,
		LastSuccessfulLoginAt:  user.LastSuccessfulLoginAt,
		CreatedAt:              user.CreatedAt,
		DeletedAt:              user.DeletedAt,
		UpdatedAt:              user.UpdatedAt,
	}, nil
}

func (r *AuthRepository) GetUserByID(ctx context.Context, userID string) (*domain.User, pkgErrors.AppError) {
	r.log.Debug("Fetching user by ID",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
	)

	query := `SELECT * FROM auth.users WHERE id = $1 LIMIT 1`
	row := r.db.QueryRow(ctx, query, userID)
	var user models.AuthUser
	err := row.ScanModel(&user)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.log.Debug("User not found by ID",
				logger.String("service", authErrors.ServiceName),
				logger.String("user_id", userID),
			)
			return nil, nil
		}
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get user by ID").
			WithDetail("user_id", userID)
	}

	r.log.Debug("User fetched successfully",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", user.ID),
		logger.Any("account_status", user.AccountStatus),
	)

	return &domain.User{
		ID:                     user.ID,
		Email:                  user.Email,
		PhoneNumber:            utils.DerefString(user.PhoneNumber),
		PhoneCountryCode:       utils.DerefString(user.PhoneCountryCode),
		PasswordHash:           user.PasswordHash,
		PasswordSalt:           user.PasswordSalt,
		PasswordAlgorithm:      user.PasswordAlgorithm,
		EmailVerified:          user.EmailVerified,
		PhoneVerified:          user.PhoneVerified,
		TwoFactorEnabled:       user.TwoFactorEnabled,
		TwoFactorSecret:        utils.DerefString(user.TwoFactorSecret),
		AccountStatus:          user.AccountStatus,
		AccountLockedUntil:     user.AccountLockedUntil,
		FailedLoginAttempts:    user.FailedLoginAttempts,
		LastFailedLoginAt:      user.LastFailedLoginAt,
		RequiresPasswordChange: user.RequiresPasswordChange,
		PasswordLastChanged:    user.PasswordLastChangedAt,
		LastSuccessfulLoginAt:  user.LastSuccessfulLoginAt,
		CreatedAt:              user.CreatedAt,
		DeletedAt:              user.DeletedAt,
		UpdatedAt:              user.UpdatedAt,
	}, nil
}

func (r *AuthRepository) GetUserActiveDevice(ctx context.Context, userID string) (*domain.UserDevice, pkgErrors.AppError) {
	r.log.Debug("Fetching active device for user",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
	)

	query := `SELECT * FROM users.devices WHERE user_id = $1 AND is_active = TRUE LIMIT 1`
	row := r.db.QueryRow(ctx, query, userID)
	var device models.Device
	err := row.ScanModel(&device)
	if err != nil {
		if postgres.IsNotFoundError(err) {
			r.log.Debug("No active device found for user",
				logger.String("service", authErrors.ServiceName),
				logger.String("user_id", userID),
			)
			return nil, nil
		}
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get active device for user").
			WithDetail("user_id", userID).
			WithDetail("query", query)
	}

	r.log.Debug("Active device fetched successfully",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
		logger.String("device_id", device.DeviceID),
	)
	return domain.NewUserDevice(device), nil
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

// ============================================================================
// Password Management
// ============================================================================

func (r *AuthRepository) UpdatePassword(ctx context.Context, userID string, hash string, salt string, algorithm string) pkgErrors.AppError {
	r.log.Info("Updating user password",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
		logger.String("algorithm", algorithm),
	)

	query := `UPDATE auth.users
		SET password_hash = $1,
		    password_salt = $2,
		    password_algorithm = $3,
		    password_last_changed_at = NOW(),
		    requires_password_change = FALSE,
		    updated_at = NOW()
		WHERE id = $4`
	result, err := r.db.Exec(ctx, query, hash, salt, algorithm, userID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to update password").
			WithDetail("user_id", userID)
	}

	rowsAffected, _ := result.RowsAffected()
	r.log.Info("Password updated successfully",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
		logger.Int64("rows_affected", rowsAffected),
	)
	return nil
}

// ============================================================================
// Email Verification
// ============================================================================

func (r *AuthRepository) MarkEmailVerified(ctx context.Context, userID string) pkgErrors.AppError {
	r.log.Info("Marking email as verified",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
	)

	query := `UPDATE auth.users
		SET email_verified = TRUE,
		    updated_at = NOW()
		WHERE id = $1`
	result, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to mark email as verified").
			WithDetail("user_id", userID)
	}

	rowsAffected, _ := result.RowsAffected()
	r.log.Info("Email marked as verified",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
		logger.Int64("rows_affected", rowsAffected),
	)
	return nil
}

func (r *AuthRepository) ActivatePendingUser(ctx context.Context, userID string) pkgErrors.AppError {
	r.log.Info("Activating pending user account",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
	)

	query := `UPDATE auth.users
		SET account_status = $1,
		    updated_at = NOW()
		WHERE id = $2 AND account_status = $3`
	result, err := r.db.Exec(ctx, query, models.AccountStatusActive, userID, models.AccountStatusPending)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to activate pending user").
			WithDetail("user_id", userID)
	}

	rowsAffected, _ := result.RowsAffected()
	r.log.Info("Pending user activated",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
		logger.Int64("rows_affected", rowsAffected),
	)
	return nil
}

// ============================================================================
// Two-Factor Authentication
// ============================================================================

func (r *AuthRepository) Update2FASecret(ctx context.Context, userID string, secret string) pkgErrors.AppError {
	r.log.Info("Updating 2FA secret",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
	)

	query := `UPDATE auth.users
		SET two_factor_secret = $1,
		    updated_at = NOW()
		WHERE id = $2`
	_, err := r.db.Exec(ctx, query, secret, userID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to update 2FA secret").
			WithDetail("user_id", userID)
	}

	r.log.Info("2FA secret updated",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
	)
	return nil
}

func (r *AuthRepository) Enable2FA(ctx context.Context, userID string) pkgErrors.AppError {
	r.log.Info("Enabling 2FA",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
	)

	query := `UPDATE auth.users
		SET two_factor_enabled = TRUE,
		    updated_at = NOW()
		WHERE id = $1`
	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to enable 2FA").
			WithDetail("user_id", userID)
	}

	r.log.Info("2FA enabled",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
	)
	return nil
}

func (r *AuthRepository) Disable2FA(ctx context.Context, userID string) pkgErrors.AppError {
	r.log.Info("Disabling 2FA",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
	)

	query := `UPDATE auth.users
		SET two_factor_enabled = FALSE,
		    two_factor_secret = NULL,
		    two_factor_backup_codes = NULL,
		    updated_at = NOW()
		WHERE id = $1`
	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to disable 2FA").
			WithDetail("user_id", userID)
	}

	r.log.Info("2FA disabled",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
	)
	return nil
}

// ============================================================================
// Account Management
// ============================================================================

func (r *AuthRepository) SoftDeleteUser(ctx context.Context, userID string) pkgErrors.AppError {
	r.log.Info("Soft deleting user",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
	)

	query := `UPDATE auth.users
		SET account_status = $1,
		    deleted_at = NOW(),
		    updated_at = NOW()
		WHERE id = $2`
	_, err := r.db.Exec(ctx, query, models.AccountStatusDeleted, userID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to soft delete user").
			WithDetail("user_id", userID)
	}

	r.log.Info("User soft deleted",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
	)
	return nil
}
