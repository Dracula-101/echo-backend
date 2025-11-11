package handler

import (
	"user-service/internal/service"

	"shared/pkg/logger"
	"shared/server/common/token"
)

type UserHandler struct {
	service         *service.UserService
	locationService *service.LocationService
	tokenService    *token.JWTTokenService
	log             logger.Logger
}

func NewUserHandler(service *service.UserService, locationService *service.LocationService, tokenService *token.JWTTokenService, log logger.Logger) *UserHandler {
	return &UserHandler{
		service:         service,
		locationService: locationService,
		tokenService:    tokenService,
		log:             log,
	}
}
