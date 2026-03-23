package repo

import (
	"context"
	"time"

	"shared/pkg/database"
	"shared/pkg/logger"

	"github.com/google/uuid"
)

// ConversationSyncState holds sync metadata for a conversation
type ConversationSyncState struct {
	ConversationID uuid.UUID
	UnreadCount    int
	LastMessageID  *uuid.UUID
	LastMessageAt  *time.Time
}

// ConversationRepository handles conversation-related database operations
type ConversationRepository interface {
	IsUserParticipant(ctx context.Context, userID, conversationID uuid.UUID) (bool, error)
	IsUserBlocked(ctx context.Context, userID, conversationID uuid.UUID) (bool, error)
	GetUserConversations(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	GetUserConversationSyncState(ctx context.Context, userID uuid.UUID) ([]ConversationSyncState, error)
	ValidateConversationAccess(ctx context.Context, userID, conversationID uuid.UUID) (bool, error)
	ConversationExists(ctx context.Context, conversationID uuid.UUID) (bool, error)
}

type conversationRepository struct {
	db  database.Database
	log logger.Logger
}

// NewConversationRepository creates a new conversation repository
func NewConversationRepository(db database.Database, log logger.Logger) ConversationRepository {
	return &conversationRepository{
		db:  db,
		log: log,
	}
}

// IsUserParticipant checks if a user is an active participant in a conversation
func (r *conversationRepository) IsUserParticipant(ctx context.Context, userID, conversationID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM messages.conversation_participants
			WHERE user_id = $1
			  AND conversation_id = $2
			  AND left_at IS NULL
			  AND removed_at IS NULL
		)
	`

	var isParticipant bool
	err := r.db.QueryRow(ctx, query, userID, conversationID).Scan(&isParticipant)
	if err != nil {
		r.log.Error("Failed to check if user is participant",
			logger.String("user_id", userID.String()),
			logger.String("conversation_id", conversationID.String()),
			logger.Error(err),
		)
		return false, err
	}

	r.log.Debug("User participant check completed",
		logger.String("user_id", userID.String()),
		logger.String("conversation_id", conversationID.String()),
		logger.Bool("is_participant", isParticipant),
	)

	return isParticipant, nil
}

// IsUserBlocked checks if the user is blocked by any participant in the conversation
// This prevents blocked users from subscribing to conversations
func (r *conversationRepository) IsUserBlocked(ctx context.Context, userID, conversationID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM messages.conversation_participants cp
			INNER JOIN users.blocked_users bu
			  ON bu.user_id = cp.user_id
			  AND bu.blocked_user_id = $1
			WHERE cp.conversation_id = $2
			  AND cp.left_at IS NULL
			  AND cp.removed_at IS NULL
			  AND bu.unblocked_at IS NULL
		)
	`

	var isBlocked bool
	err := r.db.QueryRow(ctx, query, userID, conversationID).Scan(&isBlocked)
	if err != nil {
		r.log.Error("Failed to check if user is blocked",
			logger.String("user_id", userID.String()),
			logger.String("conversation_id", conversationID.String()),
			logger.Error(err),
		)
		return false, err
	}

	r.log.Debug("User blocked check completed",
		logger.String("user_id", userID.String()),
		logger.String("conversation_id", conversationID.String()),
		logger.Bool("is_blocked", isBlocked),
	)

	return isBlocked, nil
}

