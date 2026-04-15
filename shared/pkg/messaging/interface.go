package messaging

import (
	"context"
	"time"

	pkgErrors "shared/pkg/errors"
)

type Broker interface {
	Publish(ctx context.Context, topic string, message *Message) pkgErrors.AppError
	Subscribe(ctx context.Context, topic string, handler Handler) pkgErrors.AppError
	Unsubscribe(ctx context.Context, topic string) pkgErrors.AppError
	Close() error
}

type Producer interface {
	Send(ctx context.Context, topic string, message *Message) pkgErrors.AppError
	SendBatch(ctx context.Context, topic string, messages []*Message) pkgErrors.AppError
	Close() error
}

type Consumer interface {
	Consume(ctx context.Context, topics []string, handler Handler) pkgErrors.AppError
	Close() error
}

type Handler interface {
	Handle(ctx context.Context, message *Message) error
}

// BatchHandler is an optional interface for handlers that want batched delivery.
// When implemented, the consumer will deliver messages in batches and commit
// offsets only after the batch is processed successfully.
type BatchHandler interface {
	HandleBatch(ctx context.Context, messages []*Message) error
}

type HandlerFunc func(ctx context.Context, message *Message) error

func (f HandlerFunc) Handle(ctx context.Context, message *Message) error {
	return f(ctx, message)
}

// ProducerConfig holds all configuration for a Kafka producer.
type ProducerConfig struct {
	Brokers []string
	// ClientID identifies this producer to the broker.
	ClientID string
	// MaxRetries is the number of times to retry a failed send before giving up.
	MaxRetries int
	// RetryBackoff is the wait between retries.
	RetryBackoff time.Duration
	// Compression codec: "none", "gzip", "snappy", "lz4", "zstd". Defaults to "snappy".
	Compression string
	// Acks controls durability: "none", "local", "all". Defaults to "all".
	Acks string
	// BatchSize is the maximum number of bytes buffered before flushing (sarama Producer.Flush.Bytes).
	BatchSize int
	// LingerMs is how long the producer waits before flushing (sarama Producer.Flush.Frequency).
	LingerMs int
	// EnableIdempotence ensures exactly-once delivery per partition. Forces Acks=all and MaxInFlight=1.
	EnableIdempotence bool
	// MaxInFlight is the max number of in-flight requests per broker connection.
	MaxInFlight int
	// Network timeouts. Zero means use the sarama default (10s).
	DialTimeout     time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ProducerTimeout time.Duration
}

// ConsumerConfig holds all configuration for a Kafka consumer group.
type ConsumerConfig struct {
	Brokers  []string
	ClientID string
	GroupID  string
	// SessionTimeout is the max time a consumer can be out of contact before rebalance.
	SessionTimeout time.Duration
	// HeartbeatInterval is how often the consumer sends heartbeats to the broker.
	HeartbeatInterval time.Duration
}
