package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"echo-backend/services/message-service/internal/domain"
	msgError "echo-backend/services/message-service/internal/error"
	"echo-backend/services/message-service/internal/repo"
	"shared/pkg/cache"
	dbModels "shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/utils"
	"shared/server/common/hashing"
	"shared/server/env"
	"shared/server/request"

	"shared/pkg/logger"

	"github.com/google/uuid"
)

// MessageServiceInterface defines the contract for message service operations
type MessageServiceInterface interface {
	// Core message operations
	SendMessage(ctx context.Context, req *domain.SendMessageInput, deviceInfo request.DeviceInfo, clientIP string) (*domain.Message, *msgError.MessageError)
	GetMessages(ctx context.Context, conversationID string, limit int, beforeMessageID *string) ([]*domain.Message, bool, *msgError.MessageError)
	GetMessageByID(ctx context.Context, messageID string) (*domain.Message, *msgError.MessageError)

	// Event processing
	ProcessChatMessage(ctx context.Context, event *domain.ChatMessageEvent) pkgErrors.AppError
	ProcessChatMessages(ctx context.Context, events []*domain.ChatMessageEvent) pkgErrors.AppError
}

type messageService struct {
	messageRepo      repo.MessageRepository
	conversationRepo repo.ConversationRepository
	userRepo         repo.UserRepository
	eventPublisher   EventPublisher
	cache            cache.Cache
	hashing          hashing.HashingService
	logger           logger.Logger
}

func NewMessageService(
	messageRepo repo.MessageRepository,
	conversationRepo repo.ConversationRepository,
	userRepo repo.UserRepository,
	eventPublisher EventPublisher,
	cache cache.Cache,
	log logger.Logger,
) MessageServiceInterface {
	hashing, err := hashing.DefaultService()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize hashing manager: %v", err))
	}
	return &messageService{
		messageRepo:      messageRepo,
		conversationRepo: conversationRepo,
		eventPublisher:   eventPublisher,
		cache:            cache,
		userRepo:         userRepo,
		hashing:          *hashing,
		logger:           log,
	}
}

