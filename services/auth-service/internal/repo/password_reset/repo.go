package password_reset

import (
	"shared/pkg/database"
	"shared/pkg/logger"
)

type PasswordResetTokenRepository struct {
	db  database.Database
	log logger.Logger
}

func NewPasswordResetTokenRepo(db database.Database, log logger.Logger) *PasswordResetTokenRepository {
	if db == nil {
		panic("Database is required for PasswordResetTokenRepo")
	}
	if log == nil {
		panic("Logger is required for PasswordResetTokenRepo")
	}
	return &PasswordResetTokenRepository{db: db, log: log}
}

var _ PasswordResetTokenRepositoryInterface = (*PasswordResetTokenRepository)(nil)
