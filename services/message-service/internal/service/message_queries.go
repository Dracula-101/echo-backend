package service

import (
	"context"

	"echo-backend/services/message-service/internal/domain"
	msgError "echo-backend/services/message-service/internal/error"
	pkgErrors "shared/pkg/errors"

	"shared/pkg/logger"
)

// GetMessages retrieves messages for a conversation with cursor-based pagination.
func (s *messageService) GetMessages(ctx context.Context, conversationID string, limit int, beforeMessageID *string) ([]*domain.Message, bool, *msgError.MessageError) {
	s.logger.Debug("retrieving messages for conversation",
		logger.String("service", msgError.ServiceName),
		logger.String("conversation_id", conversationID),
		logger.Int("limit", limit),
	)

	// Fetch one extra message to check if there are more
	fetchLimit := limit + 1
	dbMessages, dbErr := s.messageRepo.GetMessagesByConversationID(ctx, conversationID, fetchLimit, beforeMessageID)
	if dbErr != nil {
		s.logger.Error("Failed to get messages from repository",
			logger.String("service", msgError.ServiceName),
			logger.String("conversation_id", conversationID),
			logger.Error(dbErr),
		)
		return nil, false, &msgError.MessageError{
			Message: "Failed to retrieve messages",
			Code:    msgError.CodeMessageRetrievalFailed,
			Error:   pkgErrors.FromError(dbErr, pkgErrors.CodeDatabaseError, "failed to retrieve messages"),
		}
	}

	// Check if there are more messages
	hasMore := len(dbMessages) > limit
	if hasMore {
		// Trim to requested limit
		dbMessages = dbMessages[:limit]
	}

	// Convert DB models to domain models
	messages := make([]*domain.Message, 0, len(dbMessages))
	for _, dbMsg := range dbMessages {
		msg := domain.NewMessage(*dbMsg)
		messages = append(messages, &msg)
	}

	s.logger.Debug("Messages retrieved successfully",
		logger.String("service", msgError.ServiceName),
		logger.String("conversation_id", conversationID),
		logger.Int("count", len(messages)),
		logger.Bool("has_more", hasMore),
	)

	return messages, hasMore, nil
}

// GetMessagesAfter retrieves messages created after a specific message ID, in chronological order.
func (s *messageService) GetMessagesAfter(ctx context.Context, conversationID string, afterMessageID string, limit int) ([]*domain.Message, bool, *msgError.MessageError) {
	s.logger.Debug("retrieving messages after cursor",
		logger.String("service", msgError.ServiceName),
		logger.String("conversation_id", conversationID),
		logger.String("after_message_id", afterMessageID),
		logger.Int("limit", limit),
	)

	// Fetch one extra message to check if there are more
	fetchLimit := limit + 1
	dbMessages, dbErr := s.messageRepo.GetMessagesAfterID(ctx, conversationID, afterMessageID, fetchLimit)
	if dbErr != nil {
		s.logger.Error("Failed to get messages after cursor",
			logger.String("service", msgError.ServiceName),
			logger.String("conversation_id", conversationID),
			logger.String("after_message_id", afterMessageID),
			logger.Error(dbErr),
		)
		return nil, false, &msgError.MessageError{
			Message: "Failed to retrieve messages",
			Code:    msgError.CodeMessageRetrievalFailed,
			Error:   pkgErrors.FromError(dbErr, pkgErrors.CodeDatabaseError, "failed to retrieve messages"),
		}
	}

	hasMore := len(dbMessages) > limit
	if hasMore {
		dbMessages = dbMessages[:limit]
	}

	messages := make([]*domain.Message, 0, len(dbMessages))
	for _, dbMsg := range dbMessages {
		msg := domain.NewMessage(*dbMsg)
		messages = append(messages, &msg)
	}

	s.logger.Debug("Messages after cursor retrieved successfully",
		logger.String("service", msgError.ServiceName),
		logger.String("conversation_id", conversationID),
		logger.Int("count", len(messages)),
		logger.Bool("has_more", hasMore),
	)

	return messages, hasMore, nil
}

// GetMessageByID retrieves a single message by ID.
func (s *messageService) GetMessageByID(ctx context.Context, messageID string) (*domain.Message, *msgError.MessageError) {
	s.logger.Debug("retrieving message by ID",
		logger.String("service", msgError.ServiceName),
		logger.String("message_id", messageID),
	)

	dbMsg, dbErr := s.messageRepo.GetMessageByID(ctx, messageID)
	if dbErr != nil {
		s.logger.Error("Failed to get message from repository",
			logger.String("service", msgError.ServiceName),
			logger.String("message_id", messageID),
			logger.Error(dbErr),
		)
		return nil, &msgError.MessageError{
			Message: "Failed to retrieve message",
			Code:    msgError.CodeMessageRetrievalFailed,
			Error:   pkgErrors.FromError(dbErr, pkgErrors.CodeDatabaseError, "failed to retrieve message"),
		}
	}

	msg := domain.NewMessage(*dbMsg)

	s.logger.Debug("Message retrieved successfully",
		logger.String("service", msgError.ServiceName),
		logger.String("message_id", messageID),
	)

	return &msg, nil
}