// SendMessage handles the complete flow of sending a message
func (s *messageService) SendMessage(ctx context.Context, req *domain.SendMessageInput, deviceInfo request.DeviceInfo, clientIP string) (*domain.Message, *msgError.MessageError) {
	dbErr := s.conversationRepo.ValidateConversationParticipant(req.ConversationID, req.SenderUserID)
	if dbErr != nil {
		s.logger.Error("Conversation participant validation failed",
			logger.String("service", msgError.ServiceName),
			logger.String("conversation_id", req.ConversationID.String()),
			logger.String("sender_user_id", req.SenderUserID.String()),
			logger.Error(dbErr),
		)
		return nil, s.handleValidationError(dbErr)
	}

	isBlocked, dbErr := s.userRepo.CheckForBlockedUser(req.SenderUserID)
	if dbErr != nil {
		s.logger.Error("Failed to check if user is blocked",
			logger.String("service", msgError.ServiceName),
			logger.String("user_id", req.SenderUserID.String()),
			logger.Error(dbErr),
		)
		return nil, &msgError.MessageError{
			Message: "Failed to check if user is blocked",
			Code:    msgError.CodeBlockedUserNotFound,
			Error:   dbErr,
		}
	}
	if isBlocked {
		s.logger.Warn("User is blocked from sending messages",
			logger.String("service", msgError.ServiceName),
			logger.String("user_id", req.SenderUserID.String()),
		)
		return nil, &msgError.MessageError{
			Message: "User is blocked from sending messages",
			Code:    msgError.CodeUserBlocked,
			Error:   pkgErrors.New(pkgErrors.CodePermissionDenied, "user is blocked"),
		}
	}
	rateLimitKey := fmt.Sprintf("rate:msg:%s", req.SenderUserID.String())
	rateLimit, cacheErr := s.cache.GetInt(ctx, rateLimitKey)
	if cacheErr != nil {
		s.logger.Error("Failed to get rate limit from cache",
			logger.String("service", msgError.ServiceName),
			logger.String("user_id", req.SenderUserID.String()),
			logger.Error(cacheErr),
		)
	}
	if rateLimit > 100 && !env.IsDevelopment() {
		s.logger.Warn("Rate limit exceeded for sending messages",
			logger.String("service", msgError.ServiceName),
			logger.String("user_id", req.SenderUserID.String()),
		)
		return nil, &msgError.MessageError{
			Message: "Rate limit exceeded for sending messages",
			Code:    msgError.CodeRateLimitExceeded,
			Error:   pkgErrors.New(pkgErrors.CodeResourceExhausted, "rate limit exceeded"),
		}
	} else {
		s.logger.Debug("Incrementing message send rate limit counter",
			logger.String("service", msgError.ServiceName),
			logger.String("user_id", req.SenderUserID.String()),
			logger.Int("current_rate", int(rateLimit)),
		)
		s.cache.Increment(ctx, rateLimitKey, 60)
	}

	validationErr := s.validateMessage(string(req.MessageType), req.Metadata)
	if validationErr != nil {
		return nil, validationErr
	}

	// Get conversation participants for delivery tracking
	participants, participantsErr := s.conversationRepo.GetConversationParticipants(ctx, req.ConversationID)
	if participantsErr != nil {
		s.logger.Error("Failed to get conversation participants",
			logger.String("service", msgError.ServiceName),
			logger.String("conversation_id", req.ConversationID.String()),
			logger.Error(participantsErr),
		)
		// Don't fail the message send, just log the error - delivery tracking will be incomplete
	}

	// Build recipient IDs list (all participants except sender)
	var recipientIDs []uuid.UUID
	for _, p := range participants {
		if p.UserID != req.SenderUserID {
			recipientIDs = append(recipientIDs, p.UserID)
		}
	}

	messageEvent := domain.ChatMessageEvent{
		ID:               uuid.New(),
		ConversationID:   req.ConversationID,
		SenderUserID:     req.SenderUserID,
		RecipientIDs:     recipientIDs,
		ParentMessageID:  req.ParentMessageID,
		MessageType:      req.MessageType,
		Content:          req.Content,
		ContentEncrypted: false,
		ContentHash:      s.hashing.HashString(req.Content),
		FormatType:       domain.MessageFormatPlain,
		Status:           domain.MessageStatusSent,
		Mentions:         json.RawMessage("[]"),
		Links:            json.RawMessage("[]"),
		Hashtags:         []string{},
		DeliveryCount:    0,
		ReadCount:        0,
		ReplyCount:       0,
		ReactionCount:    0,
		ForwardCount:     0,
		IsForwarded:      false,
		SentFromDeviceID: deviceInfo.ID,
		SentFromIP:       clientIP,
		Metadata:         utils.MarshalRawMessageSafe(req.Metadata),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	// publish event
	publishErr := s.eventPublisher.PublishChatMessage(ctx, &messageEvent)
	if publishErr != nil {
		s.logger.Error("Failed to publish chat message event",
			logger.String("service", msgError.ServiceName),
			logger.String("conversation_id", req.ConversationID.String()),
			logger.String("sender_user_id", req.SenderUserID.String()),
			logger.Error(publishErr),
		)
		return nil, &msgError.MessageError{
			Message: "Failed to publish chat message event",
			Code:    msgError.CodeMessageEventPublishFailed,
			Error:   pkgErrors.FromError(publishErr, pkgErrors.CodeInternal, "failed to publish chat message event"),
		}
	}

	s.logger.Debug("Chat message event published",
		logger.String("service", msgError.ServiceName),
		logger.String("conversation_id", req.ConversationID.String()),
		logger.String("sender_user_id", req.SenderUserID.String()),
	)

	msg := &domain.Message{
		ID:              messageEvent.ID,
		ConversationID:  messageEvent.ConversationID,
		SenderUserID:    messageEvent.SenderUserID,
		ParentMessageID: messageEvent.ParentMessageID,
		MessageType:     messageEvent.MessageType,
		Content:         messageEvent.Content,
		Status:          messageEvent.Status,
		ReadCount:       messageEvent.ReadCount,
		CreatedAt:       messageEvent.CreatedAt,
		UpdatedAt:       messageEvent.UpdatedAt,
		Metadata:        req.Metadata,
	}

	return msg, nil
}

func (s *messageService) handleValidationError(dbErr pkgErrors.AppError) *msgError.MessageError {
	switch dbErr.Code() {
	case msgError.CodeBlockedUserNotFound.String():
		return &msgError.MessageError{
			Message: "Participant not in conversation",
			Code:    msgError.CodeParticipantNotInConversation,
			Error:   dbErr,
		}
	case msgError.CodeParticipantLeftConversation.String():
		return &msgError.MessageError{
			Message: "Participant has left the conversation",
			Code:    msgError.CodeParticipantLeftConversation,
			Error:   dbErr,
		}
	case msgError.CodeParticipantRemovedFromConversation.String():
		return &msgError.MessageError{
			Message: "Participant has been removed from the conversation",
			Code:    msgError.CodeParticipantRemovedFromConversation,
			Error:   dbErr,
		}
	case msgError.CodeConversationInactive.String():
		return &msgError.MessageError{
			Message: "Conversation is inactive",
			Code:    msgError.CodeConversationInactive,
			Error:   dbErr,
		}
	case msgError.CodeParticipantNotAllowedToSendMessages.String():
		return &msgError.MessageError{
			Message: "Participant is not allowed to send messages",
			Code:    msgError.CodeParticipantNotAllowedToSendMessages,
			Error:   dbErr,
		}
	default:
		return &msgError.MessageError{
			Message: "Failed to validate conversation participant",
			Code:    msgError.CodeConversationValidationFailed,
			Error:   dbErr,
		}
	}
}

func (s *messageService) validateMessage(rawMsgType string, metadata *domain.MessageMetadata) *msgError.MessageError {
	msgType := domain.MessageType(rawMsgType)
	switch msgType {
	case domain.MessageTypeImage:
		if metadata == nil || metadata.MediaURL == "" {
			return &msgError.MessageError{
				Message: "Metadata with valid URL is required for image messages",
				Code:    msgError.CodeMissingMessageMetadata,
				Error:   pkgErrors.New(pkgErrors.CodeInvalidArgument, "missing metadata for image message"),
			}
		}
	case domain.MessageTypeVideo:
		if metadata == nil || metadata.MediaURL == "" {
			return &msgError.MessageError{
				Message: "Metadata with valid URL is required for video messages",
				Code:    msgError.CodeMissingMessageMetadata,
				Error:   pkgErrors.New(pkgErrors.CodeInvalidArgument, "missing metadata for video message"),
			}
		}
	case domain.MessageTypeFile:
		if metadata == nil || metadata.MediaURL == "" || metadata.FileName == "" {
			return &msgError.MessageError{
				Message: "Metadata with valid URL and filename is required for file messages",
				Code:    msgError.CodeMissingMessageMetadata,
				Error:   pkgErrors.New(pkgErrors.CodeInvalidArgument, "missing metadata for file message"),
			}
		}
	case domain.MessageTypeText:
		// No specific metadata validation for text messages
	case domain.MessageTypeAudio:
		if metadata == nil || metadata.MediaURL == "" {
			return &msgError.MessageError{
				Message: "Metadata with valid URL is required for audio messages",
				Code:    msgError.CodeMissingMessageMetadata,
				Error:   pkgErrors.New(pkgErrors.CodeInvalidArgument, "missing metadata for audio message"),
			}
		}
	case domain.MessageTypeLocation:
		if metadata == nil || metadata.Latitude == 0 || metadata.Longitude == 0 {
			return &msgError.MessageError{
				Message: "Metadata with valid latitude and longitude is required for location messages",
				Code:    msgError.CodeMissingMessageMetadata,
				Error:   pkgErrors.New(pkgErrors.CodeInvalidArgument, "missing metadata for location message"),
			}
		}
	default:
		return &msgError.MessageError{
			Message: "Unsupported message type",
			Code:    msgError.CodeMissingMessageMetadata,
			Error:   pkgErrors.New(pkgErrors.CodeInvalidArgument, "unsupported message type"),
		}
	}
	return nil
}

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
		if deliveryErr := s.messageRepo.CreateDeliveryStatuses(ctx, event.ID.String(), recipientStrings); deliveryErr != nil {
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

// ProcessChatMessage processes a ChatMessageEvent by persisting it to the database
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

// GetMessages retrieves messages for a conversation with cursor-based pagination
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

// GetMessageByID retrieves a single message by ID
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
