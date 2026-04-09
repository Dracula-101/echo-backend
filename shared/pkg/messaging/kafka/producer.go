package kafka

import (
	"context"
	"errors"
	"fmt"
	"time"

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

	// Timeout guards — prevent hanging forever if broker is unreachable
	config.Net.DialTimeout = 10 * time.Second
	config.Net.ReadTimeout = 10 * time.Second
	config.Net.WriteTimeout = 10 * time.Second
	config.Producer.Timeout = 10 * time.Second

	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("kafka producer: no brokers configured")
	}

	prod, err := sarama.NewSyncProducer(cfg.Brokers, config)
	if err != nil {
		// Surface the full sarama error — broker unreachable, auth failure, etc.
		return nil, fmt.Errorf("failed to create kafka producer (brokers: %v): %w", cfg.Brokers, err)
	}

	return &producer{producer: prod}, nil
}

func (p *producer) Send(ctx context.Context, topic string, message *messaging.Message) pkgErrors.AppError {
	// Respect context cancellation before doing network I/O
	if err := ctx.Err(); err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeInternal, "context cancelled before kafka send").
			WithService("kafka-producer").
			WithDetail("topic", topic)
	}

	if topic == "" {
		return pkgErrors.New(pkgErrors.CodeInternal, "kafka topic is empty").
			WithService("kafka-producer")
	}

	msg := buildProducerMessage(topic, message)

	_, _, err := p.producer.SendMessage(msg)
	if err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeInternal, "failed to send kafka message").
			WithService("kafka-producer").
			WithDetail("topic", topic).
			WithDetail("key", string(message.Key)). // Fix: was []byte, now readable string
			WithDetail("cause", err.Error())        // Fix: surface the actual sarama error
	}

	return nil
}

func (p *producer) SendBatch(ctx context.Context, topic string, messages []*messaging.Message) pkgErrors.AppError {
	if err := ctx.Err(); err != nil {
		return pkgErrors.FromError(err, pkgErrors.CodeInternal, "context cancelled before kafka batch send").
			WithService("kafka-producer").
			WithDetail("topic", topic)
	}

	if topic == "" {
		return pkgErrors.New(pkgErrors.CodeInternal, "kafka topic is empty").
			WithService("kafka-producer")
	}

	msgs := make([]*sarama.ProducerMessage, 0, len(messages))
	for _, message := range messages {
		msgs = append(msgs, buildProducerMessage(topic, message))
	}

	err := p.producer.SendMessages(msgs)
	if err == nil {
		return nil
	}

	// sarama.SendMessages returns ProducerErrors — a slice of per-message failures.
	// Unwrap it to get individual causes instead of a useless aggregate string.
	var prodErrs sarama.ProducerErrors
	if errors.As(err, &prodErrs) {
		causes := make([]string, 0, len(prodErrs))
		for _, pe := range prodErrs {
			causes = append(causes, fmt.Sprintf("key=%s: %s", pe.Msg.Key, pe.Err.Error()))
		}
		return pkgErrors.FromError(err, pkgErrors.CodeInternal, "failed to send kafka batch").
			WithService("kafka-producer").
			WithDetail("topic", topic).
			WithDetail("total_count", len(messages)).
			WithDetail("failed_count", len(prodErrs)).
			WithDetail("causes", causes) // each failed message's key + reason
	}

	// Fallback for unexpected non-ProducerErrors error type
	return pkgErrors.FromError(err, pkgErrors.CodeInternal, "failed to send kafka batch (unknown error type)").
		WithService("kafka-producer").
		WithDetail("topic", topic).
		WithDetail("cause", err.Error())
}

func (p *producer) Close() error {
	return p.producer.Close()
}

// buildProducerMessage extracts the shared message-building logic to avoid duplication.
func buildProducerMessage(topic string, message *messaging.Message) *sarama.ProducerMessage {
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

	return msg
}
