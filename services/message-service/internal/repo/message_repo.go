package repo

import (
	"context"
	"database/sql"
	"echo-backend/services/message-service/internal/models"
	"fmt"

	"shared/pkg/database"
	pkgErrors "shared/pkg/errors"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type MessageRepository interface {
	// Core message operations
	CreateMessage(ctx context.Context, msg *models.Message) pkgErrors.AppError
	GetMessageByID(ctx context.Context, messageID uuid.UUID) (*models.Message, pkgErrors.AppError)
	GetMessages(ctx context.Context, conversationID uuid.UUID, params *models.PaginationParams) ([]models.Message, pkgErrors.AppError)
	UpdateMessage(ctx context.Context, messageID uuid.UUID, content string) pkgErrors.AppError
	DeleteMessage(ctx context.Context, messageID uuid.UUID, userID uuid.UUID) pkgErrors.AppError

	// Delivery tracking
	CreateDeliveryStatus(ctx context.Context, messageID uuid.UUID, userIDs []uuid.UUID) pkgErrors.AppError
	UpdateDeliveryStatus(ctx context.Context, messageID, userID uuid.UUID, status string) pkgErrors.AppError
	MarkAsDelivered(ctx context.Context, messageID, userID uuid.UUID) pkgErrors.AppError
	MarkAsRead(ctx context.Context, messageID, userID uuid.UUID) pkgErrors.AppError
	GetDeliveryStatus(ctx context.Context, messageID uuid.UUID) ([]models.DeliveryStatus, pkgErrors.AppError)

	// Conversation operations
	GetConversationParticipants(ctx context.Context, conversationID uuid.UUID) ([]models.ConversationParticipant, pkgErrors.AppError)
	GetParticipantUserIDs(ctx context.Context, conversationID uuid.UUID) ([]uuid.UUID, pkgErrors.AppError)
	ValidateParticipant(ctx context.Context, conversationID, userID uuid.UUID) (bool, pkgErrors.AppError)
	UpdateConversationLastMessage(ctx context.Context, conversationID, messageID uuid.UUID) pkgErrors.AppError
	UpdateParticipantUnreadCount(ctx context.Context, conversationID, userID uuid.UUID, increment bool) pkgErrors.AppError
	ResetUnreadCount(ctx context.Context, conversationID, userID uuid.UUID) pkgErrors.AppError

	// Typing indicators
	SetTypingIndicator(ctx context.Context, conversationID, userID uuid.UUID, isTyping bool) pkgErrors.AppError
	GetTypingUsers(ctx context.Context, conversationID uuid.UUID) ([]uuid.UUID, pkgErrors.AppError)
}

type messageRepository struct {
	db database.Database
}

func NewMessageRepository(db database.Database) MessageRepository {
	return &messageRepository{db: db}
}

// CreateMessage creates a new message in the database
func (r *messageRepository) CreateMessage(ctx context.Context, msg *models.Message) pkgErrors.AppError {
	query := `
		INSERT INTO messages.messages (
			id, conversation_id, sender_user_id, parent_message_id,
			content, message_type, status, mentions, metadata, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at
	`

	// Mentions and Metadata are already json.RawMessage from the service layer
	row := r.db.QueryRow(ctx, query,
		msg.ID,
		msg.ConversationID,
		msg.SenderUserID,
		msg.ParentMessageID,
		msg.Content,
		msg.MessageType,
		msg.Status,
		msg.Mentions,
		msg.Metadata,
		msg.CreatedAt,
		msg.UpdatedAt,
	)
	err := row.Scan(&msg.ID, &msg.CreatedAt, &msg.UpdatedAt)

	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to create message").
			WithDetail("message_id", msg.ID.String()).
			WithDetail("conversation_id", msg.ConversationID.String())
	}

	return nil
}

