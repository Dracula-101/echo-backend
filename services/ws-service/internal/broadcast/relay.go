package broadcast

import (
	"context"
	"encoding/json"

	"shared/pkg/logger"
	"ws-service/internal/protocol"
	"ws-service/internal/redis"

	"github.com/google/uuid"
)

type DistributedBroadcaster struct {
	local      *LocalBroadcaster
	pubsub     *redis.PubSub
	instanceID string
	log        logger.Logger
}

func NewDistributedBroadcaster(
	local *LocalBroadcaster,
	pubsub *redis.PubSub,
	instanceID string,
	log logger.Logger,
) *DistributedBroadcaster {
	d := &DistributedBroadcaster{
		local:      local,
		pubsub:     pubsub,
		instanceID: instanceID,
		log:        log,
	}

	pubsub.OnMessage(redis.ChannelBroadcast, d.handleRelayMessage)
	return d
}

func (d *DistributedBroadcaster) ToUser(ctx context.Context, userID uuid.UUID, msg *protocol.ServerMessage) error {
	if err := d.local.ToUser(ctx, userID, msg); err != nil {
		return err
	}

	payload, _ := json.Marshal(msg)
	return d.pubsub.PublishBroadcast(ctx, &redis.BroadcastMessage{
		Type:       "user",
		TargetType: "user",
		TargetID:   userID.String(),
		Payload:    payload,
	})
}

func (d *DistributedBroadcaster) ToConversation(ctx context.Context, convID uuid.UUID, msg *protocol.ServerMessage, exclude ...uuid.UUID) error {
	if err := d.local.ToConversation(ctx, convID, msg, exclude...); err != nil {
		return err
	}

	payload, _ := json.Marshal(msg)
	excludeIDs := make([]string, len(exclude))
	for i, id := range exclude {
		excludeIDs[i] = id.String()
	}

	return d.pubsub.PublishBroadcast(ctx, &redis.BroadcastMessage{
		Type:       "conversation",
		TargetType: "conversation",
		TargetID:   convID.String(),
		ExcludeIDs: excludeIDs,
		Payload:    payload,
	})
}

func (d *DistributedBroadcaster) ToTopic(ctx context.Context, topic string, msg *protocol.ServerMessage) error {
	if err := d.local.ToTopic(ctx, topic, msg); err != nil {
		return err
	}

	payload, _ := json.Marshal(msg)
	return d.pubsub.PublishBroadcast(ctx, &redis.BroadcastMessage{
		Type:       "topic",
		TargetType: "topic",
		TargetID:   topic,
		Payload:    payload,
	})
}

func (d *DistributedBroadcaster) ToConnection(ctx context.Context, connID string, msg *protocol.ServerMessage) error {
	return d.local.ToConnection(ctx, connID, msg)
}

func (d *DistributedBroadcaster) handleRelayMessage(msg *redis.BroadcastMessage) {
	ctx := context.Background()

	var serverMsg protocol.ServerMessage
	if err := json.Unmarshal(msg.Payload, &serverMsg); err != nil {
		d.log.Error("failed to unmarshal relay message",
			logger.Error(err),
		)
		return
	}

	switch msg.TargetType {
	case "user":
		userID, err := uuid.Parse(msg.TargetID)
		if err != nil {
			return
		}
		d.local.ToUser(ctx, userID, &serverMsg)

	case "conversation":
		convID, err := uuid.Parse(msg.TargetID)
		if err != nil {
			return
		}
		exclude := make([]uuid.UUID, 0, len(msg.ExcludeIDs))
		for _, idStr := range msg.ExcludeIDs {
			if id, err := uuid.Parse(idStr); err == nil {
				exclude = append(exclude, id)
			}
		}
		d.local.ToConversation(ctx, convID, &serverMsg, exclude...)

	case "topic":
		d.local.ToTopic(ctx, msg.TargetID, &serverMsg)
	}
}
