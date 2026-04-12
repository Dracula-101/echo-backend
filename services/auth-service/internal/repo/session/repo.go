package session

import (
	"shared/pkg/database"
	"shared/pkg/logger"
)

// ============================================================================
// Repository Definition
// ============================================================================

type SessionRepository struct {
	db  database.Database
	log logger.Logger
}

func NewSessionRepo(db database.Database, log logger.Logger) *SessionRepository {
	return &SessionRepository{
		db:  db,
		log: log,
	}
}

var _ SessionRepositoryInterface = (*SessionRepository)(nil)
