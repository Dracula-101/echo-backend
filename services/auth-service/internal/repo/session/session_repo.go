package session

import (
	"auth-service/internal/domain"
	"context"
	"encoding/json"
	"shared/pkg/database/postgres"
	"shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/pkg/utils"
	"time"
)

// SessionRepositoryInterface defines the contract for session repository operations
type SessionRepository interface {
	// Session management
	CreateSession(ctx context.Context, session *domain.Session) pkgErrors.AppError
	GetSessionByUserId(ctx context.Context, userID string, deviceID *string) (*domain.Session, pkgErrors.AppError)
	GetSessionByRefreshToken(ctx context.Context, refreshToken string) (*domain.Session, pkgErrors.AppError)
	GetAllSessionsByUserId(ctx context.Context, userID string, limit int, offset int) (sessions []*domain.Session, total int, err pkgErrors.AppError)
	GetSessionByID(ctx context.Context, sessionID string) (*domain.Session, pkgErrors.AppError)
	UpdateSession(ctx context.Context, session *domain.UpdateAuthSession) (*domain.Session, pkgErrors.AppError)
	DeleteSessionByID(ctx context.Context, sessionID string) pkgErrors.AppError
	CountActiveSessions(ctx context.Context, userID string) (int64, pkgErrors.AppError)
	EvictOldestSession(ctx context.Context, userID string) (*domain.Session, pkgErrors.AppError)
	RevokeSession(ctx context.Context, sessionID string, reason string) pkgErrors.AppError
	ClearNotificationsForSession(ctx context.Context, sessionID string) pkgErrors.AppError
	ClearNotificationsForAllSessions(ctx context.Context, userID string) pkgErrors.AppError
	RevokeSessionByRefreshToken(ctx context.Context, refreshToken string, reason string) pkgErrors.AppError
	RevokeAllUserSessions(ctx context.Context, userID string, reason string) (expiredTokens *[]string, err pkgErrors.AppError)
	RevokeActiveDeviceSession(ctx context.Context, userID, deviceID, reason string) (string, pkgErrors.AppError)
}

// ============================================================================
// Session Management
// ============================================================================

