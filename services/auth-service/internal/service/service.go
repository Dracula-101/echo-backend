package service

import (
	"auth-service/internal/config"
	repository "auth-service/internal/repo"
	"shared/pkg/cache"
	"shared/pkg/logger"
	"shared/server/common/hashing"
	"shared/server/common/token"
)

type AuthService struct {
	repo           *repository.AuthRepository
	tokenService   token.JWTTokenService
	hashingService hashing.HashingService
	cache          cache.Cache
	cfg            *config.AuthConfig
	log            logger.Logger
	*repository.LoginHistoryRepo
}

func NewAuthServiceBuilder() *AuthServiceBuilder {
	return &AuthServiceBuilder{}
}

type AuthServiceBuilder struct {
	repo             *repository.AuthRepository
	loginHistoryRepo *repository.LoginHistoryRepo
	tokenService     token.JWTTokenService
	hashingService   hashing.HashingService
	cache            cache.Cache
	cfg              *config.AuthConfig
	log              logger.Logger
}

func (b *AuthServiceBuilder) WithRepo(repo *repository.AuthRepository) *AuthServiceBuilder {
	b.repo = repo
	return b
}

func (b *AuthServiceBuilder) WithLoginHistoryRepo(repo *repository.LoginHistoryRepo) *AuthServiceBuilder {
	b.loginHistoryRepo = repo
	return b
}

func (b *AuthServiceBuilder) WithTokenService(tokenService token.JWTTokenService) *AuthServiceBuilder {
	b.tokenService = tokenService
	return b
}

func (b *AuthServiceBuilder) WithHashingService(hashingService hashing.HashingService) *AuthServiceBuilder {
	b.hashingService = hashingService
	return b
}

func (b *AuthServiceBuilder) WithCache(cache cache.Cache) *AuthServiceBuilder {
	b.cache = cache
	return b
}

func (b *AuthServiceBuilder) WithConfig(cfg *config.AuthConfig) *AuthServiceBuilder {
	b.cfg = cfg
	return b
}

func (b *AuthServiceBuilder) WithLogger(log logger.Logger) *AuthServiceBuilder {
	b.log = log
	return b
}

func (b *AuthServiceBuilder) Build() *AuthService {
	if b.repo == nil {
		panic("AuthRepository is required")
	}
	if b.loginHistoryRepo == nil {
		panic("LoginHistoryRepo is required")
	}
	if b.cfg == nil {
		panic("Config is required")
	}
	if b.log == nil {
		panic("Logger is required")
	}

	b.log.Info("Building AuthService",
		logger.String("service", "auth-service"),
	)

	return &AuthService{
		repo:             b.repo,
		LoginHistoryRepo: b.loginHistoryRepo,
		tokenService:     b.tokenService,
		hashingService:   b.hashingService,
		cache:            b.cache,
		cfg:              b.cfg,
		log:              b.log,
	}
}

func (s *AuthService) TokenService() token.JWTTokenService {
	return s.tokenService
}

func (s *AuthService) HashingService() hashing.HashingService {
	return s.hashingService
}
