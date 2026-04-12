package handler

import (
	"auth-service/internal/repo/security_event"
	"auth-service/internal/service"
	"auth-service/internal/service/cache"
	"auth-service/internal/service/location"
	"auth-service/internal/service/session"
	"shared/pkg/logger"
)

type AuthHandler struct {
	authService       service.AuthService
	sessionService    session.SessionService
	locationService   location.LocationService
	securityEventRepo security_event.SecurityEventRepository
	authCache         cache.AuthCache
	log               logger.Logger
}

func NewAuthHandler(
	authService service.AuthService,
	sessionService session.SessionService,
	locationService location.LocationService,
	securityEventRepo security_event.SecurityEventRepository,
	authCache cache.AuthCache,
	log logger.Logger,
) *AuthHandler {
	return &AuthHandler{
		authService:       authService,
		sessionService:    sessionService,
		locationService:   locationService,
		securityEventRepo: securityEventRepo,
		authCache:         authCache,
		log:               log,
	}
}
