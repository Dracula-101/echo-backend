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
	"strings"
	"time"

	"shared/pkg/database"
	"shared/pkg/database/postgres"
	dbModels "shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/pkg/utils"
	"shared/server/request"
)

const (
	AccountLockDuration = 30 * time.Minute
)

// ============================================================================
// Repository Definition
// ============================================================================

type authRepository struct {
	db  database.Database
	log logger.Logger
}

func NewAuthRepository(db database.Database, log logger.Logger) AuthRepository {
	if db == nil {
		panic("Database is required for AuthRepository")
	}
	if log == nil {
		panic("Logger is required for AuthRepository")
	}

	log.Info("Initializing AuthRepository",
		logger.String("service", authErrors.ServiceName),
	)

	return &authRepository{
		db:  db,
		log: log,
	}
}

// ============================================================================
// Email Operations
// ============================================================================
func (r *authRepository) ExistsByEmail(ctx context.Context, email string) (*dbModels.AccountStatus, pkgErrors.AppError) {
	r.log.Debug("Checking if email exists",
		logger.String("service", authErrors.ServiceName),
		logger.String("email", email),
	)
	//Query auth.users using idx_auth_users_email: SELECT id, account_status FROM auth.users WHERE LOWER(email) = LOWER($1)
	query := `SELECT account_status FROM auth.users WHERE LOWER(email) = LOWER($1) LIMIT 1`
	row := r.db.QueryRow(ctx, query, email)
	var accountStatus dbModels.AccountStatus
	err := row.Scan(&accountStatus)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.log.Debug("Email does not exist",
				logger.String("service", authErrors.ServiceName),
				logger.String("email", email),
			)
			return nil, nil
		}
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to check email existence").
			WithDetail("email", email)
	}

	r.log.Debug("Email exists with account status",
		logger.String("service", authErrors.ServiceName),
		logger.String("email", email),
		logger.Any("account_status", accountStatus),
	)

	return &accountStatus, nil
}

// ============================================================================
// User Creation
// ============================================================================

