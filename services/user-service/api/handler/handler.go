package handler

import (
	"user-service/internal/service"

	"shared/pkg/logger"
)

type UserHandler struct {
	service *service.UserService
	log     logger.Logger
}

func NewUserHandler(service *service.UserService, log logger.Logger) *UserHandler {
	return &UserHandler{
		service: service,
		log:     log,
	}
}
