package service

import (
	"auth-service/internal/config"
	repository "auth-service/internal/repo"
	"context"
	"shared/pkg/cache"
	"shared/pkg/logger"
)

type AuthService struct {
	repo  *repository.AuthRepository
	cache cache.Cache
	cfg   *config.Config
	log   logger.Logger
}

func NewAuthService(repo *repository.AuthRepository, cache cache.Cache, cfg *config.Config, log logger.Logger) *AuthService {
	return &AuthService{
		repo:  repo,
		cache: cache,
		cfg:   cfg,
		log:   log,
	}
}

func (s *AuthService) Register(ctx context.Context, email, password string) error {
	s.log.Info("Registering user", logger.String("email", email))
	return s.repo.CreateUser(ctx, email, password)
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	s.log.Info("User login", logger.String("email", email))
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", nil
	}
	return "token", nil
}
