package email_verification

import (
	"shared/pkg/database"
	"shared/pkg/logger"
)

type EmailVerificationTokenRepository struct {
	db  database.Database
	log logger.Logger
}

func NewEmailVerificationTokenRepo(db database.Database, log logger.Logger) *EmailVerificationTokenRepository {
	if db == nil {
		panic("Database is required for EmailVerificationTokenRepo")
	}
	if log == nil {
		panic("Logger is required for EmailVerificationTokenRepo")
	}
	return &EmailVerificationTokenRepository{db: db, log: log}
}

var _ EmailVerificationTokenRepositoryInterface = (*EmailVerificationTokenRepository)(nil)
