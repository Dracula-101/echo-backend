package pubsub

import (
	"context"
	"sync"

	"shared/pkg/logger"
)

// Subscriber is a subscriber to a topic
type Subscriber struct {
	ID      string
	Channel chan *Message
}

// Message represents a pub/sub message
type Message struct {
	Topic   string
	Payload []byte
	Metadata map[string]interface{}
}

// Broker is a message broker for pub/sub
type Broker struct {
	// topic -> subscribers
	subscribers map[string]map[string]*Subscriber
	mu          sync.RWMutex

	log logger.Logger
}

// NewBroker creates a new broker
func NewBroker(log logger.Logger) *Broker {
	return &Broker{
		subscribers: make(map[string]map[string]*Subscriber),
		log:         log,
	}
}

// Subscribe subscribes to a topic
func (b *Broker) Subscribe(ctx context.Context, topic string, subscriberID string, bufferSize int) *Subscriber {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.subscribers[topic] == nil {
		b.subscribers[topic] = make(map[string]*Subscriber)
	}

	sub := &Subscriber{
		ID:      subscriberID,
		Channel: make(chan *Message, bufferSize),
	}

	b.subscribers[topic][subscriberID] = sub

	b.log.Info("Subscriber added",
		logger.String("topic", topic),
		logger.String("subscriber_id", subscriberID),
	)

	// Cleanup on context cancellation
	go func() {
		<-ctx.Done()
		b.Unsubscribe(topic, subscriberID)
	}()

	return sub
}

// Unsubscribe unsubscribes from a topic
func (b *Broker) Unsubscribe(topic, subscriberID string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if subs, exists := b.subscribers[topic]; exists {
		if sub, ok := subs[subscriberID]; ok {
			close(sub.Channel)
			delete(subs, subscriberID)

			if len(subs) == 0 {
				delete(b.subscribers, topic)
			}

			b.log.Info("Subscriber removed",
				logger.String("topic", topic),
				logger.String("subscriber_id", subscriberID),
			)
		}
	}
}

// Publish publishes a message to a topic
func (b *Broker) Publish(msg *Message) int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	subs, exists := b.subscribers[msg.Topic]
	if !exists || len(subs) == 0 {
		return 0
	}

	count := 0
	for _, sub := range subs {
		select {
		case sub.Channel <- msg:
			count++
		default:
			b.log.Warn("Subscriber channel full, skipping",
				logger.String("topic", msg.Topic),
				logger.String("subscriber_id", sub.ID),
			)
		}
	}

	return count
}

// SubscriberCount returns the number of subscribers for a topic
func (b *Broker) SubscriberCount(topic string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if subs, exists := b.subscribers[topic]; exists {
		return len(subs)
	}
	return 0
}

// Topics returns all topics
func (b *Broker) Topics() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	topics := make([]string, 0, len(b.subscribers))
	for topic := range b.subscribers {
		topics = append(topics, topic)
	}
	return topics
}
