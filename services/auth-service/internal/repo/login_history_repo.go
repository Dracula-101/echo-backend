package repository

import (
	repoModels "auth-service/internal/repo/models"
	"context"
	"shared/pkg/database"
	"shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
)

// ============================================================================
// Repository Definition
// ============================================================================

type LoginHistoryRepo struct {
	db  database.Database
	log logger.Logger
}

func NewLoginHistoryRepo(db database.Database, log logger.Logger) *LoginHistoryRepo {
	return &LoginHistoryRepo{
		db:  db,
		log: log,
	}
}

// ============================================================================
// Login History Operations
// ============================================================================

func (r *LoginHistoryRepo) CreateLoginHistory(ctx context.Context, input repoModels.CreateLoginHistoryInput) pkgErrors.AppError {
	r.log.Debug("Creating login history entry",
		logger.String("user_id", input.UserID),
		logger.String("status", safeDerefString(input.Status)),
		logger.String("ip_address", safeDerefString(&input.IPInfo.IP)),
	)
	id, err := r.db.Insert(ctx, &models.LoginHistory{
		UserID:          input.UserID,
		SessionID:       input.SessionID,
		LoginMethod:     input.LoginMethod,
		Status:          input.Status,
		FailureReason:   input.FailureReason,
		IPAddress:       &input.IPInfo.IP,
		UserAgent:       input.UserAgent,
		DeviceID:        &input.DeviceInfo.ID,
		LocationCountry: &input.IPInfo.Country,
		LocationCity:    &input.IPInfo.City,
		Latitude:        &input.IPInfo.Latitude,
		Longitude:       &input.IPInfo.Longitude,
		IsNewDevice:     *input.IsNewDevice,
		IsNewLocation:   *input.IsNewLocation,
	})
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to create login history").
			WithDetail("user_id", input.UserID).
			WithDetail("status", safeDerefString(input.Status))
	}
	r.log.Debug("Login history created successfully",
		logger.String("login_history_id", *id),
	)
	return nil
}

func (r *LoginHistoryRepo) GetLoginHistoryByUserID(ctx context.Context, userID string, limit int) ([]*models.LoginHistory, pkgErrors.AppError) {
	r.log.Debug("Fetching login history by user ID",
		logger.String("user_id", userID),
		logger.Int("limit", limit),
	)
	var histories []*models.LoginHistory
	query := `SELECT id, user_id, session_id, login_method, status, failure_reason, 
		ip_address, user_agent, device_id, device_fingerprint, location_country, 
		location_city, latitude, longitude, is_new_device, is_new_location, created_at 
		FROM auth.login_history 
		WHERE user_id = $1 
		ORDER BY created_at DESC 
		LIMIT $2`
	err := r.db.FindMany(ctx, &histories, query, userID, limit)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get login history by user ID").
			WithDetail("user_id", userID).
			WithDetail("limit", limit)
	}
	r.log.Debug("Login history fetched successfully",
		logger.String("user_id", userID),
		logger.Int("count", len(histories)),
	)
	return histories, nil
}

func (r *LoginHistoryRepo) GetLoginHistoryByID(ctx context.Context, id string) (*models.LoginHistory, pkgErrors.AppError) {
	r.log.Debug("Fetching login history by ID",
		logger.String("login_history_id", id),
	)
	var history models.LoginHistory
	query := `SELECT id, user_id, session_id, login_method, status, failure_reason, 
		ip_address, user_agent, device_id, device_fingerprint, location_country, 
		location_city, latitude, longitude, is_new_device, is_new_location, created_at 
		FROM auth.login_history 
		WHERE id = $1 
		LIMIT 1`
	err := r.db.FindOne(ctx, &history, query, id)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get login history by ID").
			WithDetail("login_history_id", id)
	}
	r.log.Debug("Login history fetched successfully",
		logger.String("login_history_id", history.ID),
	)
	return &history, nil
}

func (r *LoginHistoryRepo) GetFailedLoginAttempts(ctx context.Context, userID string, duration string) (int, pkgErrors.AppError) {
	r.log.Debug("Counting failed login attempts",
		logger.String("user_id", userID),
		logger.String("duration", duration),
	)
	var count int
	query := `SELECT COUNT(*) 
		FROM auth.login_history 
		WHERE user_id = $1 
		AND status = 'failed' 
		AND created_at > NOW() - INTERVAL '1 ' || $2`
	err := r.db.QueryRow(ctx, query, userID, duration).Scan(&count)
	if err != nil {
		return 0, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to count failed login attempts").
			WithDetail("user_id", userID).
			WithDetail("duration", duration)
	}
	r.log.Debug("Failed login attempts counted",
		logger.String("user_id", userID),
		logger.Int("count", count),
	)
	return count, nil
}

func (r *LoginHistoryRepo) DeleteLoginHistoryByUserID(ctx context.Context, userID string) pkgErrors.AppError {
	r.log.Debug("Deleting login history by user ID",
		logger.String("user_id", userID),
	)
	query := `DELETE FROM auth.login_history WHERE user_id = $1`
	result, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to delete login history").
			WithDetail("user_id", userID)
	}
	rowsAffected, _ := result.RowsAffected()
	r.log.Debug("Login history deleted successfully",
		logger.String("user_id", userID),
		logger.Int64("rows_affected", rowsAffected),
	)
	return nil
}

func (r *LoginHistoryRepo) DeleteLoginHistoryByID(ctx context.Context, id string) pkgErrors.AppError {
	r.log.Debug("Deleting login history by ID",
		logger.String("login_history_id", id),
	)
	query := `DELETE FROM auth.login_history WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to delete login history").
			WithDetail("login_history_id", id)
	}
	r.log.Debug("Login history deleted successfully",
		logger.String("login_history_id", id),
	)
	return nil
}

// ============================================================================
// Helper Functions
// ============================================================================

func safeDerefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