func (r *sessionRepository) CreateSession(ctx context.Context, session *domain.Session) pkgErrors.AppError {
	r.log.Debug("Creating new session",
		logger.String("user_id", session.UserID),
		logger.String("device_id", session.DeviceID),
		logger.String("ip_address", session.IPAddress),
	)
	_, err := r.db.Insert(ctx, session.ToSessionModel(session.Metadata))
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

func (r *sessionRepository) GetAllSessionsByUserId(ctx context.Context, userID string, limit int, offset int) (sessions []*domain.Session, total int, err pkgErrors.AppError) {
	r.log.Debug("Fetching all sessions by user ID",
		logger.String("user_id", userID),
	)

	query := `
		WITH session_count AS (
			SELECT COUNT(*) as total FROM auth.sessions 
			WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
		)
		SELECT s.*, sc.total
		FROM auth.sessions s
		CROSS JOIN session_count sc
		WHERE s.user_id = $1 AND s.revoked_at IS NULL AND s.expires_at > NOW()
		ORDER BY s.last_activity_at DESC
		LIMIT $2 OFFSET $3`

	rows, dbErr := r.db.Query(ctx, query, userID, limit, offset)
	if dbErr != nil {
		return nil, 0, pkgErrors.FromError(dbErr, pkgErrors.CodeDatabaseError, "failed to query sessions").
			WithDetail("user_id", userID)
	}
	defer rows.Close()

	var results []*models.AuthSession
	var totalCount int
	for rows.Next() {
		var r models.AuthSession
		err := rows.ScanModelAndAppend(&r, &totalCount) // Scan session fields into r and total count into r.Total
		if err != nil {
			return nil, 0, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to scan session row").
				WithDetail("user_id", userID)
		}
		results = append(results, &r)
	}

	if rows.Err() != nil {
		err = pkgErrors.FromError(rows.Err(), pkgErrors.CodeDatabaseError, "error iterating session rows").
			WithDetail("user_id", userID)
		return nil, 0, err
	}

	if len(results) == 0 {
		return nil, 0, nil
	}

	r.log.Debug("Sessions fetched successfully",
		logger.String("user_id", userID),
		logger.Int("session_count", len(results)),
	)

	var result []*domain.Session
	for _, res := range results {
		metaData := make(map[string]interface{})
		utils.UnmarshalRawMessageSafe(*res.Metadata, &metaData)
		s := domain.FromSessionModel(res)
		s.Metadata = metaData
		result = append(result, &s)
	}
	return result, totalCount, nil
}

func (r *sessionRepository) GetSessionByRefreshToken(ctx context.Context, refreshToken string) (*domain.Session, pkgErrors.AppError) {
	r.log.Debug("Fetching session by refresh token")
	var session models.AuthSession
	// Hits auth.sessions using idx_auth_sessions_refresh_token (unique B-tree on refresh_token). Condition: refresh_token = $1. Returns full session row including revoked_at and expires_at.
	query := `SELECT * FROM auth.sessions WHERE refresh_token = $1 LIMIT 1`
	err := r.db.QueryRow(ctx, query, refreshToken).ScanModel(&session)
	if err != nil {
		if postgres.IsNotFoundError(err) {
			r.log.Debug("No session found for refresh token")
			return nil, nil
		}
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get session by refresh token")
	}

	metaData := make(map[string]interface{})
	json.Unmarshal(*session.Metadata, &metaData)
	s := domain.FromSessionModel(&session)
	s.Metadata = metaData

	r.log.Debug("Session fetched successfully by refresh token",
		logger.String("session_id", session.ID),
	)
	return &s, nil
}

func (r *sessionRepository) UpdateSession(ctx context.Context, session *domain.UpdateAuthSession) (*domain.Session, pkgErrors.AppError) {
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

	r.log.Info("Session updated successfully",
		logger.String("session_id", session.ID),
	)
	s := domain.FromSessionModel(&updatedSession)
	s.Metadata = metaData
	return &s, nil
}

func (r *sessionRepository) GetSessionByUserId(ctx context.Context, userID string, deviceID *string) (*domain.Session, pkgErrors.AppError) {
	r.log.Debug("Fetching session by user ID",
		logger.String("user_id", userID),
	)
	var session models.AuthSession
	query := `SELECT * FROM auth.sessions WHERE user_id = $1 AND coalesce(device_id, '') = coalesce($2, '') AND revoked_at IS NULL AND expires_at > NOW() LIMIT 1`
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
	metaData := make(map[string]interface{})
	if session.Metadata != nil {
		json.Unmarshal(*session.Metadata, &metaData)
	}

	r.log.Debug("Session fetched successfully by user ID",
		logger.String("session_id", session.ID),
	)
	s := domain.FromSessionModel(&session)
	s.Metadata = metaData
	return &s, nil
}

func (r *sessionRepository) CountActiveSessions(ctx context.Context, userID string) (int64, pkgErrors.AppError) {
	var count int64
	query := `SELECT COUNT(*) FROM auth.sessions WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()`
	if err := r.db.QueryRow(ctx, query, userID).Scan(&count); err != nil {
		return 0, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to count active sessions").
			WithDetail("user_id", userID)
	}
	return count, nil
}

// EvictOldestSession revokes the oldest active session for the user and returns
func (r *sessionRepository) EvictOldestSession(ctx context.Context, userID string) (*domain.Session, pkgErrors.AppError) {
	r.log.Info("Evicting oldest session for user",
		logger.String("user_id", userID),
	)

	query := `
		WITH oldest AS (
			SELECT id, session_token, user_id
			FROM auth.sessions
			WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
			ORDER BY last_activity_at ASC
			LIMIT 1
		)
		UPDATE auth.sessions
		SET revoked_at = NOW(),
		    revoked_reason = 'session_limit_exceeded'
		WHERE id = (SELECT id FROM oldest)
		RETURNING id, session_token, user_id`

	var sessionID, sessionToken, uid string
	err := r.db.QueryRow(ctx, query, userID).Scan(&sessionID, &sessionToken, &uid)
	if err != nil {
		if postgres.IsNotFoundError(err) {
			r.log.Info("No active session to evict",
				logger.String("user_id", userID),
			)
			return nil, nil
		}
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to evict oldest session").
			WithDetail("user_id", userID)
	}

	r.log.Info("Oldest session evicted",
		logger.String("user_id", userID),
		logger.String("session_id", sessionID),
	)
	return &domain.Session{ID: sessionID, SessionToken: sessionToken, UserID: uid}, nil
}

func (r *sessionRepository) InsertOutboxEvent(ctx context.Context, userID string, eventType string, payload []byte) pkgErrors.AppError {
	query := `INSERT INTO auth.outbox (user_id, event_type, payload, created_at) VALUES ($1, $2, $3, NOW())`
	_, err := r.db.Exec(ctx, query, userID, eventType, payload)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to insert outbox event").
			WithDetail("user_id", userID).
			WithDetail("event_type", eventType)
	}
	return nil
}