// GetMessageByID retrieves a single message by ID
func (r *messageRepository) GetMessageByID(ctx context.Context, messageID uuid.UUID) (*models.Message, pkgErrors.AppError) {
	query := `
		SELECT id, conversation_id, sender_user_id, parent_message_id,
		       content, message_type, status, is_edited, is_deleted,
		       mentions, metadata, created_at, updated_at, deleted_at, edited_at
		FROM messages.messages
		WHERE id = $1 AND is_deleted = FALSE
	`

	msg := &models.Message{}
	err := r.db.QueryRow(ctx, query, messageID).Scan(
		&msg.ID,
		&msg.ConversationID,
		&msg.SenderUserID,
		&msg.ParentMessageID,
		&msg.Content,
		&msg.MessageType,
		&msg.Status,
		&msg.IsEdited,
		&msg.IsDeleted,
		&msg.Mentions,
		&msg.Metadata,
		&msg.CreatedAt,
		&msg.UpdatedAt,
		&msg.DeletedAt,
		&msg.EditedAt,
	)

	if err == sql.ErrNoRows {
		return nil, pkgErrors.New(pkgErrors.CodeNotFound, "message not found").
			WithDetail("message_id", messageID.String())
	}
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to get message").
			WithDetail("message_id", messageID.String())
	}

	return msg, nil
}

// GetMessages retrieves messages for a conversation with pagination
func (r *messageRepository) GetMessages(ctx context.Context, conversationID uuid.UUID, params *models.PaginationParams) ([]models.Message, pkgErrors.AppError) {
	if params.Limit == 0 {
		params.Limit = 50
	}
	if params.Limit > 100 {
		params.Limit = 100
	}

	// Begin transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to begin transaction").
			WithDetail("conversation_id", conversationID.String())
	}
	defer func() { _ = tx.Rollback() }()

	var exists bool
	existsQuery := `SELECT EXISTS(SELECT 1 FROM messages.conversations WHERE id = $1)`
	if err := tx.QueryRow(ctx, existsQuery, conversationID).Scan(&exists); err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to check conversation existence").
			WithDetail("conversation_id", conversationID.String())
	}
	if !exists {
		return nil, pkgErrors.New(pkgErrors.CodeNotFound, "conversation not found").
			WithDetail("conversation_id", conversationID.String())
	}

	query := `
		SELECT m.id, m.conversation_id, m.sender_user_id, m.parent_message_id,
		       m.content, m.message_type, m.status, m.is_edited, m.is_deleted,
		       m.mentions, m.metadata, m.created_at, m.updated_at, m.deleted_at, m.edited_at,
		       COUNT(ds.id) FILTER (WHERE ds.status = 'read') as read_count
		FROM messages.messages m
		LEFT JOIN messages.delivery_status ds ON m.id = ds.message_id
		WHERE m.conversation_id = $1 AND m.is_deleted = FALSE
	`

	args := []interface{}{conversationID}
	argIdx := 2

	if params.BeforeID != nil {
		query += fmt.Sprintf(` AND m.created_at < (SELECT created_at FROM messages.messages WHERE id = $%d)`, argIdx)
		args = append(args, *params.BeforeID)
		argIdx++
	}

	if params.AfterID != nil {
		query += fmt.Sprintf(` AND m.created_at > (SELECT created_at FROM messages.messages WHERE id = $%d)`, argIdx)
		args = append(args, *params.AfterID)
		argIdx++
	}

	query += `
		GROUP BY m.id
		ORDER BY m.created_at DESC
		LIMIT $` + fmt.Sprintf("%d", argIdx)
	args = append(args, params.Limit)

	rows, dbErr := tx.Query(ctx, query, args...)
	if dbErr != nil {
		return nil, pkgErrors.FromError(dbErr, pkgErrors.CodeDatabaseError, "failed to query messages").
			WithDetail("conversation_id", conversationID.String()).
			WithDetail("limit", params.Limit)
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var msg models.Message
		err := rows.Scan(
			&msg.ID,
			&msg.ConversationID,
			&msg.SenderUserID,
			&msg.ParentMessageID,
			&msg.Content,
			&msg.MessageType,
			&msg.Status,
			&msg.IsEdited,
			&msg.IsDeleted,
			&msg.Mentions,
			&msg.Metadata,
			&msg.CreatedAt,
			&msg.UpdatedAt,
			&msg.DeletedAt,
			&msg.EditedAt,
			&msg.ReadCount,
		)
		if err != nil {
			return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to scan message").
				WithDetail("conversation_id", conversationID.String())
		}
		messages = append(messages, msg)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to commit transaction").
			WithDetail("conversation_id", conversationID.String())
	}

	return messages, nil
}

