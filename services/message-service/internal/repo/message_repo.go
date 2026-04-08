package repo

import (
	"context"

	"shared/pkg/cache"
	"shared/pkg/database"
	dbModels "shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"

	"shared/pkg/logger"

	"github.com/lib/pq"
)

// MessageRepository handles core message CRUD operations.
type MessageRepository interface {
	InsertMessage(ctx context.Context, message *dbModels.Message) pkgErrors.AppError
	InsertMessages(ctx context.Context, messages []*dbModels.Message) pkgErrors.AppError
	GetMessagesByConversationID(ctx context.Context, conversationID string, limit int, beforeMessageID *string) ([]*dbModels.Message, pkgErrors.AppError)
	GetMessagesAfterID(ctx context.Context, conversationID string, afterMessageID string, limit int) ([]*dbModels.Message, pkgErrors.AppError)
	GetMessageByID(ctx context.Context, messageID string) (*dbModels.Message, pkgErrors.AppError)
	GetMessageCount(ctx context.Context, conversationID string) (int64, pkgErrors.AppError)
}

type messageRepository struct {
	db     database.Database
	cache  cache.Cache
	logger logger.Logger
}

func NewMessageRepository(db database.Database, cache cache.Cache, logger logger.Logger) MessageRepository {
	return &messageRepository{db: db, cache: cache, logger: logger}
}

func (r *messageRepository) InsertMessage(ctx context.Context, message *dbModels.Message) pkgErrors.AppError {
	tx, dbErr := r.db.BeginTx(ctx, nil)
	if dbErr != nil {
		return pkgErrors.FromError(dbErr, pkgErrors.CodeInternal, "failed to begin transaction")
	}

	var txErr *database.DBError
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if txErr != nil {
			tx.Rollback()
		}
	}()

	query := `
		INSERT INTO messages.messages (
			id,
			conversation_id,
			sender_user_id,
			parent_message_id,
			message_type,
			content,
			content_encrypted,
			content_hash,
			format_type,
			mentions,
			hashtags,
			links,
			status,
			delivered_at,
			delivery_count,
			read_count,
			reply_count,
			last_reply_at,
			reaction_count,
			is_forwarded,
			forwarded_from_message_id,
			forward_count,
			sent_from_device_id,
			sent_from_ip,
			created_at,
			updated_at,
			metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, $22, $23, $24::inet, $25, $26, $27
		)
		ON CONFLICT (id) DO NOTHING
	`

	_, txErr = tx.Exec(ctx, query,
		message.ID,
		message.ConversationID,
		message.SenderUserID,
		message.ParentMessageID,
		string(message.MessageType),
		message.Content,
		message.ContentEncrypted,
		message.ContentHash,
		string(message.FormatType),
		message.Mentions,
		pq.Array(message.Hashtags),
		message.Links,
		string(message.Status),
		message.DeliveredAt,
		message.DeliveryCount,
		message.ReadCount,
		message.ReplyCount,
		message.LastReplyAt,
		message.ReactionCount,
		message.IsForwarded,
		message.ForwardedFromMessageID,
		message.ForwardCount,
		message.SentFromDeviceID,
		message.SentFromIP,
		message.CreatedAt,
		message.UpdatedAt,
		message.Metadata,
	)

	if txErr != nil {
		r.logger.Error("failed to insert message",
			logger.String("message_id", message.ID),
			logger.String("conversation_id", message.ConversationID),
			logger.Error(txErr),
		)
		return pkgErrors.FromError(txErr, pkgErrors.CodeDatabaseError, "failed to insert message")
	}

	if txErr = tx.Commit(); txErr != nil {
		r.logger.Error("failed to commit transaction",
			logger.String("message_id", message.ID),
			logger.Error(txErr),
		)
		return pkgErrors.FromError(txErr, pkgErrors.CodeInternal, "failed to commit transaction")
	}

	r.logger.Debug("message inserted successfully",
		logger.String("message_id", message.ID),
		logger.String("conversation_id", message.ConversationID),
	)

	return nil
}

