package repo

import (
	"context"

	"shared/pkg/database"
	"shared/pkg/logger"

	"github.com/google/uuid"
)

// DeliveryStatusRepository writes delivery_status rows from ws-service. The
// canonical inserter is the Postgres trigger on messages.messages, but doing
// this here too gives a safety net if the trigger is ever disabled or misses
// a row, and lets ws-service own the write path for messages that didn't
// originate from message-service (e.g. system events).
type DeliveryStatusRepository interface {
	EnsureDeliveryStatus(ctx context.Context, messageID uuid.UUID, userIDs []uuid.UUID) error
	GetUndelivered(ctx context.Context, userID uuid.UUID, limit int) ([]UndeliveredMessage, error)
}

type UndeliveredMessage struct {
	MessageID      uuid.UUID
	ConversationID uuid.UUID
	SenderUserID   uuid.UUID
	Content        []byte
	CreatedAt      string
}

type deliveryStatusRepository struct {
	db  database.Database
	log logger.Logger
}

func NewDeliveryStatusRepository(db database.Database, log logger.Logger) DeliveryStatusRepository {
	return &deliveryStatusRepository{db: db, log: log}
}

func (r *deliveryStatusRepository) EnsureDeliveryStatus(ctx context.Context, messageID uuid.UUID, userIDs []uuid.UUID) error {
	if len(userIDs) == 0 {
		return nil
	}

	const query = `
		INSERT INTO messages.delivery_status (message_id, user_id, status)
		SELECT $1, unnest($2::uuid[]), 'sent'
		ON CONFLICT (message_id, user_id) DO NOTHING
	`

	ids := make([]string, len(userIDs))
	for i, id := range userIDs {
		ids[i] = id.String()
	}

	if _, err := r.db.Exec(ctx, query, messageID, ids); err != nil {
		r.log.Error("failed to ensure delivery_status",
			logger.String("message_id", messageID.String()),
			logger.Int("user_count", len(userIDs)),
			logger.Error(err),
		)
		return err
	}
	return nil
}

// GetUndelivered returns up to `limit` messages still marked 'sent' for this
// user from the past 24h. Used by the reconnect catch-up path.
func (r *deliveryStatusRepository) GetUndelivered(ctx context.Context, userID uuid.UUID, limit int) ([]UndeliveredMessage, error) {
	const query = `
		SELECT m.id, m.conversation_id, m.sender_user_id, m.content, m.created_at
		FROM messages.delivery_status ds
		JOIN messages.messages m ON m.id = ds.message_id
		WHERE ds.user_id = $1
		  AND ds.status = 'sent'
		  AND m.created_at > NOW() - INTERVAL '24 hours'
		ORDER BY m.created_at ASC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, userID, limit)
	if err != nil {
		r.log.Error("failed to load undelivered messages",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return nil, err
	}
	defer rows.Close()

	out := make([]UndeliveredMessage, 0, 16)
	for rows.Next() {
		var m UndeliveredMessage
		if err := rows.Scan(&m.MessageID, &m.ConversationID, &m.SenderUserID, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
