package service

import (
	ratelimiter "auth-service/api/v1/rate_limiter"
	"auth-service/internal/config"
	repository "auth-service/internal/repo"
	"auth-service/internal/repo/email_verification"
	"auth-service/internal/repo/login_history"
	"auth-service/internal/repo/password_reset"
	"auth-service/internal/repo/session"
	"auth-service/internal/service/cache"
	sessionSvc "auth-service/internal/service/session"
	"shared/pkg/email"
	"shared/pkg/logger"
	"shared/server/common/hashing"
	"shared/server/common/token"
)

type authService struct {
	repo                  repository.AuthRepository
	loginHistoryRepo      login_history.LoginHistoryRespository
	sessionRepo           session.SessionRepository
	passwordResetRepo     password_reset.PasswordResetTokenRepository
	emailVerificationRepo email_verification.EmailVerificationTokenRepository
	authCache             cache.AuthCache
	sessionCache          sessionSvc.SessionCache
	emailService          email.EmailService
	tokenService          token.JWTTokenService
	hashingService        hashing.HashingService
	rateLimiter           ratelimiter.RateLimiter
	cfg                   *config.AuthConfig
	log                   logger.Logger
}

func NewAuthServiceBuilder() *AuthServiceBuilder {
	return &AuthServiceBuilder{}
}

type AuthServiceBuilder struct {
	repo                  repository.AuthRepository
	loginHistoryRepo      login_history.LoginHistoryRespository
	sessionRepo           session.SessionRepository
	passwordResetRepo     password_reset.PasswordResetTokenRepository
	emailVerificationRepo email_verification.EmailVerificationTokenRepository
	authCache             cache.AuthCache
	sessionCache          sessionSvc.SessionCache
	emailService          email.EmailService
	tokenService          token.JWTTokenService
	hashingService        hashing.HashingService
	rateLimiter           ratelimiter.RateLimiter
	cfg                   *config.AuthConfig
	log                   logger.Logger
}

func (b *AuthServiceBuilder) WithRepo(repo repository.AuthRepository) *AuthServiceBuilder {
	b.repo = repo
	return b
}

func (b *AuthServiceBuilder) WithLoginHistoryRepo(repo login_history.LoginHistoryRespository) *AuthServiceBuilder {
	b.loginHistoryRepo = repo
	return b
}

func (b *AuthServiceBuilder) WithSessionRepo(repo session.SessionRepository) *AuthServiceBuilder {
	b.sessionRepo = repo
	return b
}

func (b *AuthServiceBuilder) WithAuthCache(cache cache.AuthCache) *AuthServiceBuilder {
	b.authCache = cache
	return b
}

func (b *AuthServiceBuilder) WithSessionCache(sessionCache sessionSvc.SessionCache) *AuthServiceBuilder {
	b.sessionCache = sessionCache
	return b
}

func (b *AuthServiceBuilder) WithPasswordResetRepo(repo password_reset.PasswordResetTokenRepository) *AuthServiceBuilder {
	b.passwordResetRepo = repo
	return b
}

func (b *AuthServiceBuilder) WithEmailVerificationRepo(repo email_verification.EmailVerificationTokenRepository) *AuthServiceBuilder {
	b.emailVerificationRepo = repo
	return b
}

func (b *AuthServiceBuilder) WithRateLimiter(rateLimiter ratelimiter.RateLimiter) *AuthServiceBuilder {
	b.rateLimiter = rateLimiter
	return b
}

func (b *AuthServiceBuilder) WithEmailService(emailService email.EmailService) *AuthServiceBuilder {
	b.emailService = emailService
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

func (b *AuthServiceBuilder) WithConfig(cfg *config.AuthConfig) *AuthServiceBuilder {
	b.cfg = cfg
	return b
}

func (b *AuthServiceBuilder) WithLogger(log logger.Logger) *AuthServiceBuilder {
	b.log = log
	return b
}

func (b *AuthServiceBuilder) Build() AuthService {
	if b.cfg == nil {
		panic("Config is required")
	}
	if b.log == nil {
		panic("Logger is required")
	}

	b.log.Info("Building AuthService",
		logger.String("service", "auth-service"),
	)

	return &authService{
		repo:                  b.repo,
		loginHistoryRepo:      b.loginHistoryRepo,
		sessionRepo:           b.sessionRepo,
		passwordResetRepo:     b.passwordResetRepo,
		emailVerificationRepo: b.emailVerificationRepo,
		authCache:             b.authCache,
		sessionCache:          b.sessionCache,
		emailService:          b.emailService,
		tokenService:          b.tokenService,
		hashingService:        b.hashingService,
		rateLimiter:           b.rateLimiter,
		cfg:                   b.cfg,
		log:                   b.log,
	}
}

func (s *authService) TokenService() token.JWTTokenService {
	return s.tokenService
}

func (s *authService) HashingService() hashing.HashingService {
	return s.hashingService
}

func (s *authService) EmailService() email.EmailService {
	return s.emailService
}
