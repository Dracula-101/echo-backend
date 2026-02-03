package websocket

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"shared/pkg/logger"
	"shared/server/websocket/router"
	"ws-service/internal/protocol"
)

func (m *Manager) handleSubscribe(_ context.Context, msg *router.Message) error {
	conn, ok := m.getConnection(msg)
	if !ok {
		return nil
	}

	var payload protocol.SubscribePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		m.log.Error("Failed to unmarshal subscribe payload",
			logger.Error(err),
		)
		return err
	}

	userID, ok := m.getUserID(conn)
	if !ok {
		return m.sendError(conn, m.getRequestID(msg), protocol.ErrCodeUnauthorized, "User not authenticated")
	}

	m.log.Info("Processing subscription request",
		logger.String("user_id", userID.String()),
		logger.String("conn_id", conn.ID()),
		logger.Int("topic_count", len(payload.Topics)),
		logger.Int("conversation_count", len(payload.ConversationIDs)),
	)

	// Use a detached context with timeout for database operations
	// The incoming context is tied to the WebSocket read and may be canceled
	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	subscribedTopics := make([]protocol.Topic, 0, len(payload.Topics))

	for _, topic := range payload.Topics {
		// Handle batch conversation subscriptions
		if topic == protocol.TopicConversation && len(payload.ConversationIDs) > 0 {
			successCount := 0
			for _, convID := range payload.ConversationIDs {
				resourceID := convID.String()
				topicKey := string(topic) + ":" + resourceID

				// Validate conversation access using authorization service
				allowed, err := m.authzService.CanSubscribeToTopic(dbCtx, userID, topic, resourceID)
				if err != nil {
					m.log.Error("Failed to validate conversation subscription",
						logger.String("user_id", userID.String()),
						logger.String("topic", string(topic)),
						logger.String("resource_id", resourceID),
						logger.Error(err),
					)
					continue
				}

				if !allowed {
					m.log.Warn("Conversation subscription denied",
						logger.String("user_id", userID.String()),
						logger.String("topic", string(topic)),
						logger.String("resource_id", resourceID),
					)
					continue
				}

				m.subscriptions.Subscribe(conn.ID(), topicKey)
				m.log.Info("Conversation subscription successful",
					logger.String("user_id", userID.String()),
					logger.String("topic_key", topicKey),
				)
				successCount++
			}

			// Only add topic to subscribed list if at least one subscription succeeded
			if successCount > 0 {
				subscribedTopics = append(subscribedTopics, topic)
			}
			continue
		}

		// Handle single topic subscription (backward compatible)
		resourceID := protocol.GetResourceID(topic, payload.Filters)
		topicKey := string(topic) + ":" + resourceID

		// Validate subscription using authorization service
		allowed, err := m.authzService.CanSubscribeToTopic(dbCtx, userID, topic, resourceID)
		if err != nil {
			m.log.Error("Failed to validate subscription",
				logger.String("user_id", userID.String()),
				logger.String("topic", string(topic)),
				logger.String("resource_id", resourceID),
				logger.Error(err),
			)
			continue
		}

		if !allowed {
			m.log.Warn("Subscription denied",
				logger.String("user_id", userID.String()),
				logger.String("topic", string(topic)),
				logger.String("resource_id", resourceID),
			)
			continue
		}

		m.subscriptions.Subscribe(conn.ID(), topicKey)
		subscribedTopics = append(subscribedTopics, topic)
	}

	subscriptionDetails := buildSubscriptionDetails(m.subscriptions.GetSubscriptions(conn.ID()))

	ack := protocol.NewServerMessage(protocol.MsgTypeSubscribed, m.getRequestID(msg)).WithPayload(
		protocol.SubscribedPayload{
			Topics:            subscribedTopics,
			SubscriptionCount: m.subscriptions.CountSubscriptions(conn.ID()),
			Subscriptions:     subscriptionDetails,
		},
	)

	data, _ := json.Marshal(ack)
	return conn.Send(data)
}

func (m *Manager) handleUnsubscribe(ctx context.Context, msg *router.Message) error {
	conn, ok := m.getConnection(msg)
	if !ok {
		return nil
	}

	var payload protocol.UnsubscribePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	for _, topic := range payload.Topics {
		// Handle batch conversation unsubscriptions
		if topic == protocol.TopicConversation && len(payload.ConversationIDs) > 0 {
			for _, convID := range payload.ConversationIDs {
				resourceID := convID.String()
				topicKey := string(topic) + ":" + resourceID
				m.subscriptions.Unsubscribe(conn.ID(), topicKey)
			}
			continue
		}

		// Handle single topic unsubscription (backward compatible)
		resourceID := protocol.GetResourceID(topic, payload.Filters)
		topicKey := string(topic) + ":" + resourceID
		m.subscriptions.Unsubscribe(conn.ID(), topicKey)
	}

	ack := protocol.NewServerMessage(protocol.MsgTypeUnsubscribed, m.getRequestID(msg)).WithPayload(
		protocol.UnsubscribedPayload{
			Topics:             payload.Topics,
			RemainingCount:     m.subscriptions.CountSubscriptions(conn.ID()),
			UnsubscribedTopics: payload.Topics,
		},
	)

	data, _ := json.Marshal(ack)
	return conn.Send(data)
}

func buildSubscriptionDetails(topicKeys []string) []protocol.SubscriptionDetail {
	if len(topicKeys) == 0 {
		return nil
	}

	result := make([]protocol.SubscriptionDetail, 0, len(topicKeys))
	for _, topicKey := range topicKeys {
		parts := strings.SplitN(topicKey, ":", 2)
		topic := protocol.Topic(parts[0])
		resourceID := "default"
		if len(parts) == 2 {
			resourceID = parts[1]
		}

		result = append(result, protocol.SubscriptionDetail{
			Topic:      topic,
			ResourceID: resourceID,
			Active:     true,
		})
	}

	return result
}
