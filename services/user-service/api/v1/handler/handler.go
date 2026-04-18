package handler

import (
	"user-service/internal/service"
	"user-service/internal/service/cache"
	"user-service/internal/service/location"
	"user-service/internal/service/search"

	"shared/pkg/logger"
	"shared/server/common/token"
)

type UserHandler struct {
	service         service.UserService
	searchService   search.SearchService
	locationService location.LocationService
	cache           cache.UserCache
	tokenService    *token.JWTTokenService
	log             logger.Logger
}

func NewUserHandler(service service.UserService, searchService search.SearchService, locationService location.LocationService, cache cache.UserCache, tokenService *token.JWTTokenService, log logger.Logger) *UserHandler {
	return &UserHandler{
		service:         service,
		searchService:   searchService,
		locationService: locationService,
		cache:           cache,
		tokenService:    tokenService,
		log:             log,
	}
}
