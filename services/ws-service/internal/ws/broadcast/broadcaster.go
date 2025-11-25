package broadcast

import (
	"sync"
	"shared/server/websocket"
	"ws-service/internal/ws/protocol"
	"ws-service/internal/ws/subscription"

	"shared/pkg/logger"

	"github.com/google/uuid"
)

// Broadcaster handles broadcasting messages to clients
type Broadcaster struct {
	hub        *websocket.Hub
	subManager *subscription.Manager
	log        logger.Logger
	mu         sync.RWMutex
}

// NewBroadcaster creates a new broadcaster
func NewBroadcaster(hub *websocket.Hub, subManager *subscription.Manager, log logger.Logger) *Broadcaster {
	return &Broadcaster{
		hub:        hub,
		subManager: subManager,
		log:        log,
	}
}

// BroadcastToUser broadcasts a message to all devices of a specific user
func (b *Broadcaster) BroadcastToUser(userID uuid.UUID, message protocol.ServerMessage) int {
	b.log.Debug("Broadcasting to user",
		logger.String("user_id", userID.String()),
		logger.String("message_type", string(message.Type)),
	)

	// Check if user is online
	if !b.hub.IsUserOnline(userID) {
		b.log.Debug("User is offline",
			logger.String("user_id", userID.String()),
		)
		return 0
	}

	// Get user device count
	deviceCount := b.hub.GetUserDeviceCount(userID)
	if deviceCount == 0 {
		return 0
	}

	// Broadcast through hub
	event := &websocket.OutgoingMessage{
		Type:      string(message.Type),
		Payload:   message.Payload,
		Timestamp: message.Timestamp,
	}

	// Send to user's devices
	b.hub.SendToUser(userID, event)

	b.log.Debug("Message broadcasted to user",
		logger.String("user_id", userID.String()),
		logger.Int("devices", deviceCount),
	)

	return deviceCount
}

// BroadcastToUsers broadcasts a message to multiple users
func (b *Broadcaster) BroadcastToUsers(userIDs []uuid.UUID, message protocol.ServerMessage, excludeUserID ...uuid.UUID) int {
	b.log.Debug("Broadcasting to multiple users",
		logger.Int("user_count", len(userIDs)),
		logger.String("message_type", string(message.Type)),
	)

	sentCount := 0
	excludeMap := make(map[uuid.UUID]bool)
	for _, uid := range excludeUserID {
		excludeMap[uid] = true
	}

	for _, userID := range userIDs {
		// Skip excluded users
		if excludeMap[userID] {
			continue
		}

		if count := b.BroadcastToUser(userID, message); count > 0 {
			sentCount++
		}
	}

	b.log.Debug("Message broadcasted to users",
		logger.Int("sent_to", sentCount),
		logger.Int("total", len(userIDs)),
	)

	return sentCount
}

// BroadcastToTopic broadcasts a message to all subscribers of a topic
func (b *Broadcaster) BroadcastToTopic(topic protocol.SubscriptionTopic, resourceID string, message protocol.ServerMessage, excludeUserID ...uuid.UUID) int {
	b.log.Debug("Broadcasting to topic",
		logger.String("topic", string(topic)),
		logger.String("resource_id", resourceID),
		logger.String("message_type", string(message.Type)),
	)

	// Get subscribers
	subscribers := b.subManager.GetSubscribers(topic, resourceID)
	if len(subscribers) == 0 {
		b.log.Debug("No subscribers for topic",
			logger.String("topic", string(topic)),
			logger.String("resource_id", resourceID),
		)
		return 0
	}

	// Create exclude map
	excludeMap := make(map[uuid.UUID]bool)
	for _, uid := range excludeUserID {
		excludeMap[uid] = true
	}

	// Send to all subscribers
	sentCount := 0
	for _, client := range subscribers {
		// Skip excluded users
		if excludeMap[client.UserID] {
			continue
		}

		if err := client.SendMessage(message); err != nil {
			b.log.Error("Failed to send message to subscriber",
				logger.String("client_id", client.ID),
				logger.Error(err),
			)
		} else {
			sentCount++
		}
	}

	b.log.Debug("Message broadcasted to topic subscribers",
		logger.String("topic", string(topic)),
		logger.Int("sent_count", sentCount),
		logger.Int("total_subscribers", len(subscribers)),
	)

	return sentCount
}

// BroadcastToAll broadcasts a message to all connected clients
func (b *Broadcaster) BroadcastToAll(message protocol.ServerMessage) int {
	b.log.Info("Broadcasting to all clients",
		logger.String("message_type", string(message.Type)),
	)

	onlineUsers := b.hub.GetOnlineUsers()
	return b.BroadcastToUsers(onlineUsers, message)
}

// GetOnlineUserCount returns the number of online users
func (b *Broadcaster) GetOnlineUserCount() int {
	return len(b.hub.GetOnlineUsers())
}

// IsUserOnline checks if a user is online
func (b *Broadcaster) IsUserOnline(userID uuid.UUID) bool {
	return b.hub.IsUserOnline(userID)
}
