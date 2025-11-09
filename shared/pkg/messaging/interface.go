package messaging

import (
	"context"

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
