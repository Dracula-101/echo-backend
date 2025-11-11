package handler

import (
	"echo-backend/services/message-service/internal/service"
	"shared/pkg/logger"
)

type MessageHandler struct {
	service service.MessageService
	log     logger.Logger
}

func NewMessageHandler(messageService service.MessageService, log logger.Logger) *MessageHandler {
	return &MessageHandler{
		service: messageService,
		log:     log,
	}
}
