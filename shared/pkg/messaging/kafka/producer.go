package kafka

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"

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

func (p *producer) Send(ctx context.Context, topic string, message *messaging.Message) error {
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

	_, _, err := p.producer.SendMessage(msg)
	return err
}

func (p *producer) SendBatch(ctx context.Context, topic string, messages []*messaging.Message) error {
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

	return p.producer.SendMessages(msgs)
}

func (p *producer) Close() error {
	return p.producer.Close()
}
