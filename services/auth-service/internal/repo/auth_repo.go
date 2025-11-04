package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"shared/pkg/database"
	"shared/pkg/database/postgres/models"
	"shared/pkg/logger"
)

type AuthRepository struct {
	db  database.Database
	log logger.Logger
}

func NewAuthRepository(db database.Database, log logger.Logger) *AuthRepository {
	return &AuthRepository{
		db:  db,
		log: log,
	}
}

func (r *AuthRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM auth.users WHERE email = $1)`
	var exists bool
	err := r.db.QueryRow(ctx, query, email).Scan(&exists)
	if err != nil {
		r.log.Error("Failed to check if user exists", logger.Error(err))
		return false, err
	}

	return exists, nil
}

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

func (r *AuthRepository) CreateUser(ctx context.Context, params CreateUserParams) (string, error) {
	passwordHistory := fmt.Sprintf(`[{"hash":"%s","salt":"%s","algorithm":"%s","changed_at":"%s"}]`,
		params.PasswordHash,
		params.PasswordSalt,
		params.PasswordAlgorithm,
		time.Now().Format(time.RFC3339),
	)
	passwordHistoryJson := json.RawMessage(passwordHistory)

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
		AccountStatus:          "active",
		AccountLockedUntil:     nil,
		FailedLoginAttempts:    0,
		LastFailedLoginAt:      nil,
		RequiresPasswordChange: false,
		DeletedAt:              nil,
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
		PasswordHistory:        passwordHistoryJson,
		CreatedByIP:            &params.IPAddress,
		CreatedByUserAgent:     &params.UserAgent,
	})
	if err != nil {
		r.log.Error("Failed to create user", logger.Error(err))
		return "", err
	}

	r.log.Debug("User created successfully",
		logger.String("user_id", id),
		logger.String("email", params.Email),
	)

	return id, nil
}

var ErrAuthUserNotFound = errors.New("auth user not found")

func (r *AuthRepository) GetUserByEmail(ctx context.Context, email string) (*models.AuthUser, error) {
	query := `SELECT  * FROM auth.users WHERE email = $1 LIMIT 1`
	row := r.db.QueryRow(ctx, query, email)
	var user models.AuthUser
	err := row.ScanOne(&user)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAuthUserNotFound
		}
		r.log.Error("Failed to get user by email", logger.Error(err))
		return nil, err
	}

	return &user, nil
}

func (r *AuthRepository) RecordFailedLogin(ctx context.Context, userID string) error {
	query := `UPDATE auth.users 
		SET failed_login_attempts = failed_login_attempts + 1, 
		    last_failed_login_at = NOW(), 
		    updated_at = NOW() 
		WHERE id = $1`
	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		r.log.Error("Failed to record failed login", logger.Error(err))
		return err
	}
	r.log.Debug("Failed login recorded successfully",
		logger.String("user_id", userID),
	)
	return nil
}

func (r *AuthRepository) RecordSuccessfulLogin(ctx context.Context, userID string) error {
	query := `UPDATE auth.users 
		SET failed_login_attempts = 0, 
		    last_failed_login_at = NULL, 
		    updated_at = NOW() 
		WHERE id = $1`
	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		r.log.Error("Failed to record successful login", logger.Error(err))
		return err
	}
	r.log.Debug("Successful login recorded successfully",
		logger.String("user_id", userID),
	)
	return nil
}