func (r *sessionRepository) DeleteSessionByID(ctx context.Context, sessionID string) pkgErrors.AppError {
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

func (r *sessionRepository) GetSessionByID(ctx context.Context, sessionID string) (*domain.Session, pkgErrors.AppError) {
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

	s := domain.FromSessionModel(&session)
	s.Metadata = metaData

	r.log.Debug("Session fetched successfully by ID",
		logger.String("session_id", session.ID),
	)
	return &s, nil
}

func (r *sessionRepository) RevokeSession(ctx context.Context, sessionID string, reason string) pkgErrors.AppError {
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

func (r *sessionRepository) ClearNotificationsForSession(ctx context.Context, sessionID string) pkgErrors.AppError {
	r.log.Info("Clearing notifications for session",
		logger.String("session_id", sessionID),
	)
	// If device_type = android and fcm_token is non-null: UPDATE auth.sessions SET fcm_token = NULL WHERE id = $1.
	// UPDATE users.devices SET fcm_token = NULL, push_enabled = FALSE WHERE user_id = $1 AND device_id = $2
	// If device_type = ios and apns_token is non-null: UPDATE auth.sessions SET apns_token = NULL WHERE id = $1.

	query := `UPDATE auth.sessions
		SET fcm_token = NULL,
		    apns_token = NULL,
		    push_enabled = FALSE
		WHERE id = $1`
	_, err := r.db.Exec(ctx, query, sessionID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to clear notifications for session").
			WithDetail("session_id", sessionID)
	}
	r.log.Info("Notifications cleared for session",
		logger.String("session_id", sessionID),
	)
	return nil
}

func (r *sessionRepository) ClearNotificationsForAllSessions(ctx context.Context, userID string) pkgErrors.AppError {
	r.log.Info("Clearing notifications for all sessions of user",
		logger.String("user_id", userID),
	)
	query := `UPDATE auth.sessions
		SET fcm_token = NULL,
		    apns_token = NULL,
		    push_enabled = FALSE
		WHERE user_id = $1`
	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to clear notifications for all sessions").
			WithDetail("user_id", userID)
	}
	r.log.Info("Notifications cleared for all sessions of user",
		logger.String("user_id", userID),
	)
	return nil
}

func (r *sessionRepository) RevokeSessionByRefreshToken(ctx context.Context, refreshToken string, reason string) pkgErrors.AppError {
	r.log.Info("Revoking session by refresh token",
		logger.String("reason", reason),
	)
	query := `UPDATE auth.sessions
		SET revoked_at = NOW(),
		    revoked_reason = $1
		WHERE refresh_token = $2 AND revoked_at IS NULL`
	_, err := r.db.Exec(ctx, query, reason, refreshToken)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to revoke session by refresh token").
			WithDetail("reason", reason)
	}
	r.log.Info("Session revoked by refresh token",
		logger.String("reason", reason),
	)
	return nil
}

func (r *sessionRepository) RevokeAllUserSessions(ctx context.Context, userID string, reason string) (*[]string, pkgErrors.AppError) {
	r.log.Info("Revoking all sessions for user",
		logger.String("user_id", userID),
		logger.String("reason", reason),
	)
	var expiredTokens []string
	query := `UPDATE auth.sessions
		SET revoked_at = NOW(),
			revoked_reason = $1
		WHERE user_id = $2 AND revoked_at IS NULL
		RETURNING refresh_token`
	rows, err := r.db.Query(ctx, query, reason, userID)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to revoke all user sessions").
			WithDetail("user_id", userID).
			WithDetail("reason", reason)
	}
	defer rows.Close()

	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err != nil {
			return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to scan revoked session token").
				WithDetail("user_id", userID)
		}
		expiredTokens = append(expiredTokens, token)
	}

	if err = rows.Err(); err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "error iterating revoked sessions").
			WithDetail("user_id", userID)
	}
	r.log.Info("All sessions revoked for user",
		logger.String("user_id", userID),
	)
	return &expiredTokens, nil
}

func (r *sessionRepository) RevokeActiveDeviceSession(ctx context.Context, userID, deviceID, reason string) (string, pkgErrors.AppError) {
	r.log.Info("Revoking active session for same device",
		logger.String("user_id", userID),
		logger.String("device_id", deviceID),
	)
	// Returns the revoked session ID so the caller can purge it from Redis.
	query := `
		UPDATE auth.sessions
		SET revoked_at = NOW(),
		    revoked_reason = $3
		WHERE user_id = $1
		  AND device_id = $2
		  AND revoked_at IS NULL
		  AND expires_at > NOW()
		RETURNING id`

	var sessionID string
	err := r.db.QueryRow(ctx, query, userID, deviceID, reason).Scan(&sessionID)
	if err != nil {
		if postgres.IsNotFoundError(err) {
			return "", nil
		}
		return "", pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to revoke active device session").
			WithDetail("user_id", userID).
			WithDetail("device_id", deviceID)
	}

	r.log.Info("Active device session revoked",
		logger.String("user_id", userID),
		logger.String("device_id", deviceID),
		logger.String("session_id", sessionID),
	)
	return sessionID, nil
}