// UpdateMessage updates message content
func (r *messageRepository) UpdateMessage(ctx context.Context, messageID uuid.UUID, content string) pkgErrors.AppError {
	query := `
		UPDATE messages.messages
		SET content = $1, is_edited = TRUE, edited_at = NOW(), updated_at = NOW()
		WHERE id = $2 AND is_deleted = FALSE
	`

	result, err := r.db.Exec(ctx, query, content, messageID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to update message").
			WithDetail("message_id", messageID.String())
	}

	rows, dbErr := result.RowsAffected()
	if dbErr != nil {
		return pkgErrors.FromError(dbErr, pkgErrors.CodeDatabaseError, "failed to get affected rows").
			WithDetail("message_id", messageID.String())
	}

	if rows == 0 {
		return pkgErrors.New(pkgErrors.CodeNotFound, "message not found or already deleted").
			WithDetail("message_id", messageID.String())
	}

	return nil
}

// DeleteMessage soft deletes a message
func (r *messageRepository) DeleteMessage(ctx context.Context, messageID uuid.UUID, userID uuid.UUID) pkgErrors.AppError {
	query := `
		UPDATE messages.messages
		SET is_deleted = TRUE, deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND sender_user_id = $2 AND is_deleted = FALSE
	`

	result, err := r.db.Exec(ctx, query, messageID, userID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to delete message").
			WithDetail("message_id", messageID.String()).
			WithDetail("user_id", userID.String())
	}

	rows, dbErr := result.RowsAffected()
	if dbErr != nil {
		return pkgErrors.FromError(dbErr, pkgErrors.CodeDatabaseError, "failed to get affected rows").
			WithDetail("message_id", messageID.String())
	}

	if rows == 0 {
		return pkgErrors.New(pkgErrors.CodeNotFound, "message not found or unauthorized").
			WithDetail("message_id", messageID.String()).
			WithDetail("user_id", userID.String())
	}

	return nil
}

// CreateDeliveryStatus creates delivery status records for all participants
func (r *messageRepository) CreateDeliveryStatus(ctx context.Context, messageID uuid.UUID, userIDs []uuid.UUID) pkgErrors.AppError {
	if len(userIDs) == 0 {
		return nil
	}

	query := `
		INSERT INTO messages.delivery_status (id, message_id, user_id, status, created_at)
		SELECT gen_random_uuid(), $1, unnest($2::uuid[]), 'sent', NOW()
		ON CONFLICT (message_id, user_id) DO NOTHING
	`

	_, err := r.db.Exec(ctx, query, messageID, pq.Array(userIDs))
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to create delivery status").
			WithDetail("message_id", messageID.String()).
			WithDetail("recipient_count", len(userIDs))
	}

	return nil
}

// UpdateDeliveryStatus updates the delivery status for a message
func (r *messageRepository) UpdateDeliveryStatus(ctx context.Context, messageID, userID uuid.UUID, status string) pkgErrors.AppError {
	query := `
		UPDATE messages.delivery_status
		SET status = $1,
		    delivered_at = CASE WHEN $1 = 'delivered' AND delivered_at IS NULL THEN NOW() ELSE delivered_at END,
		    read_at = CASE WHEN $1 = 'read' AND read_at IS NULL THEN NOW() ELSE read_at END
		WHERE message_id = $2 AND user_id = $3
	`

	_, err := r.db.Exec(ctx, query, status, messageID, userID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to update delivery status").
			WithDetail("message_id", messageID.String()).
			WithDetail("user_id", userID.String()).
			WithDetail("status", status)
	}

	return nil
}

// MarkAsDelivered marks a message as delivered to a user
func (r *messageRepository) MarkAsDelivered(ctx context.Context, messageID, userID uuid.UUID) pkgErrors.AppError {
	return r.UpdateDeliveryStatus(ctx, messageID, userID, "delivered")
}

