package conversation

import (
	svc "echo-backend/services/message-service/internal/service"
	"shared/pkg/logger"
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
