package session

import (
	"auth-service/internal/config"
	sessionRepo "auth-service/internal/repo/session"
	authErrors "auth-service/internal/error"
	"shared/pkg/logger"
	"shared/server/common/token"
)

type sessionService struct {
	repo         sessionRepo.SessionRepository
	tokenService token.JWTTokenService
	cfg          config.CacheConfig
	sessionCache SessionCache
	log          logger.Logger
}

func NewSessionService(repo sessionRepo.SessionRepository, sessionCache SessionCache, token token.JWTTokenService, log logger.Logger, cfg config.CacheConfig) SessionService {
	if log == nil {
		panic("Logger is required")
	}

	log.Info("Initializing SessionService",
		logger.String("service", authErrors.ServiceName),
	)

	return &sessionService{
		repo:         repo,
		sessionCache: sessionCache,
		tokenService: token,
		log:          log,
		cfg:          cfg,
	}
}

var _ SessionService = (*sessionService)(nil)
