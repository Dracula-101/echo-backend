package repository

import (
	"auth-service/internal/domain"
	"context"
	"encoding/json"
	"shared/pkg/database"
	"shared/pkg/database/postgres"
	"shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/pkg/utils"
	"time"
)

// SessionRepositoryInterface defines the contract for session repository operations
type SessionRepositoryInterface interface {
	// Session management
	CreateSession(ctx context.Context, session *domain.Session) pkgErrors.AppError
	GetSessionByUserId(ctx context.Context, userID string, deviceID *string) (*domain.Session, pkgErrors.AppError)
	GetAllSessionsByUserId(ctx context.Context, userID string) ([]*domain.Session, pkgErrors.AppError)
	GetSessionByID(ctx context.Context, sessionID string) (*domain.Session, pkgErrors.AppError)
	UpdateSession(ctx context.Context, session *domain.UpdateAuthSession) (*domain.Session, pkgErrors.AppError)
	DeleteSessionByID(ctx context.Context, sessionID string) pkgErrors.AppError
	RevokeSession(ctx context.Context, sessionID string, reason string) pkgErrors.AppError
	RevokeAllUserSessions(ctx context.Context, userID string, reason string) pkgErrors.AppError
}

// ============================================================================
// Repository Definition
// ============================================================================

type SessionRepo struct {
	db  database.Database
	log logger.Logger
}

func NewSessionRepo(db database.Database, log logger.Logger) *SessionRepo {
	return &SessionRepo{
		db:  db,
		log: log,
	}
}

// ============================================================================
// Session Management
// ============================================================================

func (r *SessionRepo) CreateSession(ctx context.Context, session *domain.Session) pkgErrors.AppError {
	r.log.Debug("Creating new session",
		logger.String("user_id", session.UserID),
		logger.String("device_id", session.DeviceID),
		logger.String("ip_address", session.IPAddress),
	)
	_, err := r.db.Insert(ctx, &models.AuthSession{
		ID:                 session.ID,
		SessionToken:       session.SessionToken,
		UserID:             session.UserID,
		RefreshToken:       &session.RefreshToken,
		DeviceID:           &session.DeviceID,
		DeviceType:         &session.DeviceType,
		DeviceName:         &session.DeviceName,
		IPAddress:          session.IPAddress,
		CreatedAt:          session.CreatedAt,
		ExpiresAt:          session.ExpiresAt,
		DeviceOS:           &session.DeviceOS,
		BrowserName:        &session.BrowserName,
		BrowserVersion:     &session.BrowserVersion,
		IPCity:             &session.City,
		IPRegion:           &session.Region,
		IPCountry:          &session.Country,
		IPTimezone:         &session.Timezone,
		Latitude:           session.Latitude,
		Longitude:          session.Longitude,
		IsTrustedDevice:    session.IsTrustedDevice,
		IsMobile:           session.IsMobile,
		SessionType:        session.SessionType,
		UserAgent:          &session.UserAgent,
		DeviceOSVersion:    &session.DeviceOSVersion,
		DeviceModel:        &session.DeviceModel,
		DeviceManufacturer: &session.DeviceManufacturer,
		IPISP:              &session.IPISP,
		FCMToken:           &session.FCMToken,
		APNSToken:          &session.APNSToken,
		PushEnabled:        true,
		RevokedAt:          nil,
		LastActivityAt:     session.LastActivityAt,
		LastRefreshAt:      utils.Ptr(time.Now()),
		RevokedReason:      nil,
		Metadata:           utils.Ptr(json.RawMessage("{}")),
	})

	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to create session").
			WithDetail("user_id", session.UserID).
			WithDetail("device_id", session.DeviceID)
	}
	r.log.Debug("Session created successfully",
		logger.String("session_id", session.ID),
	)
	return nil
}

