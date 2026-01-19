package tracker

import (
	"sync"

	"shared/pkg/logger"
)

type SubscriptionManager struct {
	subscriptions     map[string]map[string]bool
	connSubscriptions map[string][]string
	maxPerConnection  int
	mu                sync.RWMutex
	log               logger.Logger
}

type SubscriptionManagerConfig struct {
	MaxPerConnection int
}

func NewSubscriptionManager(log logger.Logger, cfg *SubscriptionManagerConfig) *SubscriptionManager {
	maxPerConn := 100
	if cfg != nil && cfg.MaxPerConnection > 0 {
		maxPerConn = cfg.MaxPerConnection
	}

	return &SubscriptionManager{
		subscriptions:     make(map[string]map[string]bool),
		connSubscriptions: make(map[string][]string),
		maxPerConnection:  maxPerConn,
		log:               log,
	}
}

func (sm *SubscriptionManager) Subscribe(connID, topic string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if len(sm.connSubscriptions[connID]) >= sm.maxPerConnection {
		sm.log.Warn("Subscription limit reached",
			logger.String("conn_id", connID),
			logger.Int("limit", sm.maxPerConnection),
		)
		return false
	}

	if sm.subscriptions[topic] == nil {
		sm.subscriptions[topic] = make(map[string]bool)
	}

	if sm.subscriptions[topic][connID] {
		return true
	}

	sm.subscriptions[topic][connID] = true
	sm.connSubscriptions[connID] = append(sm.connSubscriptions[connID], topic)

	sm.log.Debug("Subscription added",
		logger.String("conn_id", connID),
		logger.String("topic", topic),
	)
	return true
}

func (sm *SubscriptionManager) Unsubscribe(connID, topic string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if subs, ok := sm.subscriptions[topic]; ok {
		delete(subs, connID)
		if len(subs) == 0 {
			delete(sm.subscriptions, topic)
		}
	}

	topics := sm.connSubscriptions[connID]
	newTopics := make([]string, 0, len(topics))
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

func (sm *SubscriptionManager) UnsubscribeAll(connID string) []string {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	topics, ok := sm.connSubscriptions[connID]
	if !ok {
		return nil
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

	return topics
}

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

func (sm *SubscriptionManager) GetSubscriptions(connID string) []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	topics, ok := sm.connSubscriptions[connID]
	if !ok {
		return nil
	}

	result := make([]string, len(topics))
	copy(result, topics)
	return result
}

func (sm *SubscriptionManager) CountSubscriptions(connID string) int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.connSubscriptions[connID])
}

func (sm *SubscriptionManager) IsSubscribed(connID, topic string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if subs, ok := sm.subscriptions[topic]; ok {
		return subs[connID]
	}
	return false
}

func (sm *SubscriptionManager) GetTopicCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.subscriptions)
}

func (sm *SubscriptionManager) GetTotalSubscriptions() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	total := 0
	for _, subs := range sm.subscriptions {
		total += len(subs)
	}
	return total
}

func (sm *SubscriptionManager) GetStats() SubscriptionStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	topicCounts := make(map[string]int)
	for topic, subs := range sm.subscriptions {
		topicCounts[topic] = len(subs)
	}

	totalSubscriptions := 0
	for _, subs := range sm.subscriptions {
		totalSubscriptions += len(subs)
	}

	return SubscriptionStats{
		TotalTopics:        len(sm.subscriptions),
		TotalSubscriptions: totalSubscriptions,
		TotalConnections:   len(sm.connSubscriptions),
		TopicCounts:        topicCounts,
	}
}

type SubscriptionStats struct {
	TotalTopics        int            `json:"total_topics"`
	TotalSubscriptions int            `json:"total_subscriptions"`
	TotalConnections   int            `json:"total_connections"`
	TopicCounts        map[string]int `json:"topic_counts"`
}
