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

type MessageRepository interface {
	InsertMessage(ctx context.Context, message *dbModels.Message) pkgErrors.AppError
	InsertMessages(ctx context.Context, messages []*dbModels.Message) pkgErrors.AppError
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

	var txErr error
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

	var txErr error
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