func (r *SessionRepo) GetAllSessionsByUserId(ctx context.Context, userID string) ([]*domain.Session, pkgErrors.AppError) {
	r.log.Debug("Fetching all sessions by user ID",
		logger.String("user_id", userID),
	)
	var sessions []*models.AuthSession
	query := "SELECT * FROM auth.sessions WHERE user_id = $1 ORDER BY created_at DESC"
	rows, err := r.db.Query(ctx, query, userID)

	for rows.Next() {
		var session models.AuthSession
		err := rows.ScanModel(&session)
		if err != nil {
			return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to scan session row").
				WithDetail("user_id", userID)
		}
		sessions = append(sessions, &session)
	}

	if err = rows.Err(); err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "error iterating session rows").
			WithDetail("user_id", userID)
	}

	r.log.Debug("Sessions fetched successfully",
		logger.String("user_id", userID),
		logger.Int("session_count", len(sessions)),
	)
	var result []*domain.Session
	for _, session := range sessions {
		metaData := make(map[string]interface{})
		json.Unmarshal(*session.Metadata, &metaData)
		result = append(result, &domain.Session{
			ID:              session.ID,
			SessionToken:    session.SessionToken,
			UserID:          session.UserID,
			RefreshToken:    utils.DerefString(session.RefreshToken),
			DeviceID:        utils.DerefString(session.DeviceID),
			DeviceType:      utils.DerefString(session.DeviceType),
			DeviceName:      utils.DerefString(session.DeviceName),
			IPAddress:       session.IPAddress,
			CreatedAt:       session.CreatedAt,
			ExpiresAt:       session.ExpiresAt,
			DeviceOS:        utils.DerefString(session.DeviceOS),
			BrowserName:     utils.DerefString(session.BrowserName),
			BrowserVersion:  utils.DerefString(session.BrowserVersion),
			City:            utils.DerefString(session.IPCity),
			Region:          utils.DerefString(session.IPRegion),
			Country:         utils.DerefString(session.IPCountry),
			Timezone:        utils.DerefString(session.IPTimezone),
			Latitude:        session.Latitude,
			Longitude:       session.Longitude,
			IsTrustedDevice: session.IsTrustedDevice,
			IsMobile:        session.IsMobile,
			SessionType:     models.SessionTypeMobile,
			UserAgent:       utils.DerefString(session.UserAgent),
			LastActivityAt:  session.LastActivityAt,
			Metadata:        metaData,
		})
	}
	return result, nil
}

func (r *SessionRepo) UpdateSession(ctx context.Context, session *domain.UpdateAuthSession) (*domain.Session, pkgErrors.AppError) {
	r.log.Debug("Updating session",
		logger.String("session_id", session.ID),
		logger.Any("updates", session),
	)
	updateQuery := `UPDATE auth.sessions SET
		refresh_token = COALESCE($1, refresh_token),
		fcm_token = COALESCE($2, fcm_token),
		apns_token = COALESCE($3, apns_token),
		revoked_at = COALESCE($4, revoked_at),
		revoked_reason = COALESCE($5, revoked_reason),
		push_enabled = COALESCE($6, push_enabled),
		last_activity_at = $7
		WHERE id = $8 RETURNING *`

	var updatedSession models.AuthSession
	err := r.db.QueryRow(ctx, updateQuery,
		session.RefreshToken,
		session.FCMToken,
		session.APNSToken,
		session.RevokedAt,
		session.RevokedReason,
		session.PushEnabled,
		time.Now(),
		session.ID,
	).ScanModel(&updatedSession)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to update session").
			WithDetail("session_id", session.ID)
	}

	metaData := make(map[string]interface{})
	json.Unmarshal(*updatedSession.Metadata, &metaData)

	r.log.Debug("Session updated successfully",
		logger.String("session_id", session.ID),
	)
	return &domain.Session{
		ID:                 updatedSession.ID,
		SessionToken:       updatedSession.SessionToken,
		UserID:             updatedSession.UserID,
		RefreshToken:       utils.DerefString(updatedSession.RefreshToken),
		DeviceID:           utils.DerefString(updatedSession.DeviceID),
		DeviceType:         utils.DerefString(updatedSession.DeviceType),
		DeviceName:         utils.DerefString(updatedSession.DeviceName),
		IPAddress:          updatedSession.IPAddress,
		CreatedAt:          updatedSession.CreatedAt,
		ExpiresAt:          updatedSession.ExpiresAt,
		DeviceOS:           utils.DerefString(updatedSession.DeviceOS),
		BrowserName:        utils.DerefString(updatedSession.BrowserName),
		BrowserVersion:     utils.DerefString(updatedSession.BrowserVersion),
		City:               utils.DerefString(updatedSession.IPCity),
		Region:             utils.DerefString(updatedSession.IPRegion),
		Country:            utils.DerefString(updatedSession.IPCountry),
		Timezone:           utils.DerefString(updatedSession.IPTimezone),
		Latitude:           updatedSession.Latitude,
		Longitude:          updatedSession.Longitude,
		IsTrustedDevice:    updatedSession.IsTrustedDevice,
		IsMobile:           updatedSession.IsMobile,
		SessionType:        updatedSession.SessionType,
		UserAgent:          utils.DerefString(updatedSession.UserAgent),
		LastActivityAt:     updatedSession.LastActivityAt,
		Metadata:           metaData,
		DeviceOSVersion:    utils.DerefString(updatedSession.DeviceOSVersion),
		DeviceModel:        utils.DerefString(updatedSession.DeviceModel),
		DeviceManufacturer: utils.DerefString(updatedSession.DeviceManufacturer),
		IPISP:              utils.DerefString(updatedSession.IPISP),
		FCMToken:           utils.DerefString(updatedSession.FCMToken),
		APNSToken:          utils.DerefString(updatedSession.APNSToken),
		PushEnabled:        updatedSession.PushEnabled,
		RevokedAt:          updatedSession.RevokedAt,
		RevokedReason:      updatedSession.RevokedReason,
	}, nil
}

