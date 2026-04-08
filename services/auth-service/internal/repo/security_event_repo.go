package repository

import (
	"context"
	"encoding/json"

	"shared/pkg/database"
	"shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
)

type SecurityEventRepositoryInterface interface {
	LogEvent(ctx context.Context, event SecurityEventInput) pkgErrors.AppError
}

type SecurityEventInput struct {
	UserID        string
	SessionID     *string
	EventType     models.SecurityEventType
	EventCategory string
	Severity      models.SecuritySeverity
	Status        string
	Description   string
	IPAddress     string
	UserAgent     string
}

type SecurityEventRepo struct {
	db  database.Database
	log logger.Logger
}

func NewSecurityEventRepo(db database.Database, log logger.Logger) *SecurityEventRepo {
	if db == nil {
		panic("Database is required for SecurityEventRepo")
	}
	if log == nil {
		panic("Logger is required for SecurityEventRepo")
	}
	return &SecurityEventRepo{db: db, log: log}
}

func (r *SecurityEventRepo) LogEvent(ctx context.Context, event SecurityEventInput) pkgErrors.AppError {
	r.log.Info("Logging security event",
		logger.String("user_id", event.UserID),
		logger.String("event_type", string(event.EventType)),
		logger.String("status", event.Status),
	)

	_, err := r.db.Insert(ctx, &models.SecurityEvent{
		UserID:        &event.UserID,
		SessionID:     event.SessionID,
		EventType:     event.EventType,
		EventCategory: &event.EventCategory,
		Severity:      event.Severity,
		Status:        &event.Status,
		Description:   &event.Description,
		IPAddress:     &event.IPAddress,
		UserAgent:     &event.UserAgent,
		IsSuspicious:  false,
		Metadata:      func() *json.RawMessage { r := json.RawMessage(`{}`); return &r }(),
	})
	if err != nil {
		r.log.Warn("Failed to log security event (non-critical)",
			logger.String("user_id", event.UserID),
			logger.String("event_type", string(event.EventType)),
			logger.Error(err),
		)
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to log security event")
	}
	return nil
}
