package handler

import (
	"auth-service/internal/service"
	"shared/pkg/logger"
)

type AuthHandler struct {
	authService     service.AuthServiceInterface
	sessionService  service.SessionServiceInterface
	locationService service.LocationServiceInterface
	log             logger.Logger
}

func NewAuthHandler(authService service.AuthServiceInterface, sessionService service.SessionServiceInterface, locationService service.LocationServiceInterface, log logger.Logger) *AuthHandler {
	return &AuthHandler{
		authService:     authService,
		sessionService:  sessionService,
		locationService: locationService,
		log:             log,
	}
}
