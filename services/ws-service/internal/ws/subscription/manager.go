package subscription

import (
	"sync"
	"shared/server/websocket"
	"ws-service/internal/ws/protocol"

	"shared/pkg/logger"
)

// Subscription represents a client subscription to a topic
type Subscription struct {
	Client  *websocket.Client
	Topic   protocol.SubscriptionTopic
	Filters map[string]string
}

// Manager manages client subscriptions
type Manager struct {
	// Map of topic -> resource_id -> clients
	subscriptions map[protocol.SubscriptionTopic]map[string]map[*websocket.Client]bool

	// Map of client -> subscriptions
	clientSubscriptions map[*websocket.Client]map[protocol.SubscriptionTopic][]string

	mu  sync.RWMutex
	log logger.Logger
}

// NewManager creates a new subscription manager
func NewManager(log logger.Logger) *Manager {
	return &Manager{
		subscriptions:       make(map[protocol.SubscriptionTopic]map[string]map[*websocket.Client]bool),
		clientSubscriptions: make(map[*websocket.Client]map[protocol.SubscriptionTopic][]string),
		log:                 log,
	}
}

// Subscribe subscribes a client to a topic
func (m *Manager) Subscribe(client *websocket.Client, topic protocol.SubscriptionTopic, filters map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get resource ID from filters (e.g., conversation_id, user_id)
	resourceID := m.getResourceID(topic, filters)

	// Initialize topic map if doesn't exist
	if m.subscriptions[topic] == nil {
		m.subscriptions[topic] = make(map[string]map[*websocket.Client]bool)
	}

	// Initialize resource map if doesn't exist
	if m.subscriptions[topic][resourceID] == nil {
		m.subscriptions[topic][resourceID] = make(map[*websocket.Client]bool)
	}

	// Add client to subscribers
	m.subscriptions[topic][resourceID][client] = true

	// Track client subscriptions
	if m.clientSubscriptions[client] == nil {
		m.clientSubscriptions[client] = make(map[protocol.SubscriptionTopic][]string)
	}
	m.clientSubscriptions[client][topic] = append(m.clientSubscriptions[client][topic], resourceID)

	m.log.Info("Client subscribed",
		logger.String("client_id", client.ID),
		logger.String("topic", string(topic)),
		logger.String("resource_id", resourceID),
	)

	return nil
}

// Unsubscribe unsubscribes a client from a topic
func (m *Manager) Unsubscribe(client *websocket.Client, topic protocol.SubscriptionTopic) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get client's subscriptions for this topic
	resourceIDs, ok := m.clientSubscriptions[client][topic]
	if !ok {
		return
	}

	// Remove from all resources
	for _, resourceID := range resourceIDs {
		if m.subscriptions[topic] != nil && m.subscriptions[topic][resourceID] != nil {
			delete(m.subscriptions[topic][resourceID], client)

			// Clean up empty maps
			if len(m.subscriptions[topic][resourceID]) == 0 {
				delete(m.subscriptions[topic], resourceID)
			}
		}
	}

	// Remove from client subscriptions
	delete(m.clientSubscriptions[client], topic)

	m.log.Info("Client unsubscribed",
		logger.String("client_id", client.ID),
		logger.String("topic", string(topic)),
	)
}

// UnsubscribeAll unsubscribes a client from all topics
func (m *Manager) UnsubscribeAll(client *websocket.Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get all client subscriptions
	topics, ok := m.clientSubscriptions[client]
	if !ok {
		return
	}

	// Remove from all subscriptions
	for topic, resourceIDs := range topics {
		for _, resourceID := range resourceIDs {
			if m.subscriptions[topic] != nil && m.subscriptions[topic][resourceID] != nil {
				delete(m.subscriptions[topic][resourceID], client)

				// Clean up empty maps
				if len(m.subscriptions[topic][resourceID]) == 0 {
					delete(m.subscriptions[topic], resourceID)
				}
			}
		}
	}

	// Remove client subscriptions
	delete(m.clientSubscriptions, client)

	m.log.Info("Client unsubscribed from all topics",
		logger.String("client_id", client.ID),
	)
}

// GetSubscribers returns all clients subscribed to a topic/resource
func (m *Manager) GetSubscribers(topic protocol.SubscriptionTopic, resourceID string) []*websocket.Client {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.subscriptions[topic] == nil || m.subscriptions[topic][resourceID] == nil {
		return nil
	}

	clients := make([]*websocket.Client, 0, len(m.subscriptions[topic][resourceID]))
	for client := range m.subscriptions[topic][resourceID] {
		clients = append(clients, client)
	}

	return clients
}

// GetClientSubscriptions returns all subscriptions for a client
func (m *Manager) GetClientSubscriptions(client *websocket.Client) map[protocol.SubscriptionTopic][]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	subs, ok := m.clientSubscriptions[client]
	if !ok {
		return nil
	}

	// Return a copy
	result := make(map[protocol.SubscriptionTopic][]string)
	for topic, resourceIDs := range subs {
		result[topic] = append([]string{}, resourceIDs...)
	}

	return result
}

// getResourceID extracts resource ID from filters
func (m *Manager) getResourceID(topic protocol.SubscriptionTopic, filters map[string]string) string {
	switch topic {
	case protocol.TopicUser:
		if userID, ok := filters["user_id"]; ok {
			return userID
		}
	case protocol.TopicConversation:
		if conversationID, ok := filters["conversation_id"]; ok {
			return conversationID
		}
	case protocol.TopicPresence:
		return "global" // Global presence updates
	case protocol.TopicTyping:
		if conversationID, ok := filters["conversation_id"]; ok {
			return conversationID
		}
	case protocol.TopicCalls:
		if callID, ok := filters["call_id"]; ok {
			return callID
		}
	case protocol.TopicNotifications:
		if userID, ok := filters["user_id"]; ok {
			return userID
		}
	}

	return "default"
}

// GetStats returns subscription statistics
func (m *Manager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_clients"] = len(m.clientSubscriptions)

	topicStats := make(map[string]int)
	for topic, resources := range m.subscriptions {
		count := 0
		for _, clients := range resources {
			count += len(clients)
		}
		topicStats[string(topic)] = count
	}
	stats["topics"] = topicStats

	return stats
}

// IsSubscribed checks if a client is subscribed to a topic/resource
func (m *Manager) IsSubscribed(client *websocket.Client, topic protocol.SubscriptionTopic, resourceID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.subscriptions[topic] == nil || m.subscriptions[topic][resourceID] == nil {
		return false
	}

	_, subscribed := m.subscriptions[topic][resourceID][client]
	return subscribed
}
