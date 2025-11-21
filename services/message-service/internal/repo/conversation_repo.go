package repo

import (
	"context"
	"database/sql"
	"echo-backend/services/message-service/api/v1/dto"
	"echo-backend/services/message-service/internal/models"
	"time"

	"shared/pkg/database"
	pkgErrors "shared/pkg/errors"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type ConversationRepository interface {
	// Conversation operations
	CreateConversation(ctx context.Context, conversationType, title, description string, creatorUserID uuid.UUID, isEncrypted, isPublic bool) (uuid.UUID, pkgErrors.AppError)
	AddParticipants(ctx context.Context, conversationID uuid.UUID, userIDs []uuid.UUID, role string, canSendMessages bool) pkgErrors.AppError
	GetConversationsByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]dto.ConversationResponse, int, pkgErrors.AppError)
	GetConversationByID(ctx context.Context, conversationID uuid.UUID) (*models.Conversation, pkgErrors.AppError)
}

type conversationRepository struct {
	db database.Database
}

func NewConversationRepository(db database.Database) ConversationRepository {
	return &conversationRepository{db: db}
}

// CreateConversation creates a new conversation
func (r *conversationRepository) CreateConversation(ctx context.Context, conversationType, title, description string, creatorUserID uuid.UUID, isEncrypted, isPublic bool) (uuid.UUID, pkgErrors.AppError) {
	query := `
		INSERT INTO messages.conversations (
			id, conversation_type, title, description, creator_user_id,
			is_encrypted, is_public, member_count, message_count, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		RETURNING id
	`

	conversationID := uuid.New()
	memberCount := 1 // Start with creator

	err := r.db.QueryRow(ctx, query,
		conversationID,
		conversationType,
		title,
		description,
		creatorUserID,
		isEncrypted,
		isPublic,
		memberCount,
		0, // initial message count
	).Scan(&conversationID)

	if err != nil {
		return uuid.Nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to create conversation").
			WithDetail("conversation_type", conversationType).
			WithDetail("creator_user_id", creatorUserID.String())
	}

	return conversationID, nil
}

// AddParticipants adds participants to a conversation
func (r *conversationRepository) AddParticipants(ctx context.Context, conversationID uuid.UUID, userIDs []uuid.UUID, role string, canSendMessages bool) pkgErrors.AppError {
	if len(userIDs) == 0 {
		return nil
	}

	query := `
		INSERT INTO messages.conversation_participants (
			id, conversation_id, user_id, role, can_send_messages, unread_count, joined_at, created_at, updated_at
		)
		SELECT gen_random_uuid(), $1, unnest($2::uuid[]), $3, $4, 0, NOW(), NOW(), NOW()
		ON CONFLICT (conversation_id, user_id) DO NOTHING
	`

	_, err := r.db.Exec(ctx, query, conversationID, pq.Array(userIDs), role, canSendMessages)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to add participants").
			WithDetail("conversation_id", conversationID.String()).
			WithDetail("participant_count", len(userIDs))
	}

	// Update member count
	updateQuery := `
		UPDATE messages.conversations
		SET member_count = (
			SELECT COUNT(*) FROM messages.conversation_participants
			WHERE conversation_id = $1 AND left_at IS NULL
		), updated_at = NOW()
		WHERE id = $1
	`

	_, err = r.db.Exec(ctx, updateQuery, conversationID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to update member count").
			WithDetail("conversation_id", conversationID.String())
	}

	return nil
}

// GetConversationsByUserID retrieves all conversations for a user
func (r *conversationRepository) GetConversationsByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]dto.ConversationResponse, int, pkgErrors.AppError) {
	// First get total count
	countQuery := `
		SELECT COUNT(DISTINCT c.id)
		FROM messages.conversations c
		INNER JOIN messages.conversation_participants cp ON c.id = cp.conversation_id
		WHERE cp.user_id = $1 AND cp.left_at IS NULL
	`

	var total int
	err := r.db.QueryRow(ctx, countQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to count conversations").
			WithDetail("user_id", userID.String())
	}

	// Get conversations with participant's unread count
	query := `
		SELECT
			c.id,
			c.conversation_type,
			c.title,
			c.avatar_url,
			c.is_encrypted,
			c.is_public,
			c.member_count,
			COALESCE(cp.unread_count, 0) as unread_count,
			c.last_message_at,
			c.created_at
		FROM messages.conversations c
		INNER JOIN messages.conversation_participants cp ON c.id = cp.conversation_id
		WHERE cp.user_id = $1 AND cp.left_at IS NULL
		ORDER BY COALESCE(c.last_message_at, c.created_at) DESC
		LIMIT $2 OFFSET $3
	`

	rows, dbErr := r.db.Query(ctx, query, userID, limit, offset)
	if dbErr != nil {
		return nil, 0, pkgErrors.FromError(dbErr, pkgErrors.CodeDatabaseError, "failed to query conversations").
			WithDetail("user_id", userID.String())
	}
	defer rows.Close()

	var conversations []dto.ConversationResponse
	for rows.Next() {
		var conv dto.ConversationResponse
		var lastMessageAt *time.Time
		var createdAt time.Time

		err := rows.Scan(
			&conv.ID,
			&conv.ConversationType,
			&conv.Title,
			&conv.AvatarURL,
			&conv.IsEncrypted,
			&conv.IsPublic,
			&conv.MemberCount,
			&conv.UnreadCount,
			&lastMessageAt,
			&createdAt,
		)
		if err != nil {
			return nil, 0, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to scan conversation").
				WithDetail("user_id", userID.String())
		}

		if lastMessageAt != nil {
			timestamp := lastMessageAt.Unix()
			conv.LastMessageAt = &timestamp
		}
		conv.CreatedAt = createdAt.Unix()

		conversations = append(conversations, conv)
	}

	return conversations, total, nil
}

// GetConversationByID retrieves a conversation by ID
func (r *conversationRepository) GetConversationByID(ctx context.Context, conversationID uuid.UUID) (*models.Conversation, pkgErrors.AppError) {
	query := `
		SELECT
			id, conversation_type, title, description, avatar_url,
			creator_user_id, is_encrypted, is_public, member_count,
			message_count, updated_at
		FROM messages.conversations
		WHERE id = $1
	`

	var conv models.Conversation
	err := r.db.QueryRow(ctx, query, conversationID).Scan(
		&conv.ID,
		&conv.ConversationType,
		&conv.Title,
		&conv.Description,
		&conv.AvatarURL,
		&conv.CreatorUserID,
		&conv.IsEncrypted,
		&conv.IsPublic,
		&conv.MemberCount,
		&conv.MessageCount,
		&conv.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, pkgErrors.FromError(err, pkgErrors.CodeNotFound, "conversation not found").
				WithDetail("conversation_id", conversationID.String())
		}

		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get conversation").
			WithDetail("conversation_id", conversationID.String())
	}

	return &conv, nil
}
