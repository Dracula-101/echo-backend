package messaging

import (
	"context"

	"echo-backend/services/message-service/internal/domain"
)

// EventPublisher is the full event publishing interface used by messageService
// and conversationService. It is a superset of service.EventPublisher.
type EventPublisher interface {
	PublishMessageEvent(ctx context.Context, event *domain.MessageEvent) error
	PublishChatMessage(ctx context.Context, event *domain.ChatMessageEvent) error
	PublishDeliveryEvent(ctx context.Context, event *DeliveryEvent) error
	PublishConversationEvent(ctx context.Context, event *domain.ConversationEvent) error
}
