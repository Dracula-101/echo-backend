package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type PubSub struct {
	client *redis.Client
	pubsub *redis.PubSub
}

func NewPubSub(client *redis.Client) *PubSub {
	return &PubSub{
		client: client,
	}
}

func (p *PubSub) Publish(ctx context.Context, channel string, message interface{}) error {
	return p.client.Publish(ctx, channel, message).Err()
}

func (p *PubSub) Subscribe(ctx context.Context, channels ...string) (<-chan *redis.Message, error) {
	p.pubsub = p.client.Subscribe(ctx, channels...)

	_, err := p.pubsub.Receive(ctx)
	if err != nil {
		return nil, err
	}

	return p.pubsub.Channel(), nil
}

func (p *PubSub) Unsubscribe(ctx context.Context, channels ...string) error {
	if p.pubsub == nil {
		return nil
	}
	return p.pubsub.Unsubscribe(ctx, channels...)
}

func (p *PubSub) Close() error {
	if p.pubsub == nil {
		return nil
	}
	return p.pubsub.Close()
}
