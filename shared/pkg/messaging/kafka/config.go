package kafka

import (
	"time"

	"shared/pkg/messaging"
)

// DefaultProducerConfig returns sensible production-ready defaults for a Kafka producer.
func DefaultProducerConfig() messaging.ProducerConfig {
	return messaging.ProducerConfig{
		Brokers:         []string{"localhost:9092"},
		ClientID:        "default-client",
		MaxRetries:      3,
		RetryBackoff:    100 * time.Millisecond,
		Compression:     "snappy",
		Acks:            "all",
		MaxInFlight:     5,
		DialTimeout:     10 * time.Second,
		ReadTimeout:     10 * time.Second,
		WriteTimeout:    10 * time.Second,
		ProducerTimeout: 10 * time.Second,
	}
}

// DefaultConsumerConfig returns sensible defaults for a Kafka consumer group.
func DefaultConsumerConfig() messaging.ConsumerConfig {
	return messaging.ConsumerConfig{
		Brokers:           []string{"localhost:9092"},
		ClientID:          "default-client",
		GroupID:           "default-group",
		SessionTimeout:    10 * time.Second,
		HeartbeatInterval: 3 * time.Second,
	}
}
