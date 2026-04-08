package handler

import (
	repository "auth-service/internal/repo"
	"auth-service/internal/service"
	"shared/pkg/logger"
)

type AuthHandler struct {
	authService       service.AuthServiceInterface
	sessionService    service.SessionServiceInterface
	locationService   service.LocationServiceInterface
	securityEventRepo repository.SecurityEventRepositoryInterface
	log               logger.Logger
}

func NewAuthHandler(
	authService service.AuthServiceInterface,
	sessionService service.SessionServiceInterface,
	locationService service.LocationServiceInterface,
	securityEventRepo repository.SecurityEventRepositoryInterface,
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