func (r *messageRepository) InsertMessages(ctx context.Context, messages []*dbModels.Message) pkgErrors.AppError {
	tx, dbErr := r.db.BeginTx(ctx, nil)
	if dbErr != nil {
		return pkgErrors.FromError(dbErr, pkgErrors.CodeInternal, "failed to begin transaction")
	}

	var txErr *database.DBError
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if txErr != nil {
			tx.Rollback()
		}
	}()

	query := `
		INSERT INTO messages.messages (
			id,
			conversation_id,
			sender_user_id,
			parent_message_id,
			message_type,
			content,
			content_encrypted,
			content_hash,
			format_type,
			mentions,
			hashtags,
			links,
			status,
			delivered_at,
			delivery_count,
			read_count,
			reply_count,
			last_reply_at,
			reaction_count,
			is_forwarded,
			forwarded_from_message_id,
			forward_count,
			sent_from_device_id,
			sent_from_ip,
			created_at,
			updated_at,
			metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
			$21, $22, $23, $24::inet, $25, $26, $27
		)
		ON CONFLICT (id) DO NOTHING
	`

	for _, event := range messages {
		_, txErr = tx.Exec(ctx, query,
			event.ID,
			event.ConversationID,
			event.SenderUserID,
			event.ParentMessageID,
			string(event.MessageType),
			event.Content,
			event.ContentEncrypted,
			event.ContentHash,
			string(event.FormatType),
			event.Mentions,
			pq.Array(event.Hashtags),
			event.Links,
			string(event.Status),
			event.DeliveredAt,
			event.DeliveryCount,
			event.ReadCount,
			event.ReplyCount,
			event.LastReplyAt,
			event.ReactionCount,
			event.IsForwarded,
			event.ForwardedFromMessageID,
			event.ForwardCount,
			event.SentFromDeviceID,
			event.SentFromIP,
			event.CreatedAt,
			event.UpdatedAt,
			event.Metadata,
		)

		if txErr != nil {
			r.logger.Error("failed to insert message in batch",
				logger.String("message_id", event.ID),
				logger.String("conversation_id", event.ConversationID),
				logger.Error(txErr),
			)
			return pkgErrors.FromError(txErr, pkgErrors.CodeDatabaseError, "failed to insert message in batch")
		}
	}

	if txErr = tx.Commit(); txErr != nil {
		r.logger.Error("failed to commit transaction for batch insert",
			logger.Error(txErr),
		)
		return pkgErrors.FromError(txErr, pkgErrors.CodeInternal, "failed to commit transaction")
	}

	r.logger.Debug("batch of messages inserted successfully",
		logger.Int("batch_size", len(messages)),
	)

	return nil
}

// GetMessagesByConversationID retrieves messages for a conversation with cursor-based pagination
func (r *messageRepository) GetMessagesByConversationID(ctx context.Context, conversationID string, limit int, beforeMessageID *string) ([]*dbModels.Message, pkgErrors.AppError) {
	r.logger.Debug("fetching messages by conversation ID",
		logger.String("conversation_id", conversationID),
		logger.Int("limit", limit),
	)

	var query string
	var args []interface{}

	if beforeMessageID != nil && *beforeMessageID != "" {
		// Cursor-based pagination: get messages before a specific message
		query = `
			SELECT
				id, conversation_id, sender_user_id, parent_message_id,
				message_type, content, content_encrypted, content_hash,
				format_type, mentions, hashtags, links, status,
				delivered_at, delivery_count, read_count, reply_count,
				last_reply_at, reaction_count, is_forwarded,
				forwarded_from_message_id, forward_count, sent_from_device_id,
				sent_from_ip, created_at, updated_at, edited_at, deleted_at, metadata
			FROM messages.messages
			WHERE conversation_id = $1
			  AND deleted_at IS NULL
			  AND created_at < (SELECT created_at FROM messages.messages WHERE id = $2)
			ORDER BY created_at DESC
			LIMIT $3
		`
		args = []interface{}{conversationID, *beforeMessageID, limit}
	} else {
		// No cursor: get most recent messages
		query = `
			SELECT
				id, conversation_id, sender_user_id, parent_message_id,
				message_type, content, content_encrypted, content_hash,
				format_type, mentions, hashtags, links, status,
				delivered_at, delivery_count, read_count, reply_count,
				last_reply_at, reaction_count, is_forwarded,
				forwarded_from_message_id, forward_count, sent_from_device_id,
				sent_from_ip, created_at, updated_at, edited_at, deleted_at, metadata
			FROM messages.messages
			WHERE conversation_id = $1
			  AND deleted_at IS NULL
			ORDER BY created_at DESC
			LIMIT $2
		`
		args = []interface{}{conversationID, limit}
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		r.logger.Error("failed to query messages",
			logger.String("conversation_id", conversationID),
			logger.Error(err),
		)
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to query messages")
	}
	defer rows.Close()

	var messages []*dbModels.Message
	for rows.Next() {
		var msg dbModels.Message
		if err := rows.Scan(
			&msg.ID, &msg.ConversationID, &msg.SenderUserID, &msg.ParentMessageID,
			&msg.MessageType, &msg.Content, &msg.ContentEncrypted, &msg.ContentHash,
			&msg.FormatType, &msg.Mentions, pq.Array(&msg.Hashtags), &msg.Links,
			&msg.Status, &msg.DeliveredAt, &msg.DeliveryCount, &msg.ReadCount,
			&msg.ReplyCount, &msg.LastReplyAt, &msg.ReactionCount, &msg.IsForwarded,
			&msg.ForwardedFromMessageID, &msg.ForwardCount, &msg.SentFromDeviceID,
			&msg.SentFromIP, &msg.CreatedAt, &msg.UpdatedAt, &msg.EditedAt, &msg.DeletedAt, &msg.Metadata,
		); err != nil {
			r.logger.Error("failed to scan message row",
				logger.String("conversation_id", conversationID),
				logger.Error(err),
			)
			return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to scan message row")
		}
		messages = append(messages, &msg)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("error iterating messages",
			logger.String("conversation_id", conversationID),
			logger.Error(err),
		)
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "error iterating messages")
	}

	r.logger.Debug("messages fetched successfully",
		logger.String("conversation_id", conversationID),
		logger.Int("count", len(messages)),
	)

	return messages, nil
}

