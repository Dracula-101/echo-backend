package app

import (
	"context"

	"shared/pkg/logger"
	"shared/pkg/messaging"
)

// consumerHandle bundles a Kafka consumer with the cancel func that
// stops its read goroutine. App holds a slice so shutdown can stop them
// in one pass.
type consumerHandle struct {
	name     string
	topic    string
	cancel   context.CancelFunc
	consumer messaging.Consumer
}

// startTopicConsumer launches a goroutine that subscribes the handler to
// the topic. The goroutine exits when ctx is cancelled.
func startTopicConsumer(
	ctx context.Context,
	consumer messaging.Consumer,
	handler messaging.Handler,
	topic string,
	log logger.Logger,
) {
	log.Info("Starting Kafka consumer", logger.String("topic", topic))
	if err := consumer.Consume(ctx, []string{topic}, handler); err != nil {
		if ctx.Err() == nil {
			log.Error("Consumer error",
				logger.String("topic", topic),
				logger.Error(err),
			)
		}
	}
}
