package handler

import (
	"user-service/internal/repo/blocked_users"
	"user-service/internal/repo/devices"
	repoSettings "user-service/internal/repo/settings"
	"user-service/internal/service"
	"user-service/internal/service/cache"
	"user-service/internal/service/contact"
	"user-service/internal/service/location"
	"user-service/internal/service/search"

	"shared/pkg/logger"
	"shared/server/common/token"
)

type UserHandler struct {
	userService     service.UserService
	searchService   search.SearchService
	contactService  contact.ContactService
	locationService location.LocationService
	blockedRepo     blocked_users.BlockedRepository
	settingsRepo    repoSettings.SettingsRepository
	deviceRepo      devices.DeviceRepository
	cache           cache.UserCache
	tokenService    *token.JWTTokenService
	log             logger.Logger
}

func NewUserHandler(
	service service.UserService,
	searchService search.SearchService,
	contactService contact.ContactService,
	locationService location.LocationService,
	blockedRepo blocked_users.BlockedRepository,
	settingsRepo repoSettings.SettingsRepository,
	deviceRepo devices.DeviceRepository,
	cache cache.UserCache,
	tokenService *token.JWTTokenService,
	log logger.Logger,
) *UserHandler {
	return &UserHandler{
		userService:     service,
		searchService:   searchService,
		contactService:  contactService,
		locationService: locationService,
		blockedRepo:     blockedRepo,
		settingsRepo:    settingsRepo,
		deviceRepo:      deviceRepo,
		cache:           cache,
		tokenService:    tokenService,
		log:             log,
	}
}
