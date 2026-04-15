package outbox

import (
	"context"
	"shared/pkg/logger"
)

type OutboxRepository interface {
	// auth.session.created
	AddAuthSessionCreatedEvent(ctx context.Context, userID, sessionID string) error
	// auth.suspicious.activity
	AddAuthSuspiciousActivityEvent(ctx context.Context, userID, details string) error
	// auth.session.revoked
	AddAuthSessionRevokedEvent(ctx context.Context, userID, sessionID string) error
	// auth.password.reset_requested
	AddAuthPasswordResetRequestedEvent(ctx context.Context, userID, token string) error
}

func (r *outboxRepository) AddAuthSessionCreatedEvent(ctx context.Context, userID, sessionID string) error {
	r.log.Info("Adding auth session created event to outbox",
		logger.String("user_id", userID),
		logger.String("session_id", sessionID),
	)

	query := `INSERT INTO outbox (event_type, user_id, payload) VALUES ($1, $2, $3)`
	payload := map[string]string{
		"user_id":    userID,
		"session_id": sessionID,
	}

	result, err := r.db.Exec(ctx, query, "auth.session.created", userID, payload)
	if err != nil {
		r.log.Error("Error adding auth session created event to outbox", logger.Error(err))
		return err
	}

	r.log.Info("Auth session created event added to outbox", logger.Any("result", result))
	return nil
}

func (r *outboxRepository) AddAuthSuspiciousActivityEvent(ctx context.Context, userID, details string) error {
	r.log.Info("Adding auth suspicious activity event to outbox",
		logger.String("user_id", userID),
		logger.String("details", details),
	)

	query := `INSERT INTO outbox (event_type, user_id, payload) VALUES ($1, $2, $3)`
	payload := map[string]string{
		"user_id": userID,
		"details": details,
	}

	result, err := r.db.Exec(ctx, query, "auth.suspicious.activity", userID, payload)
	if err != nil {
		r.log.Error("Error adding auth suspicious activity event to outbox", logger.Error(err))
		return err
	}

	r.log.Info("Auth suspicious activity event added to outbox", logger.Any("result", result))
	return nil
}

func (r *outboxRepository) AddAuthSessionRevokedEvent(ctx context.Context, userID, sessionID string) error {
	r.log.Info("Adding auth session revoked event to outbox",
		logger.String("user_id", userID),
		logger.String("session_id", sessionID),
	)

	query := `INSERT INTO outbox (event_type, user_id, payload) VALUES ($1, $2, $3)`
	payload := map[string]string{
		"user_id":    userID,
		"session_id": sessionID,
	}

	result, err := r.db.Exec(ctx, query, "auth.session.revoked", userID, payload)
	if err != nil {
		r.log.Error("Error adding auth session revoked event to outbox", logger.Error(err))
		return err
	}

	r.log.Info("Auth session revoked event added to outbox", logger.Any("result", result))
	return nil
}

func (r *outboxRepository) AddAuthPasswordResetRequestedEvent(ctx context.Context, userID, token string) error {
	r.log.Info("Adding auth password reset requested event to outbox",
		logger.String("user_id", userID),
		logger.String("token", token),
	)

	query := `INSERT INTO outbox (event_type, user_id, payload) VALUES ($1, $2, $3)`
	payload := map[string]string{
		"user_id": userID,
		"token":   token,
	}

	result, err := r.db.Exec(ctx, query, "auth.password.reset_requested", userID, payload)
	if err != nil {
		r.log.Error("Error adding auth password reset requested event to outbox", logger.Error(err))
		return err
	}

	r.log.Info("Auth password reset requested event added to outbox", logger.Any("result", result))
	return nil
}
