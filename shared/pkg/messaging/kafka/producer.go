package kafka

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/IBM/sarama"

	pkgErrors "shared/pkg/errors"
	"shared/pkg/messaging"
)

type producer struct {
	producer sarama.SyncProducer
}

func NewProducer(cfg messaging.ProducerConfig) (messaging.Producer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("kafka producer: no brokers configured")
	}

	config := sarama.NewConfig()
	config.Version = sarama.V3_0_0_0
	config.ClientID = cfg.ClientID
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true
	config.Producer.Retry.Max = cfg.MaxRetries

	// Acks
	config.Producer.RequiredAcks = parseAcks(cfg.Acks)

	// Compression
	config.Producer.Compression = parseCompression(cfg.Compression)

	// Batching
	if cfg.BatchSize > 0 {
		config.Producer.Flush.Bytes = cfg.BatchSize
	}
	if cfg.LingerMs > 0 {
		config.Producer.Flush.Frequency = time.Duration(cfg.LingerMs) * time.Millisecond
	}

	// Idempotence requires Acks=all and MaxOpenRequests=1.
	if cfg.EnableIdempotence {
		config.Producer.Idempotent = true
		config.Producer.RequiredAcks = sarama.WaitForAll
		config.Net.MaxOpenRequests = 1
	} else if cfg.MaxInFlight > 0 {
		config.Net.MaxOpenRequests = cfg.MaxInFlight
	}

	// Network timeouts — fall back to 10s when not set.
	config.Net.DialTimeout = durOr(cfg.DialTimeout, 10*time.Second)
	config.Net.ReadTimeout = durOr(cfg.ReadTimeout, 10*time.Second)
	config.Net.WriteTimeout = durOr(cfg.WriteTimeout, 10*time.Second)
	config.Producer.Timeout = durOr(cfg.ProducerTimeout, 10*time.Second)

	prod, err := sarama.NewSyncProducer(cfg.Brokers, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka producer (brokers: %v): %w", cfg.Brokers, err)
	}

	return &producer{producer: prod}, nil
}

func (p *producer) Send(ctx context.Context, topic string, message *messaging.Message) pkgErrors.AppError {
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
			WithDetail("key", string(message.Key)).
			WithDetail("cause", err.Error())
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
			WithDetail("causes", causes)
	}

	return pkgErrors.FromError(err, pkgErrors.CodeInternal, "failed to send kafka batch (unknown error type)").
		WithService("kafka-producer").
		WithDetail("topic", topic).
		WithDetail("cause", err.Error())
}

func (p *producer) Close() error {
	return p.producer.Close()
}

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

func parseAcks(acks string) sarama.RequiredAcks {
	switch strings.ToLower(acks) {
	case "all", "-1":
		return sarama.WaitForAll
	case "local", "1":
		return sarama.WaitForLocal
	default:
		return sarama.WaitForAll // safe default
	}
}

func parseCompression(codec string) sarama.CompressionCodec {
	switch strings.ToLower(codec) {
	case "gzip":
		return sarama.CompressionGZIP
	case "lz4":
		return sarama.CompressionLZ4
	case "zstd":
		return sarama.CompressionZSTD
	case "none", "":
		return sarama.CompressionNone
	default: // "snappy" and anything unrecognised
		return sarama.CompressionSnappy
	}
}

// durOr returns d if non-zero, otherwise fallback.
func durOr(d, fallback time.Duration) time.Duration {
	if d > 0 {
		return d
	}
	return fallback
}
