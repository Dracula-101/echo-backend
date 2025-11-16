package handler

import (
	"presence-service/internal/service"
	"shared/pkg/logger"
)

type PresenceHandler struct {
	service service.PresenceService
	log     logger.Logger
}

func NewPresenceHandler(service service.PresenceService, log logger.Logger) *PresenceHandler {
	return &PresenceHandler{
		service: service,
		log:     log,
	}
}
