package repo

import (
	"context"
	"time"

	"shared/pkg/database"
	"shared/pkg/logger"

	"github.com/google/uuid"
)

// TypingIndicator represents an active typing indicator
type TypingIndicator struct {
	ConversationID uuid.UUID
	UserID         uuid.UUID
	StartedAt      time.Time
	ExpiresAt      time.Time
}

// TypingRepository defines the typing indicator data access methods
type TypingRepository interface {
	UpsertTypingIndicator(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID, expiresAt time.Time) error
	RemoveTypingIndicator(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) error
	GetActiveTypingUsers(ctx context.Context, conversationID uuid.UUID, excludeUserID uuid.UUID) ([]TypingIndicator, error)
}

type typingRepository struct {
	db     database.Database
	logger logger.Logger
}

// NewTypingRepository creates a new typing repository
func NewTypingRepository(db database.Database, log logger.Logger) TypingRepository {
	return &typingRepository{
		db:     db,
		logger: log,
	}
}

func (r *typingRepository) UpsertTypingIndicator(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID, expiresAt time.Time) error {
	query := `
		INSERT INTO messages.typing_indicators (conversation_id, user_id, started_at, expires_at)
		VALUES ($1, $2, NOW(), $3)
		ON CONFLICT (conversation_id, user_id)
		DO UPDATE SET started_at = NOW(), expires_at = $3
	`
	_, err := r.db.Exec(ctx, query, conversationID, userID, expiresAt)
	return err
}

func (r *typingRepository) RemoveTypingIndicator(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) error {
	query := `DELETE FROM messages.typing_indicators WHERE conversation_id = $1 AND user_id = $2`
	_, err := r.db.Exec(ctx, query, conversationID, userID)
	return err
}

func (r *typingRepository) GetActiveTypingUsers(ctx context.Context, conversationID uuid.UUID, excludeUserID uuid.UUID) ([]TypingIndicator, error) {
	query := `
		SELECT conversation_id, user_id, started_at, expires_at
		FROM messages.typing_indicators
		WHERE conversation_id = $1 AND user_id != $2 AND expires_at > NOW()
		ORDER BY started_at ASC
	`
	rows, err := r.db.Query(ctx, query, conversationID, excludeUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indicators []TypingIndicator
	for rows.Next() {
		var ti TypingIndicator
		if err := rows.Scan(&ti.ConversationID, &ti.UserID, &ti.StartedAt, &ti.ExpiresAt); err != nil {
			return nil, err
		}
		indicators = append(indicators, ti)
	}
	return indicators, rows.Err()
}
