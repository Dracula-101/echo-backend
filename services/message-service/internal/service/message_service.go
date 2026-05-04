package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"echo-backend/services/message-service/internal/domain"
	msgError "echo-backend/services/message-service/internal/error"
	"echo-backend/services/message-service/internal/messaging"
	"echo-backend/services/message-service/internal/repo"
	"echo-backend/services/message-service/internal/service/cache"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/utils"
	"shared/server/common/hashing"
	"shared/server/request"

	"shared/pkg/logger"

	"github.com/google/uuid"
)

// MessageServiceInterface defines the contract for message service operations
type MessageServiceInterface interface {
	// Core message operations
	SendMessage(ctx context.Context, req *domain.SendMessageInput, deviceInfo request.DeviceInfo, clientIP string) (*domain.Message, *msgError.MsgError)
	GetMessages(ctx context.Context, conversationID string, limit int, beforeMessageID *string) ([]*domain.Message, bool, *msgError.MsgError)
	GetMessagesAfter(ctx context.Context, conversationID string, afterMessageID string, limit int) ([]*domain.Message, bool, *msgError.MsgError)
	GetMessageByID(ctx context.Context, messageID string) (*domain.Message, *msgError.MsgError)

	// Message mutations
	EditMessage(ctx context.Context, messageID string, userID string, newContent string) (*domain.Message, *msgError.MsgError)
	DeleteMessage(ctx context.Context, messageID string, userID string, deleteForEveryone bool) *msgError.MsgError
	ForwardMessage(ctx context.Context, messageID string, userID string, targetConversationID string, deviceInfo request.DeviceInfo, clientIP string) (*domain.Message, *msgError.MsgError)
	PinMessage(ctx context.Context, messageID string, userID string) *msgError.MsgError
	UnpinMessage(ctx context.Context, messageID string, userID string) *msgError.MsgError
	GetPinnedMessages(ctx context.Context, conversationID string) ([]*domain.Message, *msgError.MsgError)
	MarkAsRead(ctx context.Context, messageID string, userID string) *msgError.MsgError
	GetDeliveryStatus(ctx context.Context, messageID string, userID string) (*domain.DeliveryStatusSummary, *msgError.MsgError)
	SearchMessages(ctx context.Context, userID string, query string, conversationID *string, limit int, cursor *time.Time) ([]*domain.Message, int, *msgError.MsgError)
	SyncMessages(ctx context.Context, conversationID string, since time.Time, limit int) ([]*domain.Message, bool, *msgError.MsgError)
	GetThread(ctx context.Context, parentMessageID string, limit int, beforeMessageID *string) ([]*domain.Message, bool, *msgError.MsgError)

	// Event processing
	ProcessChatMessage(ctx context.Context, event *domain.ChatMessageEvent) pkgErrors.AppError
	ProcessChatMessages(ctx context.Context, events []*domain.ChatMessageEvent) pkgErrors.AppError
}

type messageService struct {
	messageRepo      repo.MessageRepository
	deliveryRepo     repo.DeliveryRepository
	conversationRepo repo.ConversationRepository
	userRepo         repo.UserRepository
	eventPublisher   messaging.EventPublisher
	cache            cache.MessageCache
	hashing          hashing.HashingService
	logger           logger.Logger
}