func (r *SessionRepo) GetSessionByUserId(ctx context.Context, userID string, deviceID *string) (*domain.Session, pkgErrors.AppError) {
	r.log.Debug("Fetching session by user ID",
		logger.String("user_id", userID),
	)
	var session models.AuthSession
	query := `SELECT * FROM auth.sessions WHERE user_id = $1 AND coalesce(device_id, '') = coalesce($2, '') LIMIT 1`
	err := r.db.QueryRow(ctx, query, userID, deviceID).ScanModel(&session)
	if err != nil {
		if postgres.IsNotFoundError(err) {
			r.log.Debug("No active session found for user",
				logger.String("user_id", userID),
			)
			return nil, nil
		}
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get session by user ID").
			WithDetail("user_id", userID)
	}
	return &domain.Session{
		ID:                 session.ID,
		SessionToken:       session.SessionToken,
		UserID:             session.UserID,
		RefreshToken:       utils.DerefString(session.RefreshToken),
		DeviceID:           utils.DerefString(session.DeviceID),
		DeviceType:         utils.DerefString(session.DeviceType),
		DeviceName:         utils.DerefString(session.DeviceName),
		IPAddress:          session.IPAddress,
		CreatedAt:          session.CreatedAt,
		ExpiresAt:          session.ExpiresAt,
		DeviceOS:           utils.DerefString(session.DeviceOS),
		BrowserName:        utils.DerefString(session.BrowserName),
		BrowserVersion:     utils.DerefString(session.BrowserVersion),
		City:               utils.DerefString(session.IPCity),
		Region:             utils.DerefString(session.IPRegion),
		Country:            utils.DerefString(session.IPCountry),
		Timezone:           utils.DerefString(session.IPTimezone),
		Latitude:           session.Latitude,
		Longitude:          session.Longitude,
		IsTrustedDevice:    session.IsTrustedDevice,
		IsMobile:           session.IsMobile,
		SessionType:        session.SessionType,
		UserAgent:          utils.DerefString(session.UserAgent),
		LastActivityAt:     session.LastActivityAt,
		DeviceOSVersion:    utils.DerefString(session.DeviceOSVersion),
		DeviceModel:        utils.DerefString(session.DeviceModel),
		DeviceManufacturer: utils.DerefString(session.DeviceManufacturer),
		IPISP:              utils.DerefString(session.IPISP),
		FCMToken:           utils.DerefString(session.FCMToken),
		APNSToken:          utils.DerefString(session.APNSToken),
		PushEnabled:        session.PushEnabled,
		RevokedAt:          session.RevokedAt,
		RevokedReason:      session.RevokedReason,
		Metadata: map[string]interface{}{
			"created_at":      session.CreatedAt,
			"last_refresh_at": session.LastRefreshAt,
		},
	}, nil
}

func (r *SessionRepo) DeleteSessionByID(ctx context.Context, sessionID string) pkgErrors.AppError {
	r.log.Debug("Deleting session",
		logger.String("session_id", sessionID),
	)
	query := `DELETE FROM auth.sessions WHERE id = $1`
	_, err := r.db.Exec(ctx, query, sessionID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to delete session").
			WithDetail("session_id", sessionID)
	}
	r.log.Debug("Session deleted successfully",
		logger.String("session_id", sessionID),
	)
	return nil
}

