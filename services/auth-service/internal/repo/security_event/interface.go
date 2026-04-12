package security_event

import (
	"shared/pkg/database"
	"shared/pkg/logger"
)

type SecurityEventRepository struct {
	db  database.Database
	log logger.Logger
}

func NewSecurityEventRepo(db database.Database, log logger.Logger) *SecurityEventRepository {
	if db == nil {
		panic("database is required for SecurityEventRepo")
	}
	if log == nil {
		panic("logger is required for SecurityEventRepo")
	}
	return &SecurityEventRepository{db: db, log: log}
}

var _ SecurityEventInterface = (*SecurityEventRepository)(nil)
