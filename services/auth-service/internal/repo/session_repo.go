package repository

import (
	"context"
	"shared/pkg/database"
	"shared/pkg/database/postgres/models"
	"shared/pkg/logger"
)

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

func (r *SessionRepo) CreateSession(ctx context.Context, session *models.AuthSession) error {
	r.log.Debug("Creating new session",
		logger.String("user_id", session.UserID),
		logger.String("device_id", *session.DeviceID),
		logger.String("ip_address", session.IPAddress),
	)
	_, err := r.db.Create(ctx, session)
	if err != nil {
		r.log.Error("Failed to create session", logger.Error(err))
		return err
	}
	r.log.Debug("Session created successfully",
		logger.String("session_id", session.ID),
	)
	return nil
}

func (r *SessionRepo) GetSessionByUserId(ctx context.Context, userID string) (*models.AuthSession, error) {
	r.log.Debug("Fetching session by user ID",
		logger.String("user_id", userID),
	)
	var session models.AuthSession
	query := `SELECT * FROM auth.sessions WHERE user_id = $1 AND revoked_at IS NULL ORDER BY created_at DESC LIMIT 1`
	err := r.db.QueryRow(ctx, query, userID).ScanOne(&session)
	if err != nil {
		r.log.Error("Failed to get session by user ID", logger.Error(err))
		return nil, err
	}
	r.log.Debug("Session fetched successfully",
		logger.String("session_id", session.ID),
	)
	return &session, nil
}

func (r *SessionRepo) DeleteSessionByID(ctx context.Context, sessionID string) error {
	r.log.Debug("Deleting session",
		logger.String("session_id", sessionID),
	)
	query := `DELETE FROM auth.sessions WHERE id = $1`
	_, err := r.db.Exec(ctx, query, sessionID)
	if err != nil {
		r.log.Error("Failed to delete session", logger.Error(err))
		return err
	}
	r.log.Debug("Session deleted successfully",
		logger.String("session_id", sessionID),
	)
	return nil
}
