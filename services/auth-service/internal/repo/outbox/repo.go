package outbox

import (
	"shared/pkg/database"
	"shared/pkg/logger"
)

type outboxRepository struct {
	db  database.Database
	log logger.Logger
}

func NewOutboxRepository(db database.Database, log logger.Logger) OutboxRepository {
	return &outboxRepository{
		db:  db,
		log: log,
	}
}

var _ OutboxRepository = (*outboxRepository)(nil)