// MarkAsRead marks a message as read by a user
func (r *messageRepository) MarkAsRead(ctx context.Context, messageID, userID uuid.UUID) pkgErrors.AppError {
	return r.UpdateDeliveryStatus(ctx, messageID, userID, "read")
}

// GetDeliveryStatus gets all delivery statuses for a message
func (r *messageRepository) GetDeliveryStatus(ctx context.Context, messageID uuid.UUID) ([]models.DeliveryStatus, pkgErrors.AppError) {
	query := `
		SELECT id, message_id, user_id, status, delivered_at, read_at, created_at
		FROM messages.delivery_status
		WHERE message_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(ctx, query, messageID)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to query delivery status").
			WithDetail("message_id", messageID.String())
	}
	defer rows.Close()

	var statuses []models.DeliveryStatus
	for rows.Next() {
		var ds models.DeliveryStatus
		err := rows.Scan(
			&ds.ID,
			&ds.MessageID,
			&ds.UserID,
			&ds.Status,
			&ds.DeliveredAt,
			&ds.ReadAt,
			&ds.CreatedAt,
		)
		if err != nil {
			return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to scan delivery status").
				WithDetail("message_id", messageID.String())
		}
		statuses = append(statuses, ds)
	}

	return statuses, nil
}

// GetConversationParticipants gets all participants in a conversation
func (r *messageRepository) GetConversationParticipants(ctx context.Context, conversationID uuid.UUID) ([]models.ConversationParticipant, pkgErrors.AppError) {
	query := `
		SELECT id, conversation_id, user_id, role, can_send_messages,
		       last_read_message_id, last_read_at, unread_count, joined_at, left_at
		FROM messages.conversation_participants
		WHERE conversation_id = $1 AND left_at IS NULL
		ORDER BY joined_at ASC
	`

	rows, err := r.db.Query(ctx, query, conversationID)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to query participants").
			WithDetail("conversation_id", conversationID.String())
	}
	defer rows.Close()

	var participants []models.ConversationParticipant
	for rows.Next() {
		var p models.ConversationParticipant
		err := rows.Scan(
			&p.ID,
			&p.ConversationID,
			&p.UserID,
			&p.Role,
			&p.CanSendMessages,
			&p.LastReadMessageID,
			&p.LastReadAt,
			&p.UnreadCount,
			&p.JoinedAt,
			&p.LeftAt,
		)
		if err != nil {
			return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to scan participant").
				WithDetail("conversation_id", conversationID.String())
		}
		participants = append(participants, p)
	}

	return participants, nil
}

// GetParticipantUserIDs gets all user IDs in a conversation
func (r *messageRepository) GetParticipantUserIDs(ctx context.Context, conversationID uuid.UUID) ([]uuid.UUID, pkgErrors.AppError) {
	query := `
		SELECT user_id FROM messages.conversation_participants
		WHERE conversation_id = $1 AND left_at IS NULL
	`

	rows, err := r.db.Query(ctx, query, conversationID)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to query participant user IDs").
			WithDetail("conversation_id", conversationID.String())
	}
	defer rows.Close()

	var userIDs []uuid.UUID
	for rows.Next() {
		var userID uuid.UUID
		if err := rows.Scan(&userID); err != nil {
			return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to scan user ID").
				WithDetail("conversation_id", conversationID.String())
		}
		userIDs = append(userIDs, userID)
	}

	return userIDs, nil
}

// ValidateParticipant checks if a user is a participant in a conversation
func (r *messageRepository) ValidateParticipant(ctx context.Context, conversationID, userID uuid.UUID) (bool, pkgErrors.AppError) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM messages.conversation_participants
			WHERE conversation_id = $1 AND user_id = $2 AND left_at IS NULL AND can_send_messages = TRUE
		)
	`

	var exists bool
	err := r.db.QueryRow(ctx, query, conversationID, userID).Scan(&exists)
	if err != nil {
		return false, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to validate participant").
			WithDetail("conversation_id", conversationID.String()).
			WithDetail("user_id", userID.String())
	}

	return exists, nil
}

