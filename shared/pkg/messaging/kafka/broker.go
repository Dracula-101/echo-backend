package kafka

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"

	pkgErrors "shared/pkg/errors"
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

func (b *broker) Publish(ctx context.Context, topic string, message *messaging.Message) pkgErrors.AppError {
	if b.producer == nil {
		producer, err := sarama.NewAsyncProducerFromClient(b.client)
		if err != nil {
			return pkgErrors.FromError(err, pkgErrors.CodeInternal, "failed to create producer").
				WithService("kafka-broker").
				WithDetail("topic", topic)
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
		return pkgErrors.FromError(ctx.Err(), pkgErrors.CodeInternal, "context cancelled while publishing").
			WithService("kafka-broker").
			WithDetail("topic", topic)
	}
}

func (b *broker) Subscribe(ctx context.Context, topic string, handler messaging.Handler) pkgErrors.AppError {
	return pkgErrors.New(pkgErrors.CodeNotImplemented, "not implemented: use consumer group instead").
		WithService("kafka-broker")
}

func (b *broker) Unsubscribe(ctx context.Context, topic string) pkgErrors.AppError {
	return pkgErrors.New(pkgErrors.CodeNotImplemented, "not implemented").
		WithService("kafka-broker")
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
