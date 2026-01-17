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

	messageEvent := domain.ChatMessageEvent{
		ID:               uuid.New(),
		ConversationID:   req.ConversationID,
		SenderUserID:     req.SenderUserID,
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
		Mentions: func() []domain.Mention {
			var mentions []domain.Mention
			json.Unmarshal(messageEvent.Mentions, &mentions)
			return mentions
		}(),
		ReadCount: messageEvent.ReadCount,
		CreatedAt: messageEvent.CreatedAt,
		UpdatedAt: messageEvent.UpdatedAt,
		Metadata:  req.Metadata,
	}

	return msg, nil
}

func (s *messageService) handleValidationError(dbErr pkgErrors.AppError) *msgError.MessageError {
	switch dbErr.Code() {
	case msgError.ParticipantNotInConversationError.Code():
		return &msgError.MessageError{
			Message: "Participant not in conversation",
			Code:    msgError.CodeParticipantNotInConversation,
			Error:   dbErr,
		}
	case msgError.ParticipantLeftConversationError.Code():
		return &msgError.MessageError{
			Message: "Participant has left the conversation",
			Code:    msgError.CodeParticipantLeftConversation,
			Error:   dbErr,
		}
	case msgError.ParticipantRemovedFromConversationError.Code():
		return &msgError.MessageError{
			Message: "Participant has been removed from the conversation",
			Code:    msgError.CodeParticipantRemovedFromConversation,
			Error:   dbErr,
		}
	case msgError.ConversationInactiveError.Code():
		return &msgError.MessageError{
			Message: "Conversation is inactive",
			Code:    msgError.CodeConversationInactive,
			Error:   dbErr,
		}
	case msgError.ParticipantNotAllowedToSendMessagesError.Code():
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
