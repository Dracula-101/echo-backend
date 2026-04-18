package service

import (
	"fmt"

	"shared/pkg/logger"

	repository "user-service/internal/repo"
	"user-service/internal/service/cache"
	"user-service/internal/service/location"
)

type Services struct {
	User     UserService
	Location location.LocationService
}

type ServiceBuilder struct {
	userRepo         repository.UserRepository
	cache            cache.UserCache
	log              logger.Logger
	locationEndpoint string
}

func NewServiceBuilder() *ServiceBuilder {
	return &ServiceBuilder{}
}

func (b *ServiceBuilder) WithUserRepo(repo repository.UserRepository) *ServiceBuilder {
	b.userRepo = repo
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

	userSvc := newUserService(b.userRepo, b.cache, b.log)
	return &Services{
		User:     userSvc,
		Location: location.NewLocationService(b.locationEndpoint, b.log),
	}, nil
}
