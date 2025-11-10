package handler

import (
	"media-service/internal/config"
	"media-service/internal/service"

	"shared/pkg/logger"
	"shared/pkg/media"

	"github.com/go-playground/validator/v10"
)

type Handler struct {
	mediaService   *service.MediaService
	cfg            *config.Config
	log            logger.Logger
	validator      *validator.Validate
	mediaProcessor *media.Processor
}

func NewHandler(mediaService *service.MediaService, mediaProcessor *media.Processor, cfg *config.Config, log logger.Logger) *Handler {
	return &Handler{
		mediaService:   mediaService,
		cfg:            cfg,
		log:            log,
		validator:      validator.New(),
		mediaProcessor: mediaProcessor,
	}
}