// GetUserConversations returns all active conversation IDs for a user
func (r *conversationRepository) GetUserConversations(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	query := `
		SELECT conversation_id
		FROM messages.conversation_participants
		WHERE user_id = $1
		  AND left_at IS NULL
		  AND removed_at IS NULL
		ORDER BY joined_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		r.log.Error("Failed to get user conversations",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return nil, err
	}
	defer rows.Close()

	var conversations []uuid.UUID
	for rows.Next() {
		var convID uuid.UUID
		if err := rows.Scan(&convID); err != nil {
			r.log.Error("Failed to scan conversation ID",
				logger.String("user_id", userID.String()),
				logger.Error(err),
			)
			return nil, err
		}
		conversations = append(conversations, convID)
	}

	if err := rows.Err(); err != nil {
		r.log.Error("Error iterating over conversations",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return nil, err
	}

	r.log.Debug("Retrieved user conversations",
		logger.String("user_id", userID.String()),
		logger.Int("count", len(conversations)),
	)

	return conversations, nil
}

// ValidateConversationAccess validates if user can access the conversation
// Checks: participant status, not left, not removed, not blocked
func (r *conversationRepository) ValidateConversationAccess(ctx context.Context, userID, conversationID uuid.UUID) (bool, error) {
	// First check if conversation exists and is active
	exists, err := r.ConversationExists(ctx, conversationID)
	if err != nil {
		return false, err
	}
	if !exists {
		r.log.Debug("Conversation validation failed: conversation does not exist or is inactive",
			logger.String("conversation_id", conversationID.String()),
		)
		return false, nil
	}

	// Check if user is an active participant
	isParticipant, err := r.IsUserParticipant(ctx, userID, conversationID)
	if err != nil {
		return false, err
	}
	if !isParticipant {
		r.log.Debug("Conversation validation failed: user is not an active participant",
			logger.String("user_id", userID.String()),
			logger.String("conversation_id", conversationID.String()),
		)
		return false, nil
	}

	// Check if user is blocked
	isBlocked, err := r.IsUserBlocked(ctx, userID, conversationID)
	if err != nil {
		return false, err
	}
	if isBlocked {
		r.log.Debug("Conversation validation failed: user is blocked by a participant",
			logger.String("user_id", userID.String()),
			logger.String("conversation_id", conversationID.String()),
		)
		return false, nil
	}

	r.log.Debug("Conversation access validation successful",
		logger.String("user_id", userID.String()),
		logger.String("conversation_id", conversationID.String()),
	)

	return true, nil
}

// ConversationExists checks if a conversation exists and is active
func (r *conversationRepository) ConversationExists(ctx context.Context, conversationID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM messages.conversations
			WHERE id = $1
			  AND is_active = true
			  AND deleted_at IS NULL
		)
	`

	var exists bool
	err := r.db.QueryRow(ctx, query, conversationID).Scan(&exists)
	if err != nil {
		r.log.Error("Failed to check if conversation exists",
			logger.String("conversation_id", conversationID.String()),
			logger.Error(err),
		)
		return false, err
	}

	r.log.Debug("Conversation existence check completed",
		logger.String("conversation_id", conversationID.String()),
		logger.Bool("exists", exists),
	)

	return exists, nil
}

// GetUserConversationSyncState returns sync metadata for conversations with unread messages
func (r *conversationRepository) GetUserConversationSyncState(ctx context.Context, userID uuid.UUID) ([]ConversationSyncState, error) {
	query := `
		SELECT cp.conversation_id, COALESCE(cp.unread_count, 0), c.last_message_id, c.last_message_at
		FROM messages.conversation_participants cp
		JOIN messages.conversations c ON c.id = cp.conversation_id
		WHERE cp.user_id = $1
		  AND cp.left_at IS NULL
		  AND cp.removed_at IS NULL
		  AND c.is_active = true
		  AND c.deleted_at IS NULL
		  AND COALESCE(cp.unread_count, 0) > 0
		ORDER BY c.last_message_at DESC NULLS LAST
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		r.log.Error("Failed to get conversation sync state",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return nil, err
	}
	defer rows.Close()

	var states []ConversationSyncState
	for rows.Next() {
		var state ConversationSyncState
		var lastMsgID *string
		if err := rows.Scan(&state.ConversationID, &state.UnreadCount, &lastMsgID, &state.LastMessageAt); err != nil {
			r.log.Error("Failed to scan conversation sync state",
				logger.String("user_id", userID.String()),
				logger.Error(err),
			)
			return nil, err
		}
		if lastMsgID != nil {
			parsed, err := uuid.Parse(*lastMsgID)
			if err == nil {
				state.LastMessageID = &parsed
			}
		}
		states = append(states, state)
	}

	if err := rows.Err(); err != nil {
		r.log.Error("Error iterating conversation sync states",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return nil, err
	}

	r.log.Debug("Retrieved conversation sync state",
		logger.String("user_id", userID.String()),
		logger.Int("count", len(states)),
	)

	return states, nil
}