func (r *authRepository) CreateUser(ctx context.Context, params repoModels.CreateUserParams) (string, pkgErrors.AppError) {
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

	metaData := map[string]interface{}{
		"registration_type":        "email",
		"ip_country":               utils.SafeString(&params.Country),
		"registration_app_version": utils.SafeString(&params.AppVersion),
	}
	metaDataJson := utils.MarshalRawMessageSafe(metaData)
	id, err := r.db.Insert(ctx, &dbModels.AuthUser{
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
		AccountStatus:          dbModels.AccountStatusPending,
		AccountLockedUntil:     nil,
		FailedLoginAttempts:    0,
		LastFailedLoginAt:      nil,
		RequiresPasswordChange: false,
		DeletedAt:              nil,
		UpdatedAt:              now,
		PasswordHistory:        passwordHistoryJson,
		CreatedByIP:            &params.IPAddress,
		CreatedByUserAgent:     &params.UserAgent,
		MetaData:               &metaDataJson,
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

func (r *authRepository) LockUserAccount(ctx context.Context, userID string) pkgErrors.AppError {
	r.log.Info("Locking user account",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
	)

	lockUntil := time.Now().Add(AccountLockDuration)
	query := `UPDATE auth.users
		SET account_locked_until = $1,
		    updated_at = NOW()
		WHERE id = $2`
	result, err := r.db.Exec(ctx, query, lockUntil, userID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to lock user account").
			WithDetail("user_id", userID)
	}

	rowsAffected, _ := result.RowsAffected()
	r.log.Info("User account locked successfully",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
		logger.Int64("rows_affected", rowsAffected),
	)
	return nil
}

func (r *authRepository) UnlockUserAccount(ctx context.Context, userID string) pkgErrors.AppError {
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

func (r *authRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, pkgErrors.AppError) {
	r.log.Debug("Fetching user by email",
		logger.String("service", authErrors.ServiceName),
		logger.String("email", email),
	)

	query := `SELECT  * FROM auth.users WHERE email = $1 LIMIT 1`
	row := r.db.QueryRow(ctx, query, email)
	var user dbModels.AuthUser
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

func (r *authRepository) GetUserByID(ctx context.Context, userID string) (*domain.User, pkgErrors.AppError) {
	r.log.Debug("Fetching user by ID",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
	)

	query := `SELECT * FROM auth.users WHERE id = $1 LIMIT 1`
	row := r.db.QueryRow(ctx, query, userID)
	var user dbModels.AuthUser
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

func (r *authRepository) GetLoginAttempts(ctx context.Context, userID string) (int, pkgErrors.AppError) {
	r.log.Debug("Fetching failed login attempts for user",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
	)

	query := `SELECT failed_login_attempts FROM auth.users WHERE id = $1 LIMIT 1`
	row := r.db.QueryRow(ctx, query, userID)
	var attempts int
	err := row.Scan(&attempts)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.log.Debug("User not found when fetching login attempts",
				logger.String("service", authErrors.ServiceName),
				logger.String("user_id", userID),
			)
			return 0, nil
		}
		return 0, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get login attempts for user").
			WithDetail("user_id", userID)
	}

	r.log.Debug("Failed login attempts fetched successfully",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
		logger.Int("failed_login_attempts", attempts),
	)

	return attempts, nil
}

func (r *authRepository) GetUserActiveDevice(ctx context.Context, userID string) (*domain.UserDevice, pkgErrors.AppError) {
	r.log.Debug("Fetching active device for user",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
	)

	query := `SELECT * FROM users.devices WHERE user_id = $1 AND is_active = TRUE LIMIT 1`
	row := r.db.QueryRow(ctx, query, userID)
	var device dbModels.Device
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
// Password Management
// ============================================================================

func (r *authRepository) UpdatePassword(ctx context.Context, userID string, hash string, salt string, algorithm string) pkgErrors.AppError {
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

func (r *authRepository) MarkEmailVerified(ctx context.Context, userID string, deviceInfo request.DeviceInfo, location request.IpAddressInfo) pkgErrors.AppError {
	r.log.Info("Marking email as verified",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
	)

	tx, dbErr := r.db.BeginTx(ctx, nil)
	if dbErr != nil {
		return pkgErrors.FromError(dbErr, pkgErrors.CodeInternal, "failed to begin transaction")
	}
	var txErr *database.DBError
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if txErr != nil {
			tx.Rollback()
		}
	}()

	updateTokenQuery := `UPDATE auth.email_verification_tokens SET verified_at = NOW() WHERE user_id = $1 AND verified_at IS NULL`
	_, err := tx.Exec(ctx, updateTokenQuery, userID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to update email verification token").
			WithDetail("user_id", userID).
			WithDetail("query", updateTokenQuery)
	}

	updateUserQuery := `UPDATE auth.users SET email_verified = TRUE, account_status = $1, updated_at = NOW() WHERE id = $2`
	_, err = tx.Exec(ctx, updateUserQuery, dbModels.AccountStatusActive, userID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to update user email verification status").
			WithDetail("user_id", userID).
			WithDetail("query", updateUserQuery)
	}

	metaData := map[string]interface{}{
		"device_platform": deviceInfo.Platform,
		"ip_country":      location.Country,
		"app_version":     deviceInfo.AppVersion,
	}
	metaDataJson := utils.MarshalRawMessageSafe(metaData)
	insertEventQuery := `INSERT INTO auth.security_events
		(user_id, event_type, event_category, severity, status, description, ip_address, user_agent, device_id, location_country, location_city, created_at, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`
	_, err = tx.Exec(ctx, insertEventQuery,
		userID,
		dbModels.SecurityEventEmailVerified,
		"account_management",
		dbModels.SecuritySeverityLow,
		"success",
		"Email verified successfully",
		location.IP,
		deviceInfo.UserAgent,
		deviceInfo.ID,
		location.Country,
		location.City,
		time.Now(),
		metaDataJson,
	)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to insert security event for email verification").
			WithDetail("user_id", userID).
			WithDetail("query", insertEventQuery)
	}

	insertOutboxQuery := `INSERT INTO auth.outbox (user_id, event_type, payload, created_at) VALUES ($1, 'auth.email.verified', $2, NOW())`
	payload := map[string]interface{}{
		"user_id": userID,
		"event":   "auth.email.verified",
	}
	payloadJson, _ := json.Marshal(payload)
	_, err = tx.Exec(ctx, insertOutboxQuery, userID, payloadJson)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to insert outbox event for email verification").
			WithDetail("user_id", userID).
			WithDetail("query", insertOutboxQuery)
	}

	txErr = tx.Commit()
	if txErr != nil {
		return pkgErrors.FromError(txErr, pkgErrors.CodeDatabaseError, "failed to commit transaction for marking email verified").
			WithDetail("user_id", userID)
	}

	r.log.Info("Email marked as verified successfully",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
	)

	return nil
}
func (r *authRepository) ActivatePendingUser(ctx context.Context, userID string) pkgErrors.AppError {
	r.log.Info("Activating pending user account",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
	)

	query := `UPDATE auth.users
		SET account_status = $1,
		    updated_at = NOW()
		WHERE id = $2 AND account_status = $3`
	result, err := r.db.Exec(ctx, query, dbModels.AccountStatusActive, userID, dbModels.AccountStatusPending)
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

func (r *authRepository) Update2FASecret(ctx context.Context, userID string, secret string) pkgErrors.AppError {
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

func (r *authRepository) Enable2FA(ctx context.Context, userID string) pkgErrors.AppError {
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

func (r *authRepository) Disable2FA(ctx context.Context, userID string) pkgErrors.AppError {
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

func (r *authRepository) SoftDeleteUser(ctx context.Context, userID string) pkgErrors.AppError {
	r.log.Info("Soft deleting user",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
	)

	query := `UPDATE auth.users
		SET account_status = $1,
		    deleted_at = NOW(),
		    updated_at = NOW()
		WHERE id = $2`
	_, err := r.db.Exec(ctx, query, dbModels.AccountStatusDeleted, userID)
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

// ============================================================================
// Device Management
// ============================================================================
func (r *authRepository) IsDeviceTrusted(ctx context.Context, userID string, deviceID string) (bool, pkgErrors.AppError) {
	r.log.Debug("Checking device trust via login history",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
		logger.String("device_id", deviceID),
	)

	// Primary signal: successful login history for this device.
	var loginCount int64
	loginQuery := `SELECT COUNT(*) FROM auth.login_history WHERE user_id = $1 AND device_id = $2 AND status = 'success'`
	if err := r.db.QueryRow(ctx, loginQuery, userID, deviceID).Scan(&loginCount); err != nil {
		return false, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to query login history for device trust").
			WithDetail("user_id", userID).
			WithDetail("device_id", deviceID)
	}

	if loginCount > 0 {
		r.log.Debug("Device is trusted — successful login history found",
			logger.String("service", authErrors.ServiceName),
			logger.String("user_id", userID),
			logger.String("device_id", deviceID),
			logger.Int64("successful_logins", loginCount),
		)
		return true, nil
	}

	// Edge-case check: device may exist in users.devices (registered via a
	// different flow) but has no successful login history.
	// Uses idx_users_devices_device_id. Trust is denied regardless.
	var deviceID_ string
	deviceQuery := `SELECT id FROM users.devices WHERE user_id = $1 AND device_id = $2 AND is_active = true LIMIT 1`
	err := r.db.QueryRow(ctx, deviceQuery, userID, deviceID).Scan(&deviceID_)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to query device registration for trust check").
			WithDetail("user_id", userID).
			WithDetail("device_id", deviceID)
	}

	if deviceID_ != "" {
		r.log.Debug("Device registered but no successful login history — not trusted",
			logger.String("service", authErrors.ServiceName),
			logger.String("user_id", userID),
			logger.String("device_id", deviceID),
		)
	} else {
		r.log.Debug("Device unknown — not trusted",
			logger.String("service", authErrors.ServiceName),
			logger.String("user_id", userID),
			logger.String("device_id", deviceID),
		)
	}

	return false, nil
}

// IsNewLocation returns true when the given city has not appeared in the user's
func (r *authRepository) IsNewLocation(ctx context.Context, userID string, city string) (bool, pkgErrors.AppError) {
	r.log.Debug("Checking login location history",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
		logger.String("city", city),
	)

	if city == "" {
		return true, nil
	}

	query := `
		SELECT location_city
		FROM auth.login_history
		WHERE user_id = $1
		  AND status = 'success'
		  AND location_city IS NOT NULL
		GROUP BY location_city
		ORDER BY MAX(created_at) DESC
		LIMIT 10`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return false, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to query login location history").
			WithDetail("user_id", userID)
	}
	defer rows.Close()

	for rows.Next() {
		var knownCity string
		if scanErr := rows.Scan(&knownCity); scanErr != nil {
			return false, pkgErrors.FromError(scanErr, pkgErrors.CodeDatabaseError, "failed to scan login location city").
				WithDetail("user_id", userID)
		}
		if strings.EqualFold(knownCity, city) {
			r.log.Debug("Location is known",
				logger.String("service", authErrors.ServiceName),
				logger.String("user_id", userID),
				logger.String("city", city),
			)
			return false, nil
		}
	}

	r.log.Debug("New login location detected",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
		logger.String("city", city),
	)
	return true, nil
}

func (r *authRepository) UpdateUserDevice(ctx context.Context, userID string, deviceID string, update domain.UpdateUserDevice) (*domain.UserDevice, pkgErrors.AppError) {
	r.log.Info("Updating user device",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
		logger.String("device_id", deviceID),
	)

	var count int
	checkQuery := `SELECT COUNT(1) FROM users.devices WHERE user_id = $1 AND device_id = $2`
	err := r.db.QueryRow(ctx, checkQuery, userID, deviceID).Scan(&count)
	if err != nil && !postgres.IsNotFoundError(err) {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to check if device exists").
			WithDetail("user_id", userID).
			WithDetail("device_id", deviceID)
	}
	if count == 0 {
		r.log.Info("Device does not exist, skipping update",
			logger.String("service", authErrors.ServiceName),
			logger.String("user_id", userID),
			logger.String("device_id", deviceID),
		)
		return nil, nil
	}

	updateQuery := `UPDATE users.devices
		SET app_version = COALESCE($1, app_version),
		    fcm_token = COALESCE($2, fcm_token),
		    apns_token = COALESCE($3, apns_token),
		    push_enabled = COALESCE($4, push_enabled),
		    is_active = COALESCE($5, is_active),
			is_current_device = COALESCE($6, is_current_device),
			last_active_at = NOW()
		WHERE user_id = $7 AND device_id = $8`
	result, err := r.db.Exec(ctx, updateQuery,
		update.AppVersion,
		update.FCMToken,
		update.APNSToken,
		update.PushEnabled,
		update.IsActive,
		update.IsCurrentDevice,
		userID,
		deviceID,
	)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to update user device").
			WithDetail("user_id", userID).
			WithDetail("device_id", deviceID)
	}

	rowsAffected, _ := result.RowsAffected()
	r.log.Info("User device updated successfully",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
		logger.String("device_id", deviceID),
		logger.Int64("rows_affected", rowsAffected),
	)

	if update.IsCurrentDevice != nil && *update.IsCurrentDevice {
		unsetQuery := `UPDATE users.devices SET is_current_device = FALSE, last_active_at = NOW() WHERE user_id = $1 AND device_id != $2`
		result, err := r.db.Exec(ctx, unsetQuery, userID, deviceID)
		if err != nil {
			return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to unset current device flag for other devices").
				WithDetail("user_id", userID).
				WithDetail("device_id", deviceID)
		}
		rowsAffected, _ := result.RowsAffected()
		r.log.Info("Other devices' current device flag unset",
			logger.String("service", authErrors.ServiceName),
			logger.String("user_id", userID),
			logger.Int64("rows_affected", rowsAffected),
		)
	}

	fetchQuery := `SELECT * FROM users.devices WHERE user_id = $1 AND device_id = $2 LIMIT 1`
	row := r.db.QueryRow(ctx, fetchQuery, userID, deviceID)
	var updated dbModels.Device
	if err := row.ScanModel(&updated); err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to fetch updated user device").
			WithDetail("user_id", userID).
			WithDetail("device_id", deviceID)
	}

	r.log.Info("Fetched updated user device successfully",
		logger.String("service", authErrors.ServiceName),
		logger.String("user_id", userID),
		logger.String("device_id", deviceID),
	)
	return domain.NewUserDevice(updated), nil
}
