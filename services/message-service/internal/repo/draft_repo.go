package repo

import (
	"context"
	"database/sql"
	"time"

	"shared/pkg/database"
	"shared/pkg/logger"

	"github.com/google/uuid"
)

// Draft represents a message draft stored in the messages.drafts table.
type Draft struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	ConversationID   uuid.UUID
	Content          string
	ReplyToMessageID *uuid.UUID
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// DraftRepository defines the interface for draft persistence operations.
type DraftRepository interface {
	UpsertDraft(ctx context.Context, userID, conversationID uuid.UUID, content string, replyToMessageID *uuid.UUID) error
	GetDraft(ctx context.Context, userID, conversationID uuid.UUID) (*Draft, error)
	DeleteDraft(ctx context.Context, userID, conversationID uuid.UUID) error
}

type draftRepository struct {
	db     database.Database
	logger logger.Logger
}

// NewDraftRepository creates a new instance of DraftRepository.
func NewDraftRepository(db database.Database, log logger.Logger) DraftRepository {
	return &draftRepository{
		db:     db,
		logger: log,
	}
}

// UpsertDraft inserts a new draft or updates an existing one for the given user and conversation.
func (r *draftRepository) UpsertDraft(ctx context.Context, userID, conversationID uuid.UUID, content string, replyToMessageID *uuid.UUID) error {
	query := `
		INSERT INTO messages.drafts (
			user_id, conversation_id, content, reply_to_message_id, updated_at
		) VALUES (
			$1, $2, $3, $4, NOW()
		)
		ON CONFLICT (user_id, conversation_id)
		DO UPDATE SET
			content = EXCLUDED.content,
			reply_to_message_id = EXCLUDED.reply_to_message_id,
			updated_at = NOW()
	`

	_, err := r.db.Exec(ctx, query, userID, conversationID, content, replyToMessageID)
	if err != nil {
		r.logger.Error("Failed to upsert draft",
			logger.String("user_id", userID.String()),
			logger.String("conversation_id", conversationID.String()),
			logger.Error(err),
		)
		return err
	}

	return nil
}

// GetDraft retrieves a draft for the given user and conversation. Returns nil if not found.
func (r *draftRepository) GetDraft(ctx context.Context, userID, conversationID uuid.UUID) (*Draft, error) {
	query := `
		SELECT id, user_id, conversation_id, content, reply_to_message_id, created_at, updated_at
		FROM messages.drafts
		WHERE user_id = $1 AND conversation_id = $2
	`

	var draft Draft
	var replyToMessageID sql.NullString

	err := r.db.QueryRow(ctx, query, userID, conversationID).Scan(
		&draft.ID,
		&draft.UserID,
		&draft.ConversationID,
		&draft.Content,
		&replyToMessageID,
		&draft.CreatedAt,
		&draft.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Error("Failed to get draft",
			logger.String("user_id", userID.String()),
			logger.String("conversation_id", conversationID.String()),
			logger.Error(err),
		)
		return nil, err
	}

	if replyToMessageID.Valid {
		parsed := uuid.MustParse(replyToMessageID.String)
		draft.ReplyToMessageID = &parsed
	}

	return &draft, nil
}

// DeleteDraft removes a draft for the given user and conversation.
func (r *draftRepository) DeleteDraft(ctx context.Context, userID, conversationID uuid.UUID) error {
	query := `
		DELETE FROM messages.drafts
		WHERE user_id = $1 AND conversation_id = $2
	`

	_, err := r.db.Exec(ctx, query, userID, conversationID)
	if err != nil {
		r.logger.Error("Failed to delete draft",
			logger.String("user_id", userID.String()),
			logger.String("conversation_id", conversationID.String()),
			logger.Error(err),
		)
		return err
	}

	return nil
}
