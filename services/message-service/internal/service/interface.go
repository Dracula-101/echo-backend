package service

import (
	"context"
	"echo-backend/services/message-service/internal/domain"
	pkgErrors "shared/pkg/errors"
)

// MessageServiceInterface defines the contract for message service operations
type MessageServiceInterface interface {
	// Core message operations
	SendMessage(ctx context.Context, req *domain.SendMessageInput) (*domain.Message, pkgErrors.AppError)
}

// Ensure messageService implements MessageServiceInterface
var _ MessageServiceInterface = (*messageService)(nil)

type ConversationServiceInterface interface {
}

// Ensure conversationService implements ConversationServiceInterface
var _ ConversationServiceInterface = (*conversationService)(nil)
