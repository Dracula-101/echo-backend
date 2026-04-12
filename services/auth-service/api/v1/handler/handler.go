package handler

import (
	"auth-service/internal/repo/security_event"
	"auth-service/internal/service"
	"auth-service/internal/service/location"
	"auth-service/internal/service/session"
	"shared/pkg/logger"
)

type AuthHandler struct {
	authService       service.AuthServiceInterface
	sessionService    session.SessionService
	locationService   location.LocationService
	securityEventRepo security_event.SecurityEventRepository
	log               logger.Logger
}

func NewAuthHandler(
	authService service.AuthServiceInterface,
	sessionService session.SessionService,
	locationService location.LocationService,
	securityEventRepo security_event.SecurityEventRepository,
	log logger.Logger,
) *AuthHandler {
	return &AuthHandler{
		authService:       authService,
		sessionService:    sessionService,
		locationService:   locationService,
		securityEventRepo: securityEventRepo,
		log:               log,
	}
}
