package repo

import (
	"context"
	"encoding/json"

	"shared/pkg/database"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"

	"github.com/google/uuid"
)

// OutboxEvent is a single row enqueued for relay-based Kafka publishing.
//
// Payload must already be marshaled JSON; the caller owns the schema. Headers
// is for cross-cutting fields like correlation_id / trace_id that the relay
// will copy onto the outbound Kafka message.
type OutboxEvent struct {
	EventType   string
	Topic       string
	AggregateID *uuid.UUID
	Payload     json.RawMessage
	UserID      *uuid.UUID
	Headers     map[string]string
}

// OutboxRepository writes events to messages.outbox inside an existing
// business transaction. The relay process (a separate worker) publishes
// unprocessed rows to Kafka.
//
// Enqueue requires a Transaction rather than the bare Database — the whole
// point of the outbox pattern is that the event row commits atomically with
// the business write. Standalone enqueues would defeat that.
type OutboxRepository interface {
	Enqueue(ctx context.Context, tx database.Transaction, event *OutboxEvent) pkgErrors.AppError
}

type outboxRepository struct {
	logger logger.Logger
}

func NewOutboxRepository(logger logger.Logger) OutboxRepository {
	return &outboxRepository{logger: logger}
}

func (r *outboxRepository) Enqueue(ctx context.Context, tx database.Transaction, event *OutboxEvent) pkgErrors.AppError {
	if event == nil {
		return pkgErrors.New(pkgErrors.CodeInvalidArgument, "outbox event is nil")
	}
	if event.EventType == "" {
		return pkgErrors.New(pkgErrors.CodeInvalidArgument, "outbox event_type is required")
	}
	if event.Topic == "" {
		return pkgErrors.New(pkgErrors.CodeInvalidArgument, "outbox topic is required")
	}
	if len(event.Payload) == 0 {
		return pkgErrors.New(pkgErrors.CodeInvalidArgument, "outbox payload is required")
	}

	headersJSON, err := marshalHeaders(event.Headers)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeInternal, "failed to marshal outbox headers")
	}

	const query = `
		INSERT INTO messages.outbox (
			event_type, topic, aggregate_id, payload, user_id, headers
		) VALUES (
			$1, $2, $3, $4, $5, $6
		)
	`

	_, dbErr := tx.Exec(ctx, query,
		event.EventType,
		event.Topic,
		event.AggregateID,
		[]byte(event.Payload),
		event.UserID,
		headersJSON,
	)
	if dbErr != nil {
		r.logger.Error("failed to enqueue outbox event",
			logger.String("event_type", event.EventType),
			logger.String("topic", event.Topic),
			logger.Error(dbErr),
		)
		return pkgErrors.FromError(dbErr, pkgErrors.CodeDatabaseError, "failed to enqueue outbox event")
	}

	r.logger.Debug("outbox event enqueued",
		logger.String("event_type", event.EventType),
		logger.String("topic", event.Topic),
	)
	return nil
}

func marshalHeaders(h map[string]string) ([]byte, error) {
	if len(h) == 0 {
		return []byte("{}"), nil
	}
	return json.Marshal(h)
}
