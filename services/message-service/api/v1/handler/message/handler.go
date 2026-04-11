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
	bookmarkService service.BookmarkServiceInterface
	reportService   service.ReportServiceInterface
	log             logger.Logger
}

func NewMessageHandler(
	messageService service.MessageServiceInterface,
	reactionService service.ReactionServiceInterface,
	pollService service.PollServiceInterface,
	bookmarkService service.BookmarkServiceInterface,
	reportService service.ReportServiceInterface,
	log logger.Logger,
) *MessageHandler {
	return &MessageHandler{
		messageService:  messageService,
		reactionService: reactionService,
		pollService:     pollService,
		bookmarkService: bookmarkService,
		reportService:   reportService,
		log:             log,
	}
}

func (h *MessageHandler) sendAppError(handler *request.RequestHandler, err error) {
	response.RespondWithError(handler.Context(), handler.Request(), handler.Writer(), 400, err)
}
