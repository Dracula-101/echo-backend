package handler

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"shared/pkg/logger"
	"ws-service/internal/protocol"
)

type SubscriptionHandler struct {
	deps *Dependencies
	log  logger.Logger
}

func NewSubscriptionHandler(deps *Dependencies, log logger.Logger) *SubscriptionHandler {
	return &SubscriptionHandler{deps: deps, log: log}
}

func (h *SubscriptionHandler) Handle(ctx context.Context, hctx *Context) error {
	if !hctx.Conn.IsAuthenticated() {
		return h.sendError(hctx, protocol.ErrCodeUnauthorized, "User not authenticated")
	}

	var payload protocol.SubscribePayload
	if err := json.Unmarshal(hctx.Payload, &payload); err != nil {
		return err
	}

	userID := hctx.Conn.UserID()
	connID := hctx.Conn.ID()

	h.log.Info("processing subscription request",
		logger.String("user_id", userID.String()),
		logger.String("conn_id", connID),
		logger.Int("topic_count", len(payload.Topics)),
	)

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	subscribedTopics := make([]protocol.Topic, 0, len(payload.Topics))

	for _, topic := range payload.Topics {
		if topic == protocol.TopicConversation && payload.Filters.ConversationIDs != nil && len(*payload.Filters.ConversationIDs) > 0 {
			successCount := 0
			for _, convID := range *payload.Filters.ConversationIDs {
				topicKey := string(topic) + ":" + convID

				allowed, err := h.deps.Authz.CanSubscribeToTopic(dbCtx, userID, topic, convID)
				if err != nil {
					h.log.Error("failed to validate conversation subscription",
						logger.String("user_id", userID.String()),
						logger.String("topic", string(topic)),
						logger.String("conversation_id", convID),
						logger.Error(err),
					)
					continue
				}

				if !allowed {
					h.log.Warn("conversation subscription denied",
						logger.String("user_id", userID.String()),
						logger.String("topic", string(topic)),
						logger.String("conversation_id", convID),
					)
					continue
				}

				h.deps.Subscriptions.Subscribe(connID, topicKey)
				h.log.Info("conversation subscription successful",
					logger.String("user_id", userID.String()),
					logger.String("topic_key", topicKey),
				)
				successCount++
			}

			if successCount > 0 {
				subscribedTopics = append(subscribedTopics, topic)
			}
			continue
		}

		resourceID := protocol.GetResourceID(topic, payload.Filters)
		topicKey := string(topic) + ":" + resourceID

		allowed, err := h.deps.Authz.CanSubscribeToTopic(dbCtx, userID, topic, resourceID)
		if err != nil {
			h.log.Error("failed to validate subscription",
				logger.String("user_id", userID.String()),
				logger.String("topic", string(topic)),
				logger.String("resource_id", resourceID),
				logger.Error(err),
			)
			continue
		}

		if !allowed {
			h.log.Warn("subscription denied",
				logger.String("user_id", userID.String()),
				logger.String("topic", string(topic)),
				logger.String("resource_id", resourceID),
			)
			continue
		}

		h.deps.Subscriptions.Subscribe(connID, topicKey)
		subscribedTopics = append(subscribedTopics, topic)
	}

	subscriptionDetails := buildSubscriptionDetails(h.deps.Subscriptions.GetSubscriptions(connID))

	ack := protocol.NewServerMessage(protocol.MsgTypeSubscribed, hctx.RequestID).WithPayload(
		protocol.SubscribedPayload{
			Topics:            subscribedTopics,
			SubscriptionCount: h.deps.Subscriptions.CountSubscriptions(connID),
			Subscriptions:     subscriptionDetails,
		},
	)

	data, _ := json.Marshal(ack)
	return hctx.Conn.Send(data)
}

func (h *SubscriptionHandler) sendError(hctx *Context, code protocol.ErrorCode, message string) error {
	errMsg := protocol.NewErrorMessage(hctx.RequestID, code, message)
	data, _ := json.Marshal(errMsg)
	return hctx.Conn.Send(data)
}

type UnsubscribeHandler struct {
	deps *Dependencies
	log  logger.Logger
}

func NewUnsubscribeHandler(deps *Dependencies, log logger.Logger) *UnsubscribeHandler {
	return &UnsubscribeHandler{deps: deps, log: log}
}

func (h *UnsubscribeHandler) Handle(ctx context.Context, hctx *Context) error {
	var payload protocol.UnsubscribePayload
	if err := json.Unmarshal(hctx.Payload, &payload); err != nil {
		return err
	}

	connID := hctx.Conn.ID()

	for _, topic := range payload.Topics {
		if topic == protocol.TopicConversation {
			for _, convID := range *payload.Filters.ConversationIDs {
				topicKey := string(topic) + ":" + convID
				h.deps.Subscriptions.Unsubscribe(connID, topicKey)
			}
			continue
		}

		resourceID := protocol.GetResourceID(topic, payload.Filters)
		topicKey := string(topic) + ":" + resourceID
		h.deps.Subscriptions.Unsubscribe(connID, topicKey)
	}

	ack := protocol.NewServerMessage(protocol.MsgTypeUnsubscribed, hctx.RequestID).WithPayload(
		protocol.UnsubscribedPayload{
			Topics:             payload.Topics,
			RemainingCount:     h.deps.Subscriptions.CountSubscriptions(connID),
			UnsubscribedTopics: payload.Topics,
		},
	)

	data, _ := json.Marshal(ack)
	return hctx.Conn.Send(data)
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
