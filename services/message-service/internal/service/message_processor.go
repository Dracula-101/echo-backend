package service

import (
	"context"

	"echo-backend/services/message-service/internal/domain"
	msgError "echo-backend/services/message-service/internal/error"
	dbModels "shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"

	"shared/pkg/logger"
)

// ProcessChatMessage processes a single ChatMessageEvent by persisting it to the database
// and creating delivery status records for all recipients.
func (s *messageService) ProcessChatMessage(ctx context.Context, event *domain.ChatMessageEvent) pkgErrors.AppError {
	dbErr := s.messageRepo.InsertMessage(ctx, event.ToDBModel())
	if dbErr != nil {
		s.logger.Error("Failed to insert chat message into database",
			logger.String("service", msgError.ServiceName),
			logger.String("message_id", event.ID.String()),
			logger.String("conversation_id", event.ConversationID.String()),
			logger.Error(dbErr),
		)
		return pkgErrors.FromError(dbErr, pkgErrors.CodeInternal, "failed to insert chat message")
	}

	// Create delivery status records for all recipients
	if len(event.RecipientIDs) > 0 {
		recipientStrings := make([]string, len(event.RecipientIDs))
		for i, id := range event.RecipientIDs {
			recipientStrings[i] = id.String()
		}
		if deliveryErr := s.deliveryRepo.CreateDeliveryStatuses(ctx, event.ID.String(), recipientStrings); deliveryErr != nil {
			s.logger.Error("Failed to create delivery statuses",
				logger.String("service", msgError.ServiceName),
				logger.String("message_id", event.ID.String()),
				logger.Error(deliveryErr),
			)
			// Don't fail the message processing, just log the error
		}
	}

	s.logger.Debug("Chat message inserted into database",
		logger.String("service", msgError.ServiceName),
		logger.String("message_id", event.ID.String()),
		logger.String("conversation_id", event.ConversationID.String()),
	)
	return nil
}

// ProcessChatMessages processes a batch of ChatMessageEvents by persisting them to the database.
func (s *messageService) ProcessChatMessages(ctx context.Context, event []*domain.ChatMessageEvent) pkgErrors.AppError {
	messages := make([]*dbModels.Message, 0, len(event))
	for _, e := range event {
		messages = append(messages, e.ToDBModel())
	}

	dbErr := s.messageRepo.InsertMessages(ctx, messages)
	if dbErr != nil {
		s.logger.Error("Failed to insert chat messages into database",
			logger.String("service", msgError.ServiceName),
			logger.Error(dbErr),
		)
		return pkgErrors.FromError(dbErr, pkgErrors.CodeInternal, "failed to insert chat messages")
	}

	s.logger.Debug("Chat messages inserted into database",
		logger.String("service", msgError.ServiceName),
		logger.Int("batch_size", len(messages)),
	)
	return nil
}