func NewMessageService(
	messageRepo repo.MessageRepository,
	deliveryRepo repo.DeliveryRepository,
	conversationRepo repo.ConversationRepository,
	userRepo repo.UserRepository,
	eventPublisher messaging.EventPublisher,
	cache cache.MessageCache,
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
func (s *messageService) SendMessage(ctx context.Context, req *domain.SendMessageInput, deviceInfo request.DeviceInfo, clientIP string) (*domain.Message, *msgError.MsgError) {
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
		return nil, &msgError.MsgError{
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
		return nil, &msgError.MsgError{
			Message: "User is blocked from sending messages",
			Code:    msgError.CodeUserBlocked,
			Error:   pkgErrors.New(pkgErrors.CodePermissionDenied, "user is blocked"),
		}
	}
	// rateLimitKey := fmt.Sprintf("rate:msg:%s", req.SenderUserID.String())
	// rateLimit, cacheErr := s.cache.GetInt(ctx, rateLimitKey)
	// if cacheErr != nil {
	// 	s.logger.Warn("Failed to get rate limit from cache, allowing message",
	// 		logger.String("service", msgError.ServiceName),
	// 		logger.String("user_id", req.SenderUserID.String()),
	// 		logger.Error(cacheErr),
	// 	)
	// 	rateLimit = 0
	// }
	// if rateLimit > 100 && !env.IsDevelopment() {
	// 	s.logger.Warn("Rate limit exceeded for sending messages",
	// 		logger.String("service", msgError.ServiceName),
	// 		logger.String("user_id", req.SenderUserID.String()),
	// 	)
	// 	return nil, &msgError.MsgError{
	// 		Message: "Rate limit exceeded for sending messages",
	// 		Code:    msgError.CodeRateLimitExceeded,
	// 		Error:   pkgErrors.New(pkgErrors.CodeResourceExhausted, "rate limit exceeded"),
	// 	}
	// } else {
	// 	s.logger.Debug("Incrementing message send rate limit counter",
	// 		logger.String("service", msgError.ServiceName),
	// 		logger.String("user_id", req.SenderUserID.String()),
	// 		logger.Int("current_rate", int(rateLimit)),
	// 	)
	// 	s.cache.Increment(ctx, rateLimitKey, 60)
	// }

	if validationErr := s.validateSendInput(req); validationErr != nil {
		return nil, validationErr
	}

	if req.ParentMessageID != nil {
		parent, perr := s.messageRepo.GetMessageByID(ctx, req.ParentMessageID.String())
		if perr != nil {
			return nil, &msgError.MsgError{
				Message: "Parent message not found",
				Code:    msgError.CodeMessageNotFound,
				Error:   perr,
			}
		}
		if parent.ConversationID != req.ConversationID.String() {
			return nil, &msgError.MsgError{
				Message: "parent_message_id does not belong to this conversation",
				Code:    msgError.CodeMessageNotFound,
				Error:   pkgErrors.New(pkgErrors.CodeInvalidArgument, "parent conversation mismatch"),
			}
		}
		if parent.IsDeleted {
			return nil, &msgError.MsgError{
				Message: "Cannot reply to a deleted message",
				Code:    msgError.CodeMessageDeleted,
				Error:   pkgErrors.New(pkgErrors.CodeInvalidArgument, "parent is deleted"),
			}
		}
	}

	if req.ForwardedFrom != nil {
		source, ferr := s.messageRepo.GetMessageByID(ctx, req.ForwardedFrom.String())
		if ferr != nil {
			req.ForwardedFrom = nil
		} else if source.SenderUserID != "" {
			senderUUID, parseErr := uuid.Parse(source.SenderUserID)
			if parseErr == nil {
				if blocked, _ := s.userRepo.CheckForBlockedUser(senderUUID); blocked {
					req.ForwardedFrom = nil
				}
			}
		}
	}

	participants, participantsErr := s.conversationRepo.GetConversationParticipants(ctx, req.ConversationID)
	if participantsErr != nil {
		s.logger.Error("Failed to get conversation participants",
			logger.String("service", msgError.ServiceName),
			logger.String("conversation_id", req.ConversationID.String()),
			logger.Error(participantsErr),
		)
	}

	participantSet := make(map[uuid.UUID]struct{}, len(participants))
	var recipientIDs []uuid.UUID
	for _, p := range participants {
		participantSet[p.UserID] = struct{}{}
		if p.UserID != req.SenderUserID {
			recipientIDs = append(recipientIDs, p.UserID)
		}
	}

	if len(req.Mentions) > 0 {
		filtered := req.Mentions[:0]
		for _, m := range req.Mentions {
			if _, ok := participantSet[m.UserID]; ok {
				filtered = append(filtered, m)
			}
		}
		req.Mentions = filtered
	}

	metadata := mergeMetadata(req)
	mentionsJSON := marshalMentions(req.Mentions)

	now := time.Now()
	messageID := uuid.New()
	if parsedID, parseErr := uuid.Parse(req.IdempotencyKey); parseErr == nil {
		// The send handler requires a UUID idempotency key. Reusing it as the
		// message ID keeps DB-first retries idempotent if Kafka publish fails.
		messageID = parsedID
	}

	var forwardedFrom *string
	if req.ForwardedFrom != nil {
		s := req.ForwardedFrom.String()
		forwardedFrom = &s
	}

	messageEvent := domain.ChatMessageEvent{
		ID:                     messageID,
		ConversationID:         req.ConversationID,
		SenderUserID:           req.SenderUserID,
		RecipientIDs:           recipientIDs,
		ParentMessageID:        req.ParentMessageID,
		MessageType:            req.MessageType,
		Content:                req.Content,
		ContentEncrypted:       false,
		ContentHash:            s.hashing.HashString(req.Content),
		FormatType:             domain.MessageFormatPlain,
		Status:                 domain.MessageStatusSent,
		Mentions:               mentionsJSON,
		Links:                  json.RawMessage("[]"),
		Hashtags:               []string{},
		DeliveryCount:          0,
		ReadCount:              0,
		ReplyCount:             0,
		ReactionCount:          0,
		ForwardCount:           0,
		IsForwarded:            req.ForwardedFrom != nil,
		ForwardedFromMessageID: forwardedFrom,
		MediaIDs:               uuidsToStrings(req.MediaIDs),
		StickerID:              uuidPtrToStringPtr(req.StickerID),
		GifID:                  uuidPtrToStringPtr(req.GifID),
		IsScheduled:            req.ScheduledAt != nil,
		ScheduledAt:            req.ScheduledAt,
		ExpireAfterSeconds:     req.ExpireAfter,
		SentFromDeviceID:       deviceInfo.ID,
		SentFromIP:             clientIP,
		Metadata:               utils.MarshalRawMessageSafe(metadata),
		CreatedAt:              now,
		UpdatedAt:              now,
	}

	if req.ExpireAfter != nil {
		expiry := now.Add(time.Duration(*req.ExpireAfter) * time.Second)
		messageEvent.ExpiresAt = &expiry
	}

	if persistErr := s.ProcessChatMessage(ctx, &messageEvent); persistErr != nil {
		s.logger.Error("Failed to persist chat message before publishing event",
			logger.String("service", msgError.ServiceName),
			logger.String("message_id", messageEvent.ID.String()),
			logger.String("conversation_id", req.ConversationID.String()),
			logger.String("sender_user_id", req.SenderUserID.String()),
			logger.Error(persistErr),
		)
		return nil, &msgError.MsgError{
			Message: "Failed to persist message",
			Code:    msgError.CodeDBError,
			Error:   persistErr,
		}
	}

	publishErr := s.eventPublisher.PublishChatMessage(ctx, &messageEvent)
	if publishErr != nil {
		s.logger.Error("Failed to publish chat message event",
			logger.String("service", msgError.ServiceName),
			logger.String("conversation_id", req.ConversationID.String()),
			logger.String("sender_user_id", req.SenderUserID.String()),
			logger.Error(publishErr),
		)
		return nil, &msgError.MsgError{
			Message: "Failed to publish chat message event",
			Code:    msgError.CodeMessageEventPublishFailed,
			Error:   pkgErrors.FromError(publishErr, pkgErrors.CodeInternal, "failed to publish chat message event"),
		}
	}

	msg := &domain.Message{
		ID:              messageEvent.ID,
		ConversationID:  messageEvent.ConversationID,
		SenderUserID:    messageEvent.SenderUserID,
		ParentMessageID: messageEvent.ParentMessageID,
		MessageType:     messageEvent.MessageType,
		Content:         messageEvent.Content,
		Status:          messageEvent.Status,
		ReadCount:       messageEvent.ReadCount,
		DeliveryCount:   messageEvent.DeliveryCount,
		IsScheduled:     messageEvent.IsScheduled,
		ExpiresAt:       messageEvent.ExpiresAt,
		CreatedAt:       messageEvent.CreatedAt,
		UpdatedAt:       messageEvent.UpdatedAt,
		Metadata:        metadata,
	}

	return msg, nil
}

func mergeMetadata(req *domain.SendMessageInput) *domain.MessageMetadata {
	md := &domain.MessageMetadata{}
	if req.Metadata != nil {
		*md = *req.Metadata
	}
	if len(req.MediaIDs) > 0 {
		ids := make([]string, 0, len(req.MediaIDs))
		for _, id := range req.MediaIDs {
			ids = append(ids, id.String())
		}
		md.MediaIDs = ids
	}
	if req.StickerID != nil {
		md.StickerID = req.StickerID.String()
	}
	if req.GifID != nil {
		md.GifID = req.GifID.String()
	}
	if req.Location != nil {
		md.Latitude = req.Location.Latitude
		md.Longitude = req.Location.Longitude
		md.LocationName = req.Location.Name
		md.Address = req.Location.Address
	}
	if req.Contact != nil {
		md.ContactName = req.Contact.Name
		md.ContactPhone = req.Contact.Phone
		md.ContactEmail = req.Contact.Email
	}
	return md
}

func marshalMentions(mentions []domain.Mention) json.RawMessage {
	if len(mentions) == 0 {
		return json.RawMessage("[]")
	}
	b, err := json.Marshal(mentions)
	if err != nil {
		return json.RawMessage("[]")
	}
	return b
}

func uuidsToStrings(ids []uuid.UUID) []string {
	if len(ids) == 0 {
		return nil
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}
	return out
}

func uuidPtrToStringPtr(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}
	s := id.String()
	return &s
}
