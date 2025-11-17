package repo

import (
	"context"

	"presence-service/internal/model"

	"shared/pkg/database"
	"shared/pkg/logger"

	pkgErrors "shared/pkg/errors"

	"github.com/google/uuid"
)

type PresenceRepository interface {
	UserExists(ctx context.Context, userID uuid.UUID) (bool, pkgErrors.AppError)
	UpdatePresence(ctx context.Context, update *model.PresenceUpdate) pkgErrors.AppError
	GetPresence(ctx context.Context, userID uuid.UUID) (*model.UserPresence, pkgErrors.AppError)
	GetBulkPresence(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*model.UserPresence, pkgErrors.AppError)
	UpdateHeartbeat(ctx context.Context, userID uuid.UUID, deviceID string) pkgErrors.AppError
	GetActiveDevices(ctx context.Context, userID uuid.UUID) ([]*model.Device, pkgErrors.AppError)
	SetTypingIndicator(ctx context.Context, indicator *model.TypingIndicator) pkgErrors.AppError
	GetTypingIndicators(ctx context.Context, conversationID uuid.UUID) ([]*model.TypingIndicator, pkgErrors.AppError)
	GetPrivacySettings(ctx context.Context, userID uuid.UUID) (*model.PresencePrivacy, pkgErrors.AppError)
}

type presenceRepo struct {
	db  database.Database
	log logger.Logger
}

func NewPresenceRepository(db database.Database, log logger.Logger) PresenceRepository {
	return &presenceRepo{
		db:  db,
		log: log,
	}
}

func (r *presenceRepo) UserExists(ctx context.Context, userID uuid.UUID) (bool, pkgErrors.AppError) {
	query := `
		SELECT EXISTS(
			SELECT 1 
			FROM users.profiles 
			WHERE user_id = $1
		)
	`

	var exists bool
	err := r.db.QueryRow(ctx, query, userID).Scan(&exists)
	if err != nil {
		r.log.Error("Failed to check if user exists",
			logger.Error(err),
			logger.String("user_id", userID.String()),
		)
		return false, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to check user existence")
	}

	r.log.Debug("User existence check",
		logger.String("user_id", userID.String()),
		logger.Bool("exists", exists),
	)

	return exists, nil
}

func (r *presenceRepo) UpdatePresence(ctx context.Context, update *model.PresenceUpdate) pkgErrors.AppError {
	query := `
		UPDATE users.profiles
		SET online_status = $1,
		    last_seen_at = NOW(),
		    updated_at = NOW()
		WHERE user_id = $2
	`

	_, err := r.db.Exec(ctx, query, update.OnlineStatus, update.UserID)
	if err != nil {
		r.log.Error("Failed to update presence",
			logger.Error(err),
			logger.String("user_id", update.UserID.String()),
		)
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to update presence")
	}

	r.log.Debug("Presence updated",
		logger.String("user_id", update.UserID.String()),
		logger.String("status", update.OnlineStatus),
	)

	return nil
}

func (r *presenceRepo) GetPresence(ctx context.Context, userID uuid.UUID) (*model.UserPresence, pkgErrors.AppError) {
	query := `
		SELECT user_id, online_status, last_seen_at, updated_at
		FROM users.profiles
		WHERE user_id = $1
	`

	var presence model.UserPresence
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&presence.UserID,
		&presence.OnlineStatus,
		&presence.LastSeenAt,
		&presence.UpdatedAt,
	)

	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get presence")
	}

	return &presence, nil
}

func (r *presenceRepo) GetBulkPresence(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*model.UserPresence, pkgErrors.AppError) {
	if len(userIDs) == 0 {
		return make(map[uuid.UUID]*model.UserPresence), nil
	}

	query := `
		SELECT user_id, online_status, last_seen_at, updated_at
		FROM users.profiles
		WHERE user_id = ANY($1)
	`

	rows, err := r.db.Query(ctx, query, userIDs)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get bulk presence")
	}
	defer rows.Close()

	presences := make(map[uuid.UUID]*model.UserPresence)
	for rows.Next() {
		var presence model.UserPresence
		if err := rows.Scan(
			&presence.UserID,
			&presence.OnlineStatus,
			&presence.LastSeenAt,
			&presence.UpdatedAt,
		); err != nil {
			r.log.Error("Failed to scan presence", logger.Error(err))
			continue
		}
		presences[presence.UserID] = &presence
	}

	return presences, nil
}

