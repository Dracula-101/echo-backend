package repo

import (
	"context"
	"time"

	"shared/pkg/cache"
	"shared/pkg/database"
	pkgErrors "shared/pkg/errors"

	"shared/pkg/logger"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// DeliveryRepository handles message delivery status tracking operations.
type DeliveryRepository interface {
	CreateDeliveryStatuses(ctx context.Context, messageID string, recipientIDs []string) pkgErrors.AppError
	UpdateDeliveryStatus(ctx context.Context, messageID string, userID string, status string, timestamp *time.Time) pkgErrors.AppError
	UpdateDeliveryStatusBatch(ctx context.Context, messageIDs []string, userID string, status string, timestamp *time.Time) pkgErrors.AppError
	IncrementMessageDeliveryCount(ctx context.Context, messageID string) pkgErrors.AppError
	IncrementMessageReadCount(ctx context.Context, messageID string) pkgErrors.AppError
	GetDeliveryStatuses(ctx context.Context, messageID string) ([]DeliveryStatusRow, pkgErrors.AppError)
	MarkConversationAsRead(ctx context.Context, userID string, conversationID string, upToMessageID string) pkgErrors.AppError
}

type deliveryRepository struct {
	db     database.Database
	cache  cache.Cache
	logger logger.Logger
}

func NewDeliveryRepository(db database.Database, cache cache.Cache, logger logger.Logger) DeliveryRepository {
	return &deliveryRepository{db: db, cache: cache, logger: logger}
}

// CreateDeliveryStatuses creates delivery status records for all recipients
func (r *deliveryRepository) CreateDeliveryStatuses(ctx context.Context, messageID string, recipientIDs []string) pkgErrors.AppError {
	if len(recipientIDs) == 0 {
		return nil
	}

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
		INSERT INTO messages.delivery_status (
			id, message_id, user_id, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (message_id, user_id) DO NOTHING
	`

	for _, recipientID := range recipientIDs {
		id := uuid.New().String()
		_, txErr = tx.Exec(ctx, query, id, messageID, recipientID, "sent")
		if txErr != nil {
			r.logger.Error("failed to insert delivery status",
				logger.String("message_id", messageID),
				logger.String("recipient_id", recipientID),
				logger.Error(txErr),
			)
			return pkgErrors.FromError(txErr, pkgErrors.CodeDatabaseError, "failed to insert delivery status")
		}
	}

	if txErr = tx.Commit(); txErr != nil {
		r.logger.Error("failed to commit delivery status transaction",
			logger.String("message_id", messageID),
			logger.Error(txErr),
		)
		return pkgErrors.FromError(txErr, pkgErrors.CodeInternal, "failed to commit transaction")
	}

	r.logger.Debug("delivery statuses created successfully",
		logger.String("message_id", messageID),
		logger.Int("recipient_count", len(recipientIDs)),
	)

	return nil
}

// UpdateDeliveryStatus updates the delivery status for a specific user
func (r *deliveryRepository) UpdateDeliveryStatus(ctx context.Context, messageID string, userID string, status string, timestamp *time.Time) pkgErrors.AppError {
	var query string
	var args []interface{}

	if status == "delivered" {
		query = `
			UPDATE messages.delivery_status
			SET status = $1, delivered_at = $2, updated_at = NOW()
			WHERE message_id = $3 AND user_id = $4 AND status = 'sent'
		`
		args = []interface{}{status, timestamp, messageID, userID}
	} else if status == "read" {
		query = `
			UPDATE messages.delivery_status
			SET status = $1, read_at = $2, updated_at = NOW()
			WHERE message_id = $3 AND user_id = $4 AND status IN ('sent', 'delivered')
		`
		args = []interface{}{status, timestamp, messageID, userID}
	} else {
		query = `
			UPDATE messages.delivery_status
			SET status = $1, updated_at = NOW()
			WHERE message_id = $2 AND user_id = $3
		`
		args = []interface{}{status, messageID, userID}
	}

	_, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		r.logger.Error("failed to update delivery status",
			logger.String("message_id", messageID),
			logger.String("user_id", userID),
			logger.String("status", status),
			logger.Error(err),
		)
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to update delivery status")
	}

	r.logger.Debug("delivery status updated successfully",
		logger.String("message_id", messageID),
		logger.String("user_id", userID),
		logger.String("status", status),
	)

	return nil
}

// UpdateDeliveryStatusBatch updates delivery status for multiple messages
func (r *deliveryRepository) UpdateDeliveryStatusBatch(ctx context.Context, messageIDs []string, userID string, status string, timestamp *time.Time) pkgErrors.AppError {
	if len(messageIDs) == 0 {
		return nil
	}

	var query string
	var args []interface{}

	if status == "delivered" {
		query = `
			UPDATE messages.delivery_status
			SET status = $1, delivered_at = $2, updated_at = NOW()
			WHERE message_id = ANY($3) AND user_id = $4 AND status = 'sent'
		`
		args = []interface{}{status, timestamp, pq.Array(messageIDs), userID}
	} else if status == "read" {
		query = `
			UPDATE messages.delivery_status
			SET status = $1, read_at = $2, updated_at = NOW()
			WHERE message_id = ANY($3) AND user_id = $4 AND status IN ('sent', 'delivered')
		`
		args = []interface{}{status, timestamp, pq.Array(messageIDs), userID}
	} else {
		query = `
			UPDATE messages.delivery_status
			SET status = $1, updated_at = NOW()
			WHERE message_id = ANY($2) AND user_id = $3
		`
		args = []interface{}{status, pq.Array(messageIDs), userID}
	}

	_, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		r.logger.Error("failed to update delivery status batch",
			logger.Int("message_count", len(messageIDs)),
			logger.String("user_id", userID),
			logger.String("status", status),
			logger.Error(err),
		)
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to update delivery status batch")
	}

	r.logger.Debug("delivery status batch updated successfully",
		logger.Int("message_count", len(messageIDs)),
		logger.String("user_id", userID),
		logger.String("status", status),
	)

	return nil
}

// IncrementMessageDeliveryCount increments the delivery_count on the message
func (r *deliveryRepository) IncrementMessageDeliveryCount(ctx context.Context, messageID string) pkgErrors.AppError {
	query := `
		UPDATE messages.messages
		SET delivery_count = delivery_count + 1,
			delivered_at = COALESCE(delivered_at, NOW()),
			status = CASE WHEN status = 'sent' THEN 'delivered' ELSE status END,
			updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, messageID)
	if err != nil {
		r.logger.Error("failed to increment delivery count",
			logger.String("message_id", messageID),
			logger.Error(err),
		)
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to increment delivery count")
	}

	r.logger.Debug("delivery count incremented",
		logger.String("message_id", messageID),
	)

	return nil
}

// IncrementMessageReadCount increments the read_count on the message
func (r *deliveryRepository) IncrementMessageReadCount(ctx context.Context, messageID string) pkgErrors.AppError {
	query := `
		UPDATE messages.messages
		SET read_count = read_count + 1,
			status = 'read',
			updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, messageID)
	if err != nil {
		r.logger.Error("failed to increment read count",
			logger.String("message_id", messageID),
			logger.Error(err),
		)
		return pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to increment read count")
	}

	r.logger.Debug("read count incremented",
		logger.String("message_id", messageID),
	)

	return nil
}

// DeliveryStatusRow represents a raw delivery status record
type DeliveryStatusRow struct {
	ID          string
	MessageID   string
	UserID      string
	Status      string
	DeliveredAt *time.Time
	ReadAt      *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// GetDeliveryStatuses retrieves all delivery statuses for a message
func (r *deliveryRepository) GetDeliveryStatuses(ctx context.Context, messageID string) ([]DeliveryStatusRow, pkgErrors.AppError) {
	query := `
		SELECT id, message_id, user_id, status, delivered_at, read_at, created_at, updated_at
		FROM messages.delivery_status
		WHERE message_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.db.Query(ctx, query, messageID)
	if err != nil {
		r.logger.Error("failed to query delivery statuses",
			logger.String("message_id", messageID),
			logger.Error(err),
		)
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to query delivery statuses")
	}
	defer rows.Close()

	var statuses []DeliveryStatusRow
	for rows.Next() {
		var s DeliveryStatusRow
		if err := rows.Scan(&s.ID, &s.MessageID, &s.UserID, &s.Status, &s.DeliveredAt, &s.ReadAt, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to scan delivery status")
		}
		statuses = append(statuses, s)
	}
	if err := rows.Err(); err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "error iterating delivery statuses")
	}
	return statuses, nil
}

