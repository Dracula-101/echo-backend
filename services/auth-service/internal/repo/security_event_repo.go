package repository

import (
	"context"
	"shared/pkg/database"
	"shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
)

// ============================================================================
// Repository Definition
// ============================================================================

type SecurityEventRepo struct {
	db  database.Database
	log logger.Logger
}

func NewSecurityEventRepo(db database.Database, log logger.Logger) *SecurityEventRepo {
	return &SecurityEventRepo{
		db:  db,
		log: log,
	}
}

// ============================================================================
// Security Event Operations
// ============================================================================

func (r *SecurityEventRepo) LogSecurityEvent(ctx context.Context, event *models.SecurityEvent) pkgErrors.AppError {
	r.log.Debug("Logging security event",
		logger.Any("event_type", event.EventType),
		logger.Any("severity", event.Severity),
		logger.String("user_id", safeDerefString(event.UserID)),
		logger.String("session_id", safeDerefString(event.SessionID)),
		logger.Bool("is_suspicious", event.IsSuspicious),
	)
	_, err := r.db.Create(ctx, event)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to log security event").
			WithDetail("event_type", string(event.EventType)).
			WithDetail("severity", string(event.Severity))
	}
	r.log.Debug("Security event logged successfully",
		logger.String("event_id", event.ID),
	)
	return nil
}

func (r *SecurityEventRepo) GetSecurityEventsByUserID(ctx context.Context, userID string, limit int) ([]*models.SecurityEvent, error) {
	r.log.Debug("Fetching security events by user ID",
		logger.String("user_id", userID),
		logger.Int("limit", limit),
	)
	var events []*models.SecurityEvent
	query := `SELECT id, user_id, session_id, event_type, event_category, severity, status, 
		description, ip_address, user_agent, device_id, location_country, location_city, 
		risk_score, is_suspicious, blocked_reason, created_at, metadata 
		FROM auth.security_events 
		WHERE user_id = $1 
		ORDER BY created_at DESC 
		LIMIT $2`
	err := r.db.FindMany(ctx, &events, query, userID, limit)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get security events by user ID").
			WithDetail("user_id", userID).
			WithDetail("limit", limit)
	}
	r.log.Debug("Security events fetched successfully",
		logger.String("user_id", userID),
		logger.Int("count", len(events)),
	)
	return events, nil
}

func (r *SecurityEventRepo) GetSecurityEventByID(ctx context.Context, id string) (*models.SecurityEvent, error) {
	r.log.Debug("Fetching security event by ID",
		logger.String("event_id", id),
	)
	var event models.SecurityEvent
	query := `SELECT id, user_id, session_id, event_type, event_category, severity, status, 
		description, ip_address, user_agent, device_id, location_country, location_city, 
		risk_score, is_suspicious, blocked_reason, created_at, metadata 
		FROM auth.security_events 
		WHERE id = $1 
		LIMIT 1`
	err := r.db.FindOne(ctx, &event, query, id)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get security event by ID").
			WithDetail("event_id", id)
	}
	r.log.Debug("Security event fetched successfully",
		logger.String("event_id", event.ID),
	)
	return &event, nil
}

func (r *SecurityEventRepo) GetSuspiciousEvents(ctx context.Context, userID string, limit int) ([]*models.SecurityEvent, error) {
	r.log.Debug("Fetching suspicious security events",
		logger.String("user_id", userID),
		logger.Int("limit", limit),
	)
	var events []*models.SecurityEvent
	query := `SELECT id, user_id, session_id, event_type, event_category, severity, status, 
		description, ip_address, user_agent, device_id, location_country, location_city, 
		risk_score, is_suspicious, blocked_reason, created_at, metadata 
		FROM auth.security_events 
		WHERE user_id = $1 AND is_suspicious = true 
		ORDER BY created_at DESC 
		LIMIT $2`
	err := r.db.FindMany(ctx, &events, query, userID, limit)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get suspicious security events").
			WithDetail("user_id", userID).
			WithDetail("limit", limit)
	}
	r.log.Debug("Suspicious security events fetched successfully",
		logger.String("user_id", userID),
		logger.Int("count", len(events)),
	)
	return events, nil
}

func (r *SecurityEventRepo) GetEventsByType(ctx context.Context, userID string, eventType string, limit int) ([]*models.SecurityEvent, error) {
	r.log.Debug("Fetching security events by type",
		logger.String("user_id", userID),
		logger.String("event_type", eventType),
		logger.Int("limit", limit),
	)
	var events []*models.SecurityEvent
	query := `SELECT id, user_id, session_id, event_type, event_category, severity, status, 
		description, ip_address, user_agent, device_id, location_country, location_city, 
		risk_score, is_suspicious, blocked_reason, created_at, metadata 
		FROM auth.security_events 
		WHERE user_id = $1 AND event_type = $2 
		ORDER BY created_at DESC 
		LIMIT $3`
	err := r.db.FindMany(ctx, &events, query, userID, eventType, limit)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get security events by type").
			WithDetail("user_id", userID).
			WithDetail("event_type", eventType).
			WithDetail("limit", limit)
	}
	r.log.Debug("Security events by type fetched successfully",
		logger.String("user_id", userID),
		logger.String("event_type", eventType),
		logger.Int("count", len(events)),
	)
	return events, nil
}

func (r *SecurityEventRepo) CountEventsBySeverity(ctx context.Context, userID string, severity string, duration string) (int, error) {
	r.log.Debug("Counting security events by severity",
		logger.String("user_id", userID),
		logger.String("severity", severity),
		logger.String("duration", duration),
	)
	var count int
	query := `SELECT COUNT(*) 
		FROM auth.security_events 
		WHERE user_id = $1 
		AND severity = $2 
		AND created_at > NOW() - INTERVAL '1 ' || $3`
	err := r.db.QueryRow(ctx, query, userID, severity, duration).Scan(&count)
	if err != nil {
		return 0, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to count security events by severity").
			WithDetail("user_id", userID).
			WithDetail("severity", severity).
			WithDetail("duration", duration)
	}
	r.log.Debug("Security events counted by severity",
		logger.String("user_id", userID),
		logger.String("severity", severity),
		logger.Int("count", count),
	)
	return count, nil
}

func (r *SecurityEventRepo) DeleteSecurityEventsByUserID(ctx context.Context, userID string) pkgErrors.AppError {
	r.log.Debug("Deleting security events by user ID",
		logger.String("user_id", userID),
	)
	query := `DELETE FROM auth.security_events WHERE user_id = $1`
	result, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to delete security events").
			WithDetail("user_id", userID)
	}
	rowsAffected, _ := result.RowsAffected()
	r.log.Debug("Security events deleted successfully",
		logger.String("user_id", userID),
		logger.Int64("rows_affected", rowsAffected),
	)
	return nil
}

func (r *SecurityEventRepo) DeleteSecurityEventByID(ctx context.Context, id string) pkgErrors.AppError {
	r.log.Debug("Deleting security event by ID",
		logger.String("event_id", id),
	)
	query := `DELETE FROM auth.security_events WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to delete security event").
			WithDetail("event_id", id)
	}
	r.log.Debug("Security event deleted successfully",
		logger.String("event_id", id),
	)
	return nil
}
