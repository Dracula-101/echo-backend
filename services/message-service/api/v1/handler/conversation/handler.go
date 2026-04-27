package conversation

import (
	svc "echo-backend/services/message-service/internal/service"
	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
)

type ConversationHandler struct {
	conversationService svc.ConversationService
	messageService      svc.MessageServiceInterface
	typingService       svc.TypingServiceInterface
	draftService        svc.DraftServiceInterface
	settingsService     svc.ConversationSettingsServiceInterface
	inviteService       svc.InviteServiceInterface
	log                 logger.Logger
}

func (h *ConversationHandler) authedUserID(handler *request.RequestHandler) (uuid.UUID, bool) {
	userIDStr, ok := request.GetUserIDFromContext(handler.Context())
	if !ok {
		response.UnauthorizedError(handler.Context(), handler.Request(), handler.Writer(), "User not authenticated", nil)
		return uuid.Nil, false
	}
	uid, err := uuid.Parse(userIDStr)
	if err != nil {
		response.UnauthorizedError(handler.Context(), handler.Request(), handler.Writer(), "Invalid user id", err)
		return uuid.Nil, false
	}
	return uid, true
}

func NewConversationHandler(
	conversationService svc.ConversationService,
	messageService svc.MessageServiceInterface,
	typingService svc.TypingServiceInterface,
	draftService svc.DraftServiceInterface,
	settingsService svc.ConversationSettingsServiceInterface,
	inviteService svc.InviteServiceInterface,
	log logger.Logger,
) *ConversationHandler {
	return &ConversationHandler{
		conversationService: conversationService,
		messageService:      messageService,
		typingService:       typingService,
		draftService:        draftService,
		settingsService:     settingsService,
		inviteService:       inviteService,
		log:                 log,
	}
}
