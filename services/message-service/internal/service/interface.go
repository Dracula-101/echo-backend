package service

import (
	"context"

	svcmodel "echo-backend/services/message-service/internal/service/model"

	"github.com/google/uuid"
)

// MessageServiceInterface defines the contract for message service operations
type MessageServiceInterface interface {
	// Core message operations
	SendMessage(ctx context.Context, req *svcmodel.SendMessageInput) (*svcmodel.Message, error)
	GetMessage(ctx context.Context, messageID uuid.UUID) (*svcmodel.Message, error)
	GetMessages(ctx context.Context, conversationID uuid.UUID, params *svcmodel.PaginationParams) (*svcmodel.MessagesResponse, error)
	EditMessage(ctx context.Context, messageID uuid.UUID, userID uuid.UUID, newContent string) error
	DeleteMessage(ctx context.Context, messageID uuid.UUID, userID uuid.UUID) error

	// Delivery and read receipts
	MarkAsDelivered(ctx context.Context, messageID, userID uuid.UUID) error
	MarkAsRead(ctx context.Context, messageID, userID uuid.UUID) error
	HandleReadReceipt(ctx context.Context, userID, messageID uuid.UUID) error
	MarkConversationAsRead(ctx context.Context, conversationID, userID uuid.UUID) error

	// Typing indicators
	SetTypingIndicator(ctx context.Context, conversationID, userID uuid.UUID, isTyping bool) error
}

// Ensure messageService implements MessageServiceInterface
var _ MessageServiceInterface = (*messageService)(nil)
