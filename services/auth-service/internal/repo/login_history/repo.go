package login_history

import (
	"shared/pkg/database"
	"shared/pkg/logger"
)

// ============================================================================
// Repository Definition
// ============================================================================

type loginHistoryRepository struct {
	db  database.Database
	log logger.Logger
}

func NewLoginHistoryRepo(db database.Database, log logger.Logger) LoginHistoryRespository {
	return &loginHistoryRepository{
		db:  db,
		log: log,
	}
}

var _ LoginHistoryRespository = (*loginHistoryRepository)(nil)
