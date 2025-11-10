package handler

import (
	"auth-service/internal/service"
	"shared/pkg/logger"
)

type AuthHandler struct {
	service         *service.AuthService
	sessionService  *service.SessionService
	locationService *service.LocationService
	log             logger.Logger
}

func NewAuthHandler(service *service.AuthService, sessionService *service.SessionService, locationService *service.LocationService, log logger.Logger) *AuthHandler {
	return &AuthHandler{
		service:         service,
		sessionService:  sessionService,
		locationService: locationService,
		log:             log,
	}
}
