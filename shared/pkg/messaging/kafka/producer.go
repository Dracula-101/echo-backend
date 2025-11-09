package kafka

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"

	pkgErrors "shared/pkg/errors"
	"shared/pkg/messaging"
)

type producer struct {
	producer sarama.SyncProducer
}

func NewProducer(cfg messaging.Config) (messaging.Producer, error) {
	config := sarama.NewConfig()
	config.Version = sarama.V3_0_0_0
	config.ClientID = cfg.ClientID
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = cfg.MaxRetries
	config.Producer.Compression = sarama.CompressionSnappy

	prod, err := sarama.NewSyncProducer(cfg.Brokers, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka producer: %w", err)
	}

	return &producer{producer: prod}, nil
}

func (p *producer) Send(ctx context.Context, topic string, message *messaging.Message) pkgErrors.AppError {
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

	if _, _, err := p.producer.SendMessage(msg); err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeInternal, "failed to send message").
			WithService("kafka-producer").
			WithDetail("topic", topic)
	}
	return nil
}

func (p *producer) SendBatch(ctx context.Context, topic string, messages []*messaging.Message) pkgErrors.AppError {
	msgs := make([]*sarama.ProducerMessage, 0, len(messages))

	for _, message := range messages {
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

		msgs = append(msgs, msg)
	}

	if err := p.producer.SendMessages(msgs); err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeInternal, "failed to send batch messages").
			WithService("kafka-producer").
			WithDetail("topic", topic).
			WithDetail("count", len(messages))
	}
	return nil
}

func (p *producer) Close() error {
	return p.producer.Close()
}
