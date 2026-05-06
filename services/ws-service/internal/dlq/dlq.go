// Package dlq publishes poison-pill messages to a Kafka dead-letter topic.
// "Poison" here means: deserialization failed, schema is wrong, or the message
// body is otherwise unrecoverable — keeping it on the live topic would either
// loop forever (if ReturnOnError is set) or get silently dropped.
//
// A separate dlq-consumer service is expected to drain these topics, retry
// with exponential backoff, and alert if a record fails N consecutive times.
// Until that lands, the DLQ is an inspection point: `kafka-console-consumer
// --topic chat-messages-dlq` shows whatever ws-service couldn't handle.
package dlq

import (
	"context"
	"time"

	"shared/pkg/logger"
	"shared/pkg/messaging"
)

type Publisher struct {
	producer messaging.Producer
	topic    string
	log      logger.Logger
}

func New(producer messaging.Producer, topic string, log logger.Logger) *Publisher {
	return &Publisher{producer: producer, topic: topic, log: log}
}

// Send wraps the original message with a reason header and forwards it to
// the DLQ topic. Best-effort: failure to publish to DLQ is logged but doesn't
// fail the caller (we'd rather drop than loop).
func (p *Publisher) Send(ctx context.Context, original *messaging.Message, reason string) {
	if p == nil || p.producer == nil {
		return
	}
	dctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	msg := messaging.NewMessage(original.Value).
		WithKey(original.Key).
		WithHeader("dlq_reason", reason).
		WithHeader("original_topic", original.Topic)
	for k, v := range original.Headers {
		msg = msg.WithHeader(k, v)
	}

	if err := p.producer.Send(dctx, p.topic, msg); err != nil {
		p.log.Error("failed to publish to DLQ",
			logger.String("topic", p.topic),
			logger.String("original_topic", original.Topic),
			logger.String("reason", reason),
			logger.Error(err),
		)
	}
}
