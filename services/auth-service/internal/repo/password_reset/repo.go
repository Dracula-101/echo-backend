package password_reset

import (
	"shared/pkg/database"
	"shared/pkg/logger"
)

type passwordResetTokenRepository struct {
	db  database.Database
	log logger.Logger
}

func NewPasswordResetTokenRepo(db database.Database, log logger.Logger) PasswordResetTokenRepository {
	if db == nil {
		panic("Database is required for PasswordResetTokenRepo")
	}
	if log == nil {
		panic("Logger is required for PasswordResetTokenRepo")
	}
	return &passwordResetTokenRepository{
		db:  db,
		log: log,
	}
}

var _ PasswordResetTokenRepository = (*passwordResetTokenRepository)(nil)