func (r *SessionRepo) GetSessionByID(ctx context.Context, sessionID string) (*domain.Session, pkgErrors.AppError) {
	r.log.Debug("Fetching session by ID",
		logger.String("session_id", sessionID),
	)
	var session models.AuthSession
	query := `SELECT * FROM auth.sessions WHERE id = $1 LIMIT 1`
	err := r.db.QueryRow(ctx, query, sessionID).ScanModel(&session)
	if err != nil {
		if postgres.IsNotFoundError(err) {
			return nil, nil
		}
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get session by ID").
			WithDetail("session_id", sessionID)
	}

	metaData := make(map[string]interface{})
	if session.Metadata != nil {
		json.Unmarshal(*session.Metadata, &metaData)
	}

	return &domain.Session{
		ID:                 session.ID,
		SessionToken:       session.SessionToken,
		UserID:             session.UserID,
		RefreshToken:       utils.DerefString(session.RefreshToken),
		DeviceID:           utils.DerefString(session.DeviceID),
		DeviceType:         utils.DerefString(session.DeviceType),
		DeviceName:         utils.DerefString(session.DeviceName),
		IPAddress:          session.IPAddress,
		CreatedAt:          session.CreatedAt,
		ExpiresAt:          session.ExpiresAt,
		DeviceOS:           utils.DerefString(session.DeviceOS),
		BrowserName:        utils.DerefString(session.BrowserName),
		BrowserVersion:     utils.DerefString(session.BrowserVersion),
		City:               utils.DerefString(session.IPCity),
		Region:             utils.DerefString(session.IPRegion),
		Country:            utils.DerefString(session.IPCountry),
		Timezone:           utils.DerefString(session.IPTimezone),
		Latitude:           session.Latitude,
		Longitude:          session.Longitude,
		IsTrustedDevice:    session.IsTrustedDevice,
		IsMobile:           session.IsMobile,
		SessionType:        session.SessionType,
		UserAgent:          utils.DerefString(session.UserAgent),
		LastActivityAt:     session.LastActivityAt,
		DeviceOSVersion:    utils.DerefString(session.DeviceOSVersion),
		DeviceModel:        utils.DerefString(session.DeviceModel),
		DeviceManufacturer: utils.DerefString(session.DeviceManufacturer),
		IPISP:              utils.DerefString(session.IPISP),
		FCMToken:           utils.DerefString(session.FCMToken),
		APNSToken:          utils.DerefString(session.APNSToken),
		PushEnabled:        session.PushEnabled,
		RevokedAt:          session.RevokedAt,
		RevokedReason:      session.RevokedReason,
		Metadata:           metaData,
	}, nil
}

func (r *SessionRepo) RevokeSession(ctx context.Context, sessionID string, reason string) pkgErrors.AppError {
	r.log.Info("Revoking session",
		logger.String("session_id", sessionID),
		logger.String("reason", reason),
	)
	query := `UPDATE auth.sessions
		SET revoked_at = NOW(),
		    revoked_reason = $1
		WHERE id = $2 AND revoked_at IS NULL`
	_, err := r.db.Exec(ctx, query, reason, sessionID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to revoke session").
			WithDetail("session_id", sessionID)
	}
	r.log.Info("Session revoked",
		logger.String("session_id", sessionID),
	)
	return nil
}

func (r *SessionRepo) RevokeAllUserSessions(ctx context.Context, userID string, reason string) pkgErrors.AppError {
	r.log.Info("Revoking all sessions for user",
		logger.String("user_id", userID),
		logger.String("reason", reason),
	)
	query := `UPDATE auth.sessions
		SET revoked_at = NOW(),
		    revoked_reason = $1
		WHERE user_id = $2 AND revoked_at IS NULL`
	result, err := r.db.Exec(ctx, query, reason, userID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to revoke all user sessions").
			WithDetail("user_id", userID)
	}
	rowsAffected, _ := result.RowsAffected()
	r.log.Info("All user sessions revoked",
		logger.String("user_id", userID),
		logger.Int64("sessions_revoked", rowsAffected),
	)
	return nil
}
