package service

import (
	"auth-service/internal/config"
	repository "auth-service/internal/repo"
	"shared/pkg/cache"
	"shared/pkg/circuitbreaker"
	"shared/pkg/logger"
	"shared/pkg/retry"
	"shared/server/common/hashing"
	"shared/server/common/token"
	"time"
)

type AuthService struct {
	repo           *repository.AuthRepository
	tokenService   token.JWTTokenService
	hashingService hashing.HashingService
	cache          cache.Cache
	cfg            *config.AuthConfig
	log            logger.Logger
	dbCircuit      *circuitbreaker.CircuitBreaker
	retryer        *retry.Retryer
	*repository.LoginHistoryRepo
	*repository.SecurityEventRepo
}

func NewAuthServiceBuilder() *AuthServiceBuilder {
	return &AuthServiceBuilder{}
}

type AuthServiceBuilder struct {
	repo              *repository.AuthRepository
	loginHistoryRepo  *repository.LoginHistoryRepo
	securityEventRepo *repository.SecurityEventRepo
	tokenService      token.JWTTokenService
	hashingService    hashing.HashingService
	cache             cache.Cache
	cfg               *config.AuthConfig
	log               logger.Logger
}

func (b *AuthServiceBuilder) WithRepo(repo *repository.AuthRepository) *AuthServiceBuilder {
	b.repo = repo
	return b
}

func (b *AuthServiceBuilder) WithLoginHistoryRepo(repo *repository.LoginHistoryRepo) *AuthServiceBuilder {
	b.loginHistoryRepo = repo
	return b
}

func (b *AuthServiceBuilder) WithSecurityEventRepo(repo *repository.SecurityEventRepo) *AuthServiceBuilder {
	b.securityEventRepo = repo
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
	if b.securityEventRepo == nil {
		panic("SecurityEventRepo is required")
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

	dbCircuit := circuitbreaker.New("auth-db", circuitbreaker.Config{
		MaxRequests: 2,
		Interval:    10 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts circuitbreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
		OnStateChange: func(name string, from, to circuitbreaker.State) {
			b.log.Info("Circuit breaker state changed",
				logger.String("circuit", name),
				logger.String("from", from.String()),
				logger.String("to", to.String()),
			)
		},
	})

	retryer := retry.New(retry.Config{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     2 * time.Second,
		Strategy:     retry.StrategyExponential,
		Multiplier:   2.0,
		OnRetry: func(attempt int, delay time.Duration, err error) {
			b.log.Warn("Retrying operation",
				logger.Int("attempt", attempt),
				logger.Duration("delay", delay),
				logger.Error(err),
			)
		},
	})

	return &AuthService{
		repo:              b.repo,
		LoginHistoryRepo:  b.loginHistoryRepo,
		SecurityEventRepo: b.securityEventRepo,
		tokenService:      b.tokenService,
		hashingService:    b.hashingService,
		cache:             b.cache,
		cfg:               b.cfg,
		log:               b.log,
		dbCircuit:         dbCircuit,
		retryer:           retryer,
	}
}

func (s *AuthService) TokenService() token.JWTTokenService {
	return s.tokenService
}

func (s *AuthService) HashingService() hashing.HashingService {
	return s.hashingService
}
