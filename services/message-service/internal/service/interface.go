package service

import (
	"context"
	pkgErrors "shared/pkg/errors"

	svcmodel "echo-backend/services/message-service/internal/service/model"
)

// MessageServiceInterface defines the contract for message service operations
type MessageServiceInterface interface {
	// Core message operations
	SendMessage(ctx context.Context, req *svcmodel.SendMessageInput) (*svcmodel.Message, pkgErrors.AppError)
}

// Ensure messageService implements MessageServiceInterface
var _ MessageServiceInterface = (*messageService)(nil)

type ConversationServiceInterface interface {
}

// Ensure conversationService implements ConversationServiceInterface
var _ ConversationServiceInterface = (*conversationService)(nil)