func (r *presenceRepo) UpdateHeartbeat(ctx context.Context, userID uuid.UUID, deviceID string) pkgErrors.AppError {
	// Update device last_active_at
	query := `
		UPDATE users.devices
		SET last_active_at = NOW(),
		    is_active = TRUE
		WHERE user_id = $1 AND device_id = $2
	`

	_, err := r.db.Exec(ctx, query, userID, deviceID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to update heartbeat")
	}

	// Update user profile last_seen_at if online
	updatePresenceQuery := `
		UPDATE users.profiles
		SET last_seen_at = NOW(),
		    updated_at = NOW()
		WHERE user_id = $1 AND online_status != 'offline'
	`

	_, err = r.db.Exec(ctx, updatePresenceQuery, userID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to update presence heartbeat")
	}

	return nil
}

func (r *presenceRepo) GetActiveDevices(ctx context.Context, userID uuid.UUID) ([]*model.Device, pkgErrors.AppError) {
	query := `
		SELECT id, user_id, device_id, device_name, device_type,
		       app_version, is_active, last_active_at, registered_at,
		       fcm_token, apns_token, push_enabled
		FROM users.devices
		WHERE user_id = $1
		  AND is_active = TRUE
		  AND last_active_at > NOW() - INTERVAL '5 minutes'
		ORDER BY last_active_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get active devices")
	}
	defer rows.Close()

	var devices []*model.Device
	for rows.Next() {
		var device model.Device
		var fcmToken, apnsToken *string
		if err := rows.Scan(
			&device.ID,
			&device.UserID,
			&device.DeviceID,
			&device.DeviceName,
			&device.DeviceType,
			&device.AppVersion,
			&device.IsActive,
			&device.LastActiveAt,
			&device.RegisteredAt,
			&fcmToken,
			&apnsToken,
			&device.PushEnabled,
		); err != nil {
			r.log.Error("Failed to scan device", logger.Error(err))
			continue
		}

		if fcmToken != nil {
			device.FCMToken = *fcmToken
		}
		if apnsToken != nil {
			device.APNSToken = *apnsToken
		}

		devices = append(devices, &device)
	}

	return devices, nil
}

func (r *presenceRepo) SetTypingIndicator(ctx context.Context, indicator *model.TypingIndicator) pkgErrors.AppError {
	// For now, we'll use a simple approach - you may want to create a separate typing_indicators table
	// This is a placeholder that could use Redis for better performance
	r.log.Debug("Typing indicator set",
		logger.String("user_id", indicator.UserID.String()),
		logger.String("conversation_id", indicator.ConversationID.String()),
		logger.Any("is_typing", indicator.IsTyping),
	)

	// TODO: Implement with Redis or dedicated table
	return nil
}

func (r *presenceRepo) GetTypingIndicators(ctx context.Context, conversationID uuid.UUID) ([]*model.TypingIndicator, pkgErrors.AppError) {
	// TODO: Implement with Redis or dedicated table
	return []*model.TypingIndicator{}, nil
}

func (r *presenceRepo) GetPrivacySettings(ctx context.Context, userID uuid.UUID) (*model.PresencePrivacy, pkgErrors.AppError) {
	query := `
		SELECT user_id, last_seen_visibility, online_status_visibility,
		       typing_indicators_enabled, read_receipts_enabled
		FROM users.settings
		WHERE user_id = $1
	`

	var privacy model.PresencePrivacy
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&privacy.UserID,
		&privacy.LastSeenVisibility,
		&privacy.OnlineStatusVisibility,
		&privacy.TypingIndicatorsEnabled,
		&privacy.ReadReceiptsEnabled,
	)

	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get privacy settings")
	}

	return &privacy, nil
}
