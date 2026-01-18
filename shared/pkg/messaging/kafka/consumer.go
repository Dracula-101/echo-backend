package kafka

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/IBM/sarama"

	pkgErrors "shared/pkg/errors"
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
	config.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	config.Consumer.Return.Errors = true

	group, err := sarama.NewConsumerGroup(cfg.Brokers, cfg.GroupID, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka consumer group: %w", err)
	}

	return &consumer{
		group: group,
	}, nil
}

func (c *consumer) Consume(ctx context.Context, topics []string, handler messaging.Handler) pkgErrors.AppError {
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
	const batchSize = 50
	const batchWindow = 50 * time.Millisecond

	batch := make([]*sarama.ConsumerMessage, 0, batchSize)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}

		if batchHandler, ok := c.handler.(messaging.BatchHandler); ok {
			messages := make([]*messaging.Message, 0, len(batch))
			for _, message := range batch {
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

				messages = append(messages, msg)
			}

			if err := batchHandler.HandleBatch(session.Context(), messages); err != nil {
				fmt.Printf("Batch handler error: %v\n", err)
				return err
			}

			for _, message := range batch {
				session.MarkMessage(message, "")
			}

			batch = batch[:0]
			return nil
		}

		for _, message := range batch {
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

		batch = batch[:0]
		return nil
	}

	timer := time.NewTimer(batchWindow)
	defer timer.Stop()

	for {
		select {
		case message, ok := <-claim.Messages():
			if !ok {
				return flush()
			}

			batch = append(batch, message)
			if len(batch) >= batchSize {
				if err := flush(); err != nil {
					return err
				}
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(batchWindow)
			}

		case <-timer.C:
			if err := flush(); err != nil {
				return err
			}
			timer.Reset(batchWindow)

		case <-session.Context().Done():
			return session.Context().Err()
		}
	}
}
