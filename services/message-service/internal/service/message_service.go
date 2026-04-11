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
	GetMessages(ctx context.Context, conversationID string, limit int, beforeMessageID *string) ([]*domain.Message, bool, *msgError.MessageError)
	GetMessagesAfter(ctx context.Context, conversationID string, afterMessageID string, limit int) ([]*domain.Message, bool, *msgError.MessageError)
	GetMessageByID(ctx context.Context, messageID string) (*domain.Message, *msgError.MessageError)

	// Message mutations
	EditMessage(ctx context.Context, messageID string, userID string, newContent string) (*domain.Message, *msgError.MessageError)
	DeleteMessage(ctx context.Context, messageID string, userID string, deleteForEveryone bool) *msgError.MessageError
	ForwardMessage(ctx context.Context, messageID string, userID string, targetConversationID string, deviceInfo request.DeviceInfo, clientIP string) (*domain.Message, *msgError.MessageError)
	PinMessage(ctx context.Context, messageID string, userID string) *msgError.MessageError
	UnpinMessage(ctx context.Context, messageID string, userID string) *msgError.MessageError
	GetPinnedMessages(ctx context.Context, conversationID string) ([]*domain.Message, *msgError.MessageError)
	MarkAsRead(ctx context.Context, messageID string, userID string) *msgError.MessageError
	GetDeliveryStatus(ctx context.Context, messageID string, userID string) (*domain.DeliveryStatusSummary, *msgError.MessageError)
	SearchMessages(ctx context.Context, userID string, query string, conversationID *string, limit int, offset int) ([]*domain.Message, int, *msgError.MessageError)
	SyncMessages(ctx context.Context, conversationID string, lastMessageID string, limit int) ([]*domain.Message, bool, *msgError.MessageError)
	GetThread(ctx context.Context, parentMessageID string, limit int, beforeMessageID *string) ([]*domain.Message, bool, *msgError.MessageError)

	// Event processing
	ProcessChatMessage(ctx context.Context, event *domain.ChatMessageEvent) pkgErrors.AppError
	ProcessChatMessages(ctx context.Context, events []*domain.ChatMessageEvent) pkgErrors.AppError
}

type messageService struct {
	messageRepo      repo.MessageRepository
	deliveryRepo     repo.DeliveryRepository
	conversationRepo repo.ConversationRepository
	userRepo         repo.UserRepository
	eventPublisher   EventPublisher
	cache            cache.Cache
	hashing          hashing.HashingService
	logger           logger.Logger
}

func NewMessageService(
	messageRepo repo.MessageRepository,
	deliveryRepo repo.DeliveryRepository,
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
		deliveryRepo:     deliveryRepo,
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
		s.logger.Warn("Failed to get rate limit from cache, allowing message",
			logger.String("service", msgError.ServiceName),
			logger.String("user_id", req.SenderUserID.String()),
			logger.Error(cacheErr),
		)
		rateLimit = 0
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
