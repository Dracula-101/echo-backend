package service

import (
	"fmt"

	"shared/pkg/cache"
	"shared/pkg/logger"

	repository "user-service/internal/repo"
)

type Services struct {
	User     UserServiceInterface
	Location LocationServiceInterface
}

type ServiceBuilder struct {
	userRepo         repository.UserRepositoryInterface
	cache            cache.Cache
	log              logger.Logger
	locationEndpoint string
}

func NewServiceBuilder() *ServiceBuilder {
	return &ServiceBuilder{}
}

func (b *ServiceBuilder) WithUserRepo(repo repository.UserRepositoryInterface) *ServiceBuilder {
	b.userRepo = repo
	return b
}

func (b *ServiceBuilder) WithCache(cache cache.Cache) *ServiceBuilder {
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

	var locationSvc LocationServiceInterface
	if b.locationEndpoint != "" {
		locationSvc = NewLocationService(b.locationEndpoint, b.log)
	}

	return &Services{
		User:     userSvc,
		Location: locationSvc,
	}, nil
}