// UpdateConversationLastMessage updates conversation metadata
func (r *messageRepository) UpdateConversationLastMessage(ctx context.Context, conversationID, messageID uuid.UUID) pkgErrors.AppError {
	query := `
		UPDATE messages.conversations
		SET last_message_id = $1,
		    last_message_at = NOW(),
		    last_activity_at = NOW(),
		    message_count = message_count + 1,
		    updated_at = NOW()
		WHERE id = $2
	`

	_, err := r.db.Exec(ctx, query, messageID, conversationID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to update conversation").
			WithDetail("conversation_id", conversationID.String()).
			WithDetail("message_id", messageID.String())
	}

	return nil
}

// UpdateParticipantUnreadCount updates unread count for a participant
func (r *messageRepository) UpdateParticipantUnreadCount(ctx context.Context, conversationID, userID uuid.UUID, increment bool) pkgErrors.AppError {
	var query string
	if increment {
		query = `
			UPDATE messages.conversation_participants
			SET unread_count = unread_count + 1, updated_at = NOW()
			WHERE conversation_id = $1 AND user_id = $2
		`
	} else {
		query = `
			UPDATE messages.conversation_participants
			SET unread_count = GREATEST(unread_count - 1, 0), updated_at = NOW()
			WHERE conversation_id = $1 AND user_id = $2
		`
	}

	_, err := r.db.Exec(ctx, query, conversationID, userID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to update unread count").
			WithDetail("conversation_id", conversationID.String()).
			WithDetail("user_id", userID.String())
	}

	return nil
}

// ResetUnreadCount resets unread count for a participant
func (r *messageRepository) ResetUnreadCount(ctx context.Context, conversationID, userID uuid.UUID) pkgErrors.AppError {
	query := `
		UPDATE messages.conversation_participants
		SET unread_count = 0, last_read_at = NOW(), updated_at = NOW()
		WHERE conversation_id = $1 AND user_id = $2
	`

	_, err := r.db.Exec(ctx, query, conversationID, userID)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to reset unread count").
			WithDetail("conversation_id", conversationID.String()).
			WithDetail("user_id", userID.String())
	}

	return nil
}

// SetTypingIndicator sets typing indicator for a user in a conversation
func (r *messageRepository) SetTypingIndicator(ctx context.Context, conversationID, userID uuid.UUID, isTyping bool) pkgErrors.AppError {
	if isTyping {
		query := `
			INSERT INTO messages.typing_indicators (conversation_id, user_id, started_at, expires_at)
			VALUES ($1, $2, NOW(), NOW() + INTERVAL '10 seconds')
			ON CONFLICT (conversation_id, user_id) DO UPDATE SET
				started_at = NOW(),
				expires_at = NOW() + INTERVAL '10 seconds'
		`
		_, err := r.db.Exec(ctx, query, conversationID, userID)
		if err != nil {
			return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to set typing indicator").
				WithDetail("conversation_id", conversationID.String()).
				WithDetail("user_id", userID.String())
		}
		return nil
	} else {
		query := `DELETE FROM messages.typing_indicators WHERE conversation_id = $1 AND user_id = $2`
		_, err := r.db.Exec(ctx, query, conversationID, userID)
		if err != nil {
			return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to clear typing indicator").
				WithDetail("conversation_id", conversationID.String()).
				WithDetail("user_id", userID.String())
		}
		return nil
	}
}

// GetTypingUsers gets all users currently typing in a conversation
func (r *messageRepository) GetTypingUsers(ctx context.Context, conversationID uuid.UUID) ([]uuid.UUID, pkgErrors.AppError) {
	query := `
		SELECT user_id FROM messages.typing_indicators
		WHERE conversation_id = $1 AND expires_at > NOW()
	`

	rows, err := r.db.Query(ctx, query, conversationID)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to query typing users").
			WithDetail("conversation_id", conversationID.String())
	}
	defer rows.Close()

	var userIDs []uuid.UUID
	for rows.Next() {
		var userID uuid.UUID
		if err := rows.Scan(&userID); err != nil {
			return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to scan typing user").
				WithDetail("conversation_id", conversationID.String())
		}
		userIDs = append(userIDs, userID)
	}

	return userIDs, nil
}
