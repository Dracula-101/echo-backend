package handler

import (
	"user-service/internal/service"

	"shared/pkg/logger"
	"shared/server/common/token"
)

type UserHandler struct {
	service      *service.UserService
	tokenService *token.JWTTokenService
	log          logger.Logger
}

func NewUserHandler(service *service.UserService, tokenService *token.JWTTokenService, log logger.Logger) *UserHandler {
	return &UserHandler{
		service:      service,
		tokenService: tokenService,
		log:          log,
	}
}
