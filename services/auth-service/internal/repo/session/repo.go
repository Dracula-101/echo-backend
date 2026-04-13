package session

import (
	"shared/pkg/database"
	"shared/pkg/logger"
)

// ============================================================================
// Repository Definition
// ============================================================================

type sessionRepository struct {
	db  database.Database
	log logger.Logger
}

func NewSessionRepo(db database.Database, log logger.Logger) SessionRepository {
	return &sessionRepository{
		db:  db,
		log: log,
	}
}

var _ SessionRepository = (*sessionRepository)(nil)
