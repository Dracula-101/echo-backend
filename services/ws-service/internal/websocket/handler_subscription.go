package websocket

import (
	"context"
	"encoding/json"
	"strings"

	"shared/pkg/logger"
	"shared/server/websocket/router"
	"ws-service/internal/protocol"
)

func (m *Manager) handleSubscribe(ctx context.Context, msg *router.Message) error {
	conn, ok := m.getConnection(msg)
	if !ok {
		return nil
	}

	var payload protocol.SubscribePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	userID, ok := m.getUserID(conn)
	if !ok {
		return m.sendError(conn, m.getRequestID(msg), protocol.ErrCodeUnauthorized, "User not authenticated")
	}

	subscribedTopics := make([]protocol.Topic, 0, len(payload.Topics))

	for _, topic := range payload.Topics {
		resourceID := protocol.GetResourceID(topic, payload.Filters)
		topicKey := string(topic) + ":" + resourceID

		if !m.isSubscriptionAllowed(userID, topic, resourceID) {
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
