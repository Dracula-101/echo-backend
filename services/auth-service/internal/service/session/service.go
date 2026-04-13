package session

import (
	"auth-service/internal/config"
	authErrors "auth-service/internal/error"
	"auth-service/internal/repo/session"
	"shared/pkg/cache"
	"shared/pkg/logger"
	"shared/server/common/token"
)

type SessionService struct {
	repo         session.SessionRepository
	tokenService token.JWTTokenService
	cfg          config.CacheConfig
	sessionCache SessionCache
	log          logger.Logger
}

func NewSessionService(repo session.SessionRepository, c cache.Cache, token token.JWTTokenService, log logger.Logger, cfg config.CacheConfig) *SessionService {
	if log == nil {
		panic("Logger is required")
	}

	log.Info("Initializing SessionService",
		logger.String("service", authErrors.ServiceName),
	)

	return &SessionService{
		repo:         repo,
		sessionCache: NewSessionCache(c, log),
		tokenService: token,
		log:          log,
		cfg:          cfg,
	}
}

var _ SessionServiceInterface = (*SessionService)(nil)
