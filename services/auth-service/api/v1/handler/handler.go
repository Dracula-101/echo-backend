package handler

import (
	ratelimiter "auth-service/api/v1/rate_limiter"
	"auth-service/internal/firebase"
	"auth-service/internal/service"
	"auth-service/internal/service/cache"
	"auth-service/internal/service/location"
	"auth-service/internal/service/session"
	"shared/pkg/logger"
)

type AuthHandler struct {
	authService     service.AuthService
	sessionService  session.SessionService
	locationService location.LocationService
	authCache       cache.AuthCache
	rateLimiter     ratelimiter.RateLimiter
	firebaseClient  *firebase.Client
	log             logger.Logger
}

func NewAuthHandler(
	authService service.AuthService,
	sessionService session.SessionService,
	locationService location.LocationService,
	authCache cache.AuthCache,
	rateLimiter ratelimiter.RateLimiter,
	firebaseClient *firebase.Client,
	log logger.Logger,
) *AuthHandler {
	return &AuthHandler{
		authService:     authService,
		sessionService:  sessionService,
		locationService: locationService,
		authCache:       authCache,
		rateLimiter:     rateLimiter,
		firebaseClient:  firebaseClient,
		log:             log,
	}
}
