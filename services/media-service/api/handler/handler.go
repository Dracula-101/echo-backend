package handler

import (
	"media-service/internal/config"
	"media-service/internal/service"

	"github.com/go-playground/validator/v10"
	"shared/pkg/logger"
)

type Handler struct {
	mediaService *service.MediaService
	cfg          *config.Config
	log          logger.Logger
	validator    *validator.Validate
}

func NewHandler(mediaService *service.MediaService, cfg *config.Config, log logger.Logger) *Handler {
	return &Handler{
		mediaService: mediaService,
		cfg:          cfg,
		log:          log,
		validator:    validator.New(),
	}
}
