package service

import (
	"context"

	"echo-backend/services/message-service/internal/model"

	"github.com/google/uuid"
)

// MessageServiceInterface defines the contract for message service operations
type MessageServiceInterface interface {
	// Core message operations
	SendMessage(ctx context.Context, req *model.SendMessageRequest) (*model.Message, error)
	GetMessage(ctx context.Context, messageID uuid.UUID) (*model.Message, error)
	GetMessages(ctx context.Context, conversationID uuid.UUID, params *model.PaginationParams) (*model.MessagesResponse, error)
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
