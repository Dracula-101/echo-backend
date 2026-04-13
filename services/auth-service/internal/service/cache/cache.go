package cache

import (
	authError "auth-service/internal/error"
	"context"
	"shared/pkg/cache"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/pkg/utils"
	"time"
)

// Read rl:register:ip:{X-Forwarded-For} sorted set. Remove members with score older than 3600 seconds (sliding window cleanup). Count remaining members. If count >= 5: return 429 with Retry-After header set to the oldest member's timestamp + 3600 - now. Do not proceed to any DB operation.

var (
	RegisterEmailLockPrefix = "lock:register:"
	IPBlockedPrefix         = "ip_blocked:"
)

var (
	RegisterEmailLockTTL = 10 * time.Second
)

type AuthCache interface {
	AcquireRegisterEmailLock(context context.Context, email string) *authError.AuthError
	ReleaseRegisterEmailLock(context context.Context, email string) *authError.AuthError
	CheckIPBlocked(context context.Context, ip string) *authError.AuthError
	BlockIP(context context.Context, ip string, duration time.Duration) *authError.AuthError
}

type authCache struct {
	cache cache.Cache
	log   logger.Logger
}

func NewAuthCache(cache cache.Cache, log logger.Logger) AuthCache {
	return &authCache{
		cache: cache,
		log:   log,
	}
}

func (s *authCache) AcquireRegisterEmailLock(ctx context.Context, email string) *authError.AuthError {
	s.log.Info("Acquiring email lock for registration", logger.String("email", email))
	email = utils.NormalizeEmail(email)
	s.log.Info("Locking email for registration", logger.String("email", email))
	email = utils.NormalizeEmail(email)

	locked, err := s.cache.AcquireLock(ctx, RegisterEmailLockPrefix+email, RegisterEmailLockTTL)
	if err != nil {
		s.log.Error("Failed to acquire email lock", logger.String("email", email), logger.Error(err))
		return &authError.AuthError{
			Message: "Failed to acquire email lock",
			Code:    authError.CodeCacheError,
			Error: pkgErrors.FromError(err, authError.CodeCacheError, "failed to acquire email lock").
				WithService(authError.ServiceName).
				WithDetail("email", email),
		}
	}
	if !locked {
		return &authError.AuthError{
			Message: "Email is currently locked for registration",
			Code:    authError.CodeEmailLocked,
			Error: pkgErrors.New(authError.CodeEmailLocked, "email is currently locked for registration").
				WithService(authError.ServiceName).
				WithDetail("email", email),
		}
	}
	return nil
}

func (s *authCache) ReleaseRegisterEmailLock(ctx context.Context, email string) *authError.AuthError {
	s.log.Info("Unlocking email after registration", logger.String("email", email))
	email = utils.NormalizeEmail(email)

	err := s.cache.ReleaseLock(ctx, RegisterEmailLockPrefix+email)
	if err != nil {
		s.log.Error("Failed to release email lock", logger.String("email", email), logger.Error(err))
		return &authError.AuthError{
			Message: "Failed to release email lock",
			Code:    authError.CodeCacheError,
			Error: pkgErrors.FromError(err, authError.CodeCacheError, "failed to release email lock").
				WithService(authError.ServiceName).
				WithDetail("email", email),
		}
	}
	return nil
}

func (s *authCache) CheckIPBlocked(ctx context.Context, ip string) *authError.AuthError {
	s.log.Info("Checking if IP is blocked", logger.String("ip", ip))
	blocked, err := s.cache.Get(ctx, IPBlockedPrefix+ip)
	if err != nil {
		s.log.Error("Failed to check IP block status", logger.String("ip", ip), logger.Error(err))
		return &authError.AuthError{
			Message: "Failed to check IP block status",
			Code:    authError.CodeCacheError,
			Error: pkgErrors.FromError(err, authError.CodeCacheError, "failed to check IP block status").
				WithService(authError.ServiceName).
				WithDetail("ip", ip),
		}
	}
	if blocked != nil {
		return &authError.AuthError{
			Message: "IP is blocked",
			Code:    authError.CodeIPBlocked,
			Error: pkgErrors.New(authError.CodeIPBlocked, "IP is blocked").
				WithService(authError.ServiceName).
				WithDetail("ip", ip),
		}
	}
	return nil
}

func (s *authCache) BlockIP(ctx context.Context, ip string, duration time.Duration) *authError.AuthError {
	s.log.Info("Blocking IP", logger.String("ip", ip), logger.Duration("duration", duration))
	err := s.cache.SetBool(ctx, IPBlockedPrefix+ip, true, duration)
	if err != nil {
		s.log.Error("Failed to block IP", logger.String("ip", ip), logger.Error(err))
		return &authError.AuthError{
			Message: "Failed to block IP",
			Code:    authError.CodeCacheError,
			Error: pkgErrors.FromError(err, authError.CodeCacheError, "failed to block IP").
				WithService(authError.ServiceName).
				WithDetail("ip", ip),
		}
	}
	return nil
}
