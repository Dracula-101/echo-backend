package kafka

import (
	"shared/pkg/messaging"
)

func DefaultConfig() messaging.Config {
	return messaging.Config{
		Brokers:           []string{"localhost:9092"},
		ClientID:          "default-client",
		GroupID:           "default-group",
		MaxRetries:        3,
		RetryBackoff:      100,
		SessionTimeout:    10000,
		HeartbeatInterval: 3000,
	}
}
