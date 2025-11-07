package handler

import (
	"media-service/internal/config"
	"media-service/internal/service"

	"shared/pkg/logger"
)

type MediaHandler struct {
	service *service.MediaService
	cfg     *config.Config
	log     logger.Logger
}

func NewMediaHandler(service *service.MediaService, cfg *config.Config, log logger.Logger) *MediaHandler {
	return &MediaHandler{
		service: service,
		cfg:     cfg,
		log:     log,
	}
}
