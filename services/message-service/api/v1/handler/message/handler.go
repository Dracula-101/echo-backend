package message

import (
	"echo-backend/services/message-service/internal/service"
	"shared/pkg/logger"
)

type MessageHandler struct {
	messageService service.MessageServiceInterface
	log            logger.Logger
}

func NewMessageHandler(messageService service.MessageServiceInterface, log logger.Logger) *MessageHandler {
	return &MessageHandler{
		messageService: messageService,
		log:            log,
	}
}
