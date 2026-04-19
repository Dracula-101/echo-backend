package service

import (
	"fmt"

	"shared/pkg/logger"

	repository "user-service/internal/repo"
	"user-service/internal/repo/blocked_users"
	"user-service/internal/repo/contacts"
	"user-service/internal/repo/devices"
	"user-service/internal/repo/privacy_overrides"
	"user-service/internal/repo/settings"
	"user-service/internal/service/cache"
	"user-service/internal/service/location"
)

type Services struct {
	User     UserService
	Location location.LocationService
}

type ServiceBuilder struct {
	userRepo        repository.UserRepository
	blockedRepo     blocked_users.BlockedRepository
	contactsRepo    contacts.ContactRepository
	settingsRepo    settings.SettingsRepository
	privacyRepo     privacy_overrides.PrivacyOverrideRepository
	deviceRepo      devices.DeviceRepository
	cache           cache.UserCache
	log             logger.Logger
	locationEndpoint string
}

func NewServiceBuilder() *ServiceBuilder {
	return &ServiceBuilder{}
}

func (b *ServiceBuilder) WithUserRepo(repo repository.UserRepository) *ServiceBuilder {
	b.userRepo = repo
	return b
}

func (b *ServiceBuilder) WithBlockedRepo(repo blocked_users.BlockedRepository) *ServiceBuilder {
	b.blockedRepo = repo
	return b
}

func (b *ServiceBuilder) WithContactsRepo(repo contacts.ContactRepository) *ServiceBuilder {
	b.contactsRepo = repo
	return b
}

func (b *ServiceBuilder) WithSettingsRepo(repo settings.SettingsRepository) *ServiceBuilder {
	b.settingsRepo = repo
	return b
}

func (b *ServiceBuilder) WithPrivacyRepo(repo privacy_overrides.PrivacyOverrideRepository) *ServiceBuilder {
	b.privacyRepo = repo
	return b
}

func (b *ServiceBuilder) WithDeviceRepo(repo devices.DeviceRepository) *ServiceBuilder {
	b.deviceRepo = repo
	return b
}

func (b *ServiceBuilder) WithCache(cache cache.UserCache) *ServiceBuilder {
	b.cache = cache
	return b
}

func (b *ServiceBuilder) WithLogger(log logger.Logger) *ServiceBuilder {
	b.log = log
	return b
}

func (b *ServiceBuilder) WithLocationEndpoint(endpoint string) *ServiceBuilder {
	b.locationEndpoint = endpoint
	return b
}

func (b *ServiceBuilder) Build() (*Services, error) {
	if b.userRepo == nil {
		return nil, fmt.Errorf("user repository is required")
	}
	if b.log == nil {
		return nil, fmt.Errorf("logger is required")
	}

	userSvc := newUserService(b.userRepo, b.blockedRepo, b.contactsRepo, b.settingsRepo, b.privacyRepo, b.deviceRepo, b.cache, b.log)
	return &Services{
		User:     userSvc,
		Location: location.NewLocationService(b.locationEndpoint, b.log),
	}, nil
}
