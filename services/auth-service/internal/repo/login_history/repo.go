package login_history

import (
	"shared/pkg/database"
	"shared/pkg/logger"
)

// ============================================================================
// Repository Definition
// ============================================================================

type LoginHistoryRepository struct {
	db  database.Database
	log logger.Logger
}

func NewLoginHistoryRepo(db database.Database, log logger.Logger) *LoginHistoryRepository {
	return &LoginHistoryRepository{
		db:  db,
		log: log,
	}
}

var _ LoginHistoryRepositoryInterface = (*LoginHistoryRepository)(nil)
