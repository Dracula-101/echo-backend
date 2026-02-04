package redis

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/redis/go-redis/v9"

	"shared/pkg/logger"
)

type BroadcastMessage struct {
	Type       string `json:"type"`
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
	ExcludeIDs []string `json:"exclude_ids,omitempty"`
	Payload    []byte `json:"payload"`
	SourceInstance string `json:"source_instance"`
}

type MessageHandler func(msg *BroadcastMessage)

type PubSub struct {
	client     *Client
	instanceID string
	log        logger.Logger
	handlers   map[string][]MessageHandler
	mu         sync.RWMutex
	pubsub     *redis.PubSub
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

func NewPubSub(client *Client, instanceID string, log logger.Logger) *PubSub {
	return &PubSub{
		client:     client,
		instanceID: instanceID,
		log:        log,
		handlers:   make(map[string][]MessageHandler),
		stopCh:     make(chan struct{}),
	}
}

func (p *PubSub) Start(ctx context.Context) error {
	p.pubsub = p.client.Subscribe(ctx, ChannelBroadcast, ChannelPresence, ChannelTyping)

	_, err := p.pubsub.Receive(ctx)
	if err != nil {
		return err
	}

	p.wg.Add(1)
	go p.listen()

	p.log.Info("pubsub started",
		logger.String("instance_id", p.instanceID),
	)

	return nil
}

func (p *PubSub) Stop() error {
	close(p.stopCh)
	p.wg.Wait()

	if p.pubsub != nil {
		return p.pubsub.Close()
	}
	return nil
}

func (p *PubSub) OnMessage(channel string, handler MessageHandler) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handlers[channel] = append(p.handlers[channel], handler)
}

func (p *PubSub) Publish(ctx context.Context, channel string, msg *BroadcastMessage) error {
	msg.SourceInstance = p.instanceID

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return p.client.Publish(ctx, channel, data)
}

func (p *PubSub) PublishBroadcast(ctx context.Context, msg *BroadcastMessage) error {
	return p.Publish(ctx, ChannelBroadcast, msg)
}

func (p *PubSub) listen() {
	defer p.wg.Done()

	ch := p.pubsub.Channel()

	for {
		select {
		case <-p.stopCh:
			return
		case redisMsg, ok := <-ch:
			if !ok {
				return
			}
			p.handleMessage(redisMsg)
		}
	}
}

func (p *PubSub) handleMessage(redisMsg *redis.Message) {
	var msg BroadcastMessage
	if err := json.Unmarshal([]byte(redisMsg.Payload), &msg); err != nil {
		p.log.Error("failed to unmarshal pubsub message",
			logger.Error(err),
			logger.String("channel", redisMsg.Channel),
		)
		return
	}

	if msg.SourceInstance == p.instanceID {
		return
	}

	p.mu.RLock()
	handlers := p.handlers[redisMsg.Channel]
	p.mu.RUnlock()

	for _, handler := range handlers {
		handler(&msg)
	}
}
