package handler

import (
	"user-service/internal/service"

	"shared/pkg/logger"
	"shared/server/common/token"
)

type UserHandler struct {
	service         service.UserServiceInterface
	searchService   service.SearchServiceInterface
	locationService service.LocationServiceInterface
	tokenService    *token.JWTTokenService
	log             logger.Logger
}

func NewUserHandler(service service.UserServiceInterface, searchService service.SearchServiceInterface, locationService service.LocationServiceInterface, tokenService *token.JWTTokenService, log logger.Logger) *UserHandler {
	return &UserHandler{
		service:         service,
		searchService:   searchService,
		locationService: locationService,
		tokenService:    tokenService,
		log:             log,
	}
}
