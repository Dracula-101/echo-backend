package kafka

import (
	"context"
	"fmt"
	"sync"

	"github.com/IBM/sarama"

	"shared/pkg/messaging"
)

type consumer struct {
	group   sarama.ConsumerGroup
	handler messaging.Handler
	wg      sync.WaitGroup
}

func NewConsumer(cfg messaging.Config) (messaging.Consumer, error) {
	config := sarama.NewConfig()
	config.Version = sarama.V3_0_0_0
	config.ClientID = cfg.ClientID
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	config.Consumer.Offsets.Initial = sarama.OffsetNewest
	config.Consumer.Return.Errors = true

	group, err := sarama.NewConsumerGroup(cfg.Brokers, cfg.GroupID, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka consumer group: %w", err)
	}

	return &consumer{
		group: group,
	}, nil
}

func (c *consumer) Consume(ctx context.Context, topics []string, handler messaging.Handler) error {
	c.handler = handler

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			if err := c.group.Consume(ctx, topics, c); err != nil {
				fmt.Printf("Consumer error: %v\n", err)
			}

			if ctx.Err() != nil {
				return
			}
		}
	}()

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for err := range c.group.Errors() {
			fmt.Printf("Consumer group error: %v\n", err)
		}
	}()

	return nil
}

func (c *consumer) Close() error {
	c.wg.Wait()
	return c.group.Close()
}
func (c *consumer) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (c *consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (c *consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		msg := &messaging.Message{
			Key:       message.Key,
			Value:     message.Value,
			Topic:     message.Topic,
			Partition: message.Partition,
			Offset:    message.Offset,
			Timestamp: message.Timestamp,
			Headers:   make(map[string]string),
			Metadata:  make(map[string]interface{}),
		}

		for _, header := range message.Headers {
			msg.Headers[string(header.Key)] = string(header.Value)
		}

		if err := c.handler.Handle(session.Context(), msg); err != nil {
			fmt.Printf("Handler error: %v\n", err)
			continue
		}

		session.MarkMessage(message, "")
	}

	return nil
}
