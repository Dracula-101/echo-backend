package message

import (
	"echo-backend/services/message-service/internal/service"
	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"
)

type MessageHandler struct {
	messageService  service.MessageServiceInterface
	reactionService service.ReactionServiceInterface
	pollService     service.PollServiceInterface
	log             logger.Logger
}

func NewMessageHandler(
	messageService service.MessageServiceInterface,
	reactionService service.ReactionServiceInterface,
	pollService service.PollServiceInterface,
	log logger.Logger,
) *MessageHandler {
	return &MessageHandler{
		messageService:  messageService,
		reactionService: reactionService,
		pollService:     pollService,
		log:             log,
	}
}

func (h *MessageHandler) sendAppError(handler *request.RequestHandler, err error) {
	// Simple mapping for now
	response.RespondWithError(handler.Context(), handler.Request(), handler.Writer(), 400, err)
}