// MarkConversationAsRead marks all messages up to a given message as read for a user
func (r *deliveryRepository) MarkConversationAsRead(ctx context.Context, userID string, conversationID string, upToMessageID string) pkgErrors.AppError {
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

	// Update delivery statuses to read for all messages up to the given message
	updateQuery := `
		UPDATE messages.delivery_status ds
		SET status = 'read', read_at = NOW(), updated_at = NOW()
		FROM messages.messages m
		WHERE ds.message_id = m.id
		  AND m.conversation_id = $1
		  AND ds.user_id = $2
		  AND ds.status IN ('sent', 'delivered')
		  AND m.created_at <= (SELECT created_at FROM messages.messages WHERE id = $3)
	`
	_, txErr = tx.Exec(ctx, updateQuery, conversationID, userID, upToMessageID)
	if txErr != nil {
		r.logger.Error("failed to mark conversation as read",
			logger.String("conversation_id", conversationID),
			logger.String("user_id", userID),
			logger.Error(txErr),
		)
		return pkgErrors.FromError(txErr, pkgErrors.CodeDatabaseError, "failed to mark messages as read")
	}

	// Reset unread count on participant
	participantQuery := `
		UPDATE messages.conversation_participants
		SET unread_count = 0, last_read_at = NOW()
		WHERE conversation_id = $1 AND user_id = $2
	`
	_, txErr = tx.Exec(ctx, participantQuery, conversationID, userID)
	if txErr != nil {
		r.logger.Error("failed to reset unread count",
			logger.String("conversation_id", conversationID),
			logger.String("user_id", userID),
			logger.Error(txErr),
		)
		return pkgErrors.FromError(txErr, pkgErrors.CodeDatabaseError, "failed to reset unread count")
	}

	if txErr = tx.Commit(); txErr != nil {
		return pkgErrors.FromError(txErr, pkgErrors.CodeInternal, "failed to commit mark as read transaction")
	}

	r.logger.Debug("conversation marked as read",
		logger.String("conversation_id", conversationID),
		logger.String("user_id", userID),
		logger.String("up_to_message_id", upToMessageID),
	)
	return nil
}
