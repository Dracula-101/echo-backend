package repository

import (
	repoModels "auth-service/internal/repo/models"
	"context"
	"shared/pkg/database"
	"shared/pkg/database/postgres/models"
	"shared/pkg/logger"
)

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

func (r *LoginHistoryRepo) CreateLoginHistory(ctx context.Context, input repoModels.CreateLoginHistoryInput) error {
	r.log.Debug("Creating login history entry",
		logger.String("user_id", input.UserID),
		logger.String("status", safeDerefString(input.Status)),
		logger.String("ip_address", safeDerefString(&input.IPInfo.IP)),
	)
	id, err := r.db.Create(ctx, &models.LoginHistory{
		UserID:            input.UserID,
		SessionID:         input.SessionID,
		LoginMethod:       input.LoginMethod,
		Status:            input.Status,
		FailureReason:     input.FailureReason,
		IPAddress:         &input.IPInfo.IP,
		UserAgent:         input.UserAgent,
		DeviceID:          &input.DeviceInfo.ID,
		DeviceFingerprint: &input.DeviceFingerprint,
		LocationCountry:   &input.IPInfo.Country,
		LocationCity:      &input.IPInfo.City,
		Latitude:          &input.IPInfo.Latitude,
		Longitude:         &input.IPInfo.Longitude,
		IsNewDevice:       *input.IsNewDevice,
		IsNewLocation:     *input.IsNewLocation,
	})
	if err != nil {
		r.log.Error("Failed to create login history", logger.Error(err))
		return err
	}
	r.log.Debug("Login history created successfully",
		logger.String("login_history_id", id),
	)
	return nil
}

func (r *LoginHistoryRepo) GetLoginHistoryByUserID(ctx context.Context, userID string, limit int) ([]*models.LoginHistory, error) {
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
		r.log.Error("Failed to get login history by user ID", logger.Error(err))
		return nil, err
	}
	r.log.Debug("Login history fetched successfully",
		logger.String("user_id", userID),
		logger.Int("count", len(histories)),
	)
	return histories, nil
}

func (r *LoginHistoryRepo) GetLoginHistoryByID(ctx context.Context, id string) (*models.LoginHistory, error) {
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
		r.log.Error("Failed to get login history by ID", logger.Error(err))
		return nil, err
	}
	r.log.Debug("Login history fetched successfully",
		logger.String("login_history_id", history.ID),
	)
	return &history, nil
}

func (r *LoginHistoryRepo) GetFailedLoginAttempts(ctx context.Context, userID string, duration string) (int, error) {
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
		r.log.Error("Failed to count failed login attempts", logger.Error(err))
		return 0, err
	}
	r.log.Debug("Failed login attempts counted",
		logger.String("user_id", userID),
		logger.Int("count", count),
	)
	return count, nil
}

func (r *LoginHistoryRepo) DeleteLoginHistoryByUserID(ctx context.Context, userID string) error {
	r.log.Debug("Deleting login history by user ID",
		logger.String("user_id", userID),
	)
	query := `DELETE FROM auth.login_history WHERE user_id = $1`
	result, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		r.log.Error("Failed to delete login history", logger.Error(err))
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	r.log.Debug("Login history deleted successfully",
		logger.String("user_id", userID),
		logger.Int64("rows_affected", rowsAffected),
	)
	return nil
}

func (r *LoginHistoryRepo) DeleteLoginHistoryByID(ctx context.Context, id string) error {
	r.log.Debug("Deleting login history by ID",
		logger.String("login_history_id", id),
	)
	query := `DELETE FROM auth.login_history WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		r.log.Error("Failed to delete login history", logger.Error(err))
		return err
	}
	r.log.Debug("Login history deleted successfully",
		logger.String("login_history_id", id),
	)
	return nil
}

// Helper function to safely dereference string pointers for logging
func safeDerefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
