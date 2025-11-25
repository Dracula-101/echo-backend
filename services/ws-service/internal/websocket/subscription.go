package websocket

import (
	"sync"

	"shared/pkg/logger"
)

// SubscriptionManager manages topic subscriptions (application-specific)
type SubscriptionManager struct {
	// topic -> connection IDs
	subscriptions map[string]map[string]bool

	// connection ID -> topics
	connSubscriptions map[string][]string

	mu  sync.RWMutex
	log logger.Logger
}

// NewSubscriptionManager creates a new subscription manager
func NewSubscriptionManager(log logger.Logger) *SubscriptionManager {
	return &SubscriptionManager{
		subscriptions:     make(map[string]map[string]bool),
		connSubscriptions: make(map[string][]string),
		log:               log,
	}
}

// Subscribe subscribes a connection to a topic
func (sm *SubscriptionManager) Subscribe(connID, topic string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.subscriptions[topic] == nil {
		sm.subscriptions[topic] = make(map[string]bool)
	}

	sm.subscriptions[topic][connID] = true
	sm.connSubscriptions[connID] = append(sm.connSubscriptions[connID], topic)

	sm.log.Debug("Subscription added",
		logger.String("conn_id", connID),
		logger.String("topic", topic),
	)
}

// Unsubscribe unsubscribes a connection from a topic
func (sm *SubscriptionManager) Unsubscribe(connID, topic string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if subs, ok := sm.subscriptions[topic]; ok {
		delete(subs, connID)
		if len(subs) == 0 {
			delete(sm.subscriptions, topic)
		}
	}

	// Remove from connection subscriptions
	topics := sm.connSubscriptions[connID]
	newTopics := make([]string, 0)
	for _, t := range topics {
		if t != topic {
			newTopics = append(newTopics, t)
		}
	}
	sm.connSubscriptions[connID] = newTopics

	sm.log.Debug("Subscription removed",
		logger.String("conn_id", connID),
		logger.String("topic", topic),
	)
}

// UnsubscribeAll unsubscribes a connection from all topics
func (sm *SubscriptionManager) UnsubscribeAll(connID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	topics, ok := sm.connSubscriptions[connID]
	if !ok {
		return
	}

	for _, topic := range topics {
		if subs, ok := sm.subscriptions[topic]; ok {
			delete(subs, connID)
			if len(subs) == 0 {
				delete(sm.subscriptions, topic)
			}
		}
	}

	delete(sm.connSubscriptions, connID)

	sm.log.Debug("All subscriptions removed",
		logger.String("conn_id", connID),
		logger.Int("topic_count", len(topics)),
	)
}

// GetSubscribers returns all connection IDs subscribed to a topic
func (sm *SubscriptionManager) GetSubscribers(topic string) []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	subs, ok := sm.subscriptions[topic]
	if !ok {
		return nil
	}

	connIDs := make([]string, 0, len(subs))
	for connID := range subs {
		connIDs = append(connIDs, connID)
	}

	return connIDs
}
