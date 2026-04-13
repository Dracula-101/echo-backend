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
	GetAllSessionsByUserId(ctx context.Context, userID string) ([]*domain.Session, pkgErrors.AppError)
	GetSessionByID(ctx context.Context, sessionID string) (*domain.Session, pkgErrors.AppError)
	UpdateSession(ctx context.Context, session *domain.UpdateAuthSession) (*domain.Session, pkgErrors.AppError)
	DeleteSessionByID(ctx context.Context, sessionID string) pkgErrors.AppError
	EvictOldestSession(ctx context.Context, userID string) pkgErrors.AppError
	RevokeSession(ctx context.Context, sessionID string, reason string) pkgErrors.AppError
	RevokeAllUserSessions(ctx context.Context, userID string, reason string) pkgErrors.AppError
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

func (r *sessionRepository) GetAllSessionsByUserId(ctx context.Context, userID string) ([]*domain.Session, pkgErrors.AppError) {
	r.log.Debug("Fetching all sessions by user ID",
		logger.String("user_id", userID),
	)
	var sessions []*models.AuthSession
	query := "SELECT * FROM auth.sessions WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW() ORDER BY created_at DESC"
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
		utils.UnmarshalRawMessageSafe(*session.Metadata, &metaData)
		s := domain.FromSessionModel(session)
		result = append(result, &s)
	}
	return result, nil
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

func (r *sessionRepository) EvictOldestSession(ctx context.Context, userID string) pkgErrors.AppError {
	r.log.Info("Evicting oldest session for user",
		logger.String("user_id", userID),
	)
	query := `WITH oldest AS (
		SELECT id FROM auth.sessions
			WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
			ORDER BY last_activity_at ASC
			LIMIT 1
		)
		UPDATE auth.sessions
		SET revoked_at = NOW(),
			revoked_reason = 'evicted_due_to_session_limit'
		WHERE id IN (SELECT id FROM oldest)`
	result, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to evict oldest session").
			WithDetail("user_id", userID)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		r.log.Info("Oldest session evicted",
			logger.String("user_id", userID),
			logger.Int64("sessions_evicted", rowsAffected),
		)
	} else {
		r.log.Info("No session evicted (no active sessions found)",
			logger.String("user_id", userID),
		)
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

func (r *sessionRepository) RevokeAllUserSessions(ctx context.Context, userID string, reason string) pkgErrors.AppError {
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
