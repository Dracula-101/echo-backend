package messaging

import (
	"context"
)

type Broker interface {
	Publish(ctx context.Context, topic string, message *Message) error
	Subscribe(ctx context.Context, topic string, handler Handler) error
	Unsubscribe(ctx context.Context, topic string) error
	Close() error
}

type Producer interface {
	Send(ctx context.Context, topic string, message *Message) error
	SendBatch(ctx context.Context, topic string, messages []*Message) error
	Close() error
}

type Consumer interface {
	Consume(ctx context.Context, topics []string, handler Handler) error
	Close() error
}

type Handler interface {
	Handle(ctx context.Context, message *Message) error
}

type HandlerFunc func(ctx context.Context, message *Message) error

func (f HandlerFunc) Handle(ctx context.Context, message *Message) error {
	return f(ctx, message)
}

type Config struct {
	Brokers           []string
	ClientID          string
	GroupID           string
	MaxRetries        int
	RetryBackoff      int
	SessionTimeout    int
	HeartbeatInterval int
}