// GetMessagesAfterID retrieves messages created after a specific message, in chronological order (ASC)
func (r *messageRepository) GetMessagesAfterID(ctx context.Context, conversationID string, afterMessageID string, limit int) ([]*dbModels.Message, pkgErrors.AppError) {
	r.logger.Debug("fetching messages after ID",
		logger.String("conversation_id", conversationID),
		logger.String("after_message_id", afterMessageID),
		logger.Int("limit", limit),
	)

	query := `
		SELECT
			id, conversation_id, sender_user_id, parent_message_id,
			message_type, content, content_encrypted, content_hash,
			format_type, mentions, hashtags, links, status,
			delivered_at, delivery_count, read_count, reply_count,
			last_reply_at, reaction_count, is_forwarded,
			forwarded_from_message_id, forward_count, sent_from_device_id,
			sent_from_ip, created_at, updated_at, edited_at, deleted_at, metadata
		FROM messages.messages
		WHERE conversation_id = $1
		  AND deleted_at IS NULL
		  AND created_at > (SELECT created_at FROM messages.messages WHERE id = $2)
		ORDER BY created_at ASC
		LIMIT $3
	`

	rows, err := r.db.Query(ctx, query, conversationID, afterMessageID, limit)
	if err != nil {
		r.logger.Error("failed to query messages after ID",
			logger.String("conversation_id", conversationID),
			logger.String("after_message_id", afterMessageID),
			logger.Error(err),
		)
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to query messages")
	}
	defer rows.Close()

	var messages []*dbModels.Message
	for rows.Next() {
		var msg dbModels.Message
		if err := rows.Scan(
			&msg.ID, &msg.ConversationID, &msg.SenderUserID, &msg.ParentMessageID,
			&msg.MessageType, &msg.Content, &msg.ContentEncrypted, &msg.ContentHash,
			&msg.FormatType, &msg.Mentions, pq.Array(&msg.Hashtags), &msg.Links,
			&msg.Status, &msg.DeliveredAt, &msg.DeliveryCount, &msg.ReadCount,
			&msg.ReplyCount, &msg.LastReplyAt, &msg.ReactionCount, &msg.IsForwarded,
			&msg.ForwardedFromMessageID, &msg.ForwardCount, &msg.SentFromDeviceID,
			&msg.SentFromIP, &msg.CreatedAt, &msg.UpdatedAt, &msg.EditedAt, &msg.DeletedAt, &msg.Metadata,
		); err != nil {
			r.logger.Error("failed to scan message row",
				logger.String("conversation_id", conversationID),
				logger.Error(err),
			)
			return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to scan message row")
		}
		messages = append(messages, &msg)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("error iterating messages after ID",
			logger.String("conversation_id", conversationID),
			logger.Error(err),
		)
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "error iterating messages")
	}

	r.logger.Debug("messages after ID fetched successfully",
		logger.String("conversation_id", conversationID),
		logger.String("after_message_id", afterMessageID),
		logger.Int("count", len(messages)),
	)

	return messages, nil
}

// GetMessageByID retrieves a single message by its ID
func (r *messageRepository) GetMessageByID(ctx context.Context, messageID string) (*dbModels.Message, pkgErrors.AppError) {
	r.logger.Debug("fetching message by ID",
		logger.String("message_id", messageID),
	)

	query := `
		SELECT
			id, conversation_id, sender_user_id, parent_message_id,
			message_type, content, content_encrypted, content_hash,
			format_type, mentions, hashtags, links, status,
			delivered_at, delivery_count, read_count, reply_count,
			last_reply_at, reaction_count, is_forwarded,
			forwarded_from_message_id, forward_count, sent_from_device_id,
			sent_from_ip, created_at, updated_at, edited_at, deleted_at, metadata
		FROM messages.messages
		WHERE id = $1 AND deleted_at IS NULL
	`

	var message dbModels.Message
	err := r.db.QueryRow(ctx, query, messageID).ScanModel(&message)
	if err != nil {
		r.logger.Error("failed to fetch message",
			logger.String("message_id", messageID),
			logger.Error(err),
		)
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to fetch message")
	}

	r.logger.Debug("message fetched successfully",
		logger.String("message_id", messageID),
	)

	return &message, nil
}

// GetMessageCount returns the total count of messages in a conversation
func (r *messageRepository) GetMessageCount(ctx context.Context, conversationID string) (int64, pkgErrors.AppError) {
	r.logger.Debug("counting messages",
		logger.String("conversation_id", conversationID),
	)

	query := `
		SELECT COUNT(*)
		FROM messages.messages
		WHERE conversation_id = $1 AND deleted_at IS NULL
	`

	var count int64
	err := r.db.QueryRow(ctx, query, conversationID).Scan(&count)
	if err != nil {
		r.logger.Error("failed to count messages",
			logger.String("conversation_id", conversationID),
			logger.Error(err),
		)
		return 0, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to count messages")
	}

	r.logger.Debug("messages counted successfully",
		logger.String("conversation_id", conversationID),
		logger.Int64("count", count),
	)

	return count, nil
}
