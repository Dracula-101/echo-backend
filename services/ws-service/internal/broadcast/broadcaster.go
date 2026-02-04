package broadcast

import (
	"context"
	"encoding/json"

	"shared/pkg/logger"
	"ws-service/internal/protocol"

	"github.com/google/uuid"
)

type LocalBroadcaster struct {
	connections   ConnectionProvider
	subscriptions SubscriptionProvider
	metrics       Metrics
	log           logger.Logger
}

func NewLocalBroadcaster(
	connections ConnectionProvider,
	subscriptions SubscriptionProvider,
	metrics Metrics,
	log logger.Logger,
) *LocalBroadcaster {
	return &LocalBroadcaster{
		connections:   connections,
		subscriptions: subscriptions,
		metrics:       metrics,
		log:           log,
	}
}

func (b *LocalBroadcaster) ToUser(ctx context.Context, userID uuid.UUID, msg *protocol.ServerMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	conns := b.connections.GetUserConnections(userID)
	for _, conn := range conns {
		conn.Send(data)
	}

	if b.metrics != nil {
		b.metrics.OnBroadcast()
		b.metrics.OnMessageSent(int64(len(data)))
	}

	return nil
}

func (b *LocalBroadcaster) ToConversation(ctx context.Context, convID uuid.UUID, msg *protocol.ServerMessage, exclude ...uuid.UUID) error {
	topicKey := "conversation:" + convID.String()
	subscribers := b.subscriptions.GetSubscribers(topicKey)

	b.log.Debug("broadcasting to conversation",
		logger.String("topic_key", topicKey),
		logger.String("message_type", msg.Type),
		logger.Int("subscriber_count", len(subscribers)),
	)

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	excludeMap := make(map[uuid.UUID]bool, len(exclude))
	for _, id := range exclude {
		excludeMap[id] = true
	}

	sentCount := 0
	for _, connID := range subscribers {
		conn, ok := b.connections.GetConnection(connID)
		if !ok {
			continue
		}

		if excludeMap[conn.UserID()] {
			continue
		}

		conn.Send(data)
		sentCount++

		b.log.Debug("message sent to subscriber",
			logger.String("user_id", conn.UserID().String()),
			logger.String("conn_id", connID),
		)
	}

	b.log.Debug("broadcast completed",
		logger.String("topic_key", topicKey),
		logger.Int("sent_count", sentCount),
	)

	if b.metrics != nil {
		b.metrics.OnBroadcast()
	}

	return nil
}

func (b *LocalBroadcaster) ToTopic(ctx context.Context, topic string, msg *protocol.ServerMessage) error {
	subscribers := b.subscriptions.GetSubscribers(topic)

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	for _, connID := range subscribers {
		conn, ok := b.connections.GetConnection(connID)
		if !ok {
			continue
		}
		conn.Send(data)
	}

	if b.metrics != nil {
		b.metrics.OnBroadcast()
	}

	return nil
}

func (b *LocalBroadcaster) ToConnection(ctx context.Context, connID string, msg *protocol.ServerMessage) error {
	conn, ok := b.connections.GetConnection(connID)
	if !ok {
		return nil
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return conn.Send(data)
}
