package kafka

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"

	"shared/pkg/messaging"
)

type broker struct {
	config   *sarama.Config
	client   sarama.Client
	producer sarama.AsyncProducer
	consumer sarama.ConsumerGroup
}

func NewBroker(cfg messaging.Config) (messaging.Broker, error) {
	config := sarama.NewConfig()
	config.Version = sarama.V3_0_0_0
	config.ClientID = cfg.ClientID
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	config.Consumer.Offsets.Initial = sarama.OffsetNewest

	client, err := sarama.NewClient(cfg.Brokers, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka client: %w", err)
	}

	return &broker{
		config: config,
		client: client,
	}, nil
}

func (b *broker) Publish(ctx context.Context, topic string, message *messaging.Message) error {
	if b.producer == nil {
		producer, err := sarama.NewAsyncProducerFromClient(b.client)
		if err != nil {
			return fmt.Errorf("failed to create producer: %w", err)
		}
		b.producer = producer

		go b.handleProducerErrors()
	}

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.ByteEncoder(message.Key),
		Value: sarama.ByteEncoder(message.Value),
	}

	for k, v := range message.Headers {
		msg.Headers = append(msg.Headers, sarama.RecordHeader{
			Key:   []byte(k),
			Value: []byte(v),
		})
	}

	select {
	case b.producer.Input() <- msg:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (b *broker) Subscribe(ctx context.Context, topic string, handler messaging.Handler) error {
	return fmt.Errorf("not implemented: use consumer group instead")
}

func (b *broker) Unsubscribe(ctx context.Context, topic string) error {
	return fmt.Errorf("not implemented")
}

func (b *broker) Close() error {
	if b.producer != nil {
		if err := b.producer.Close(); err != nil {
			return err
		}
	}

	if b.consumer != nil {
		if err := b.consumer.Close(); err != nil {
			return err
		}
	}

	return b.client.Close()
}

func (b *broker) handleProducerErrors() {
	for err := range b.producer.Errors() {
		fmt.Printf("Producer error: %v\n", err)
	}
}
