package ratelimiter

import (
	"context"
	msgError "echo-backend/services/message-service/internal/error"
	"fmt"
	"shared/pkg/cache"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"time"
)

var (
	RLSendPrefix       = "rl:send:"
	RLCreateConvPrefix = "rl:create_conv:"
	RLReactPrefix      = "rl:react:"
	RLTypingPrefix     = "rl:typing:"
)

const (
	RLSendTTL         = 60 * time.Second
	RLSendLimit       = int64(30)
	RLCreateConvTTL   = 3600 * time.Second
	RLCreateConvLimit = int64(10)
	RLReactTTL        = 60 * time.Second
	RLReactLimit      = int64(60)
	RLTypingTTL       = 3 * time.Second
)

type RateLimiter interface {
	CheckSendRateLimit(ctx context.Context, userID string) (bool, int64, error)
	CheckCreateConvRateLimit(ctx context.Context, userID string) (bool, int64, error)
	CheckReactRateLimit(ctx context.Context, userID string) (bool, int64, error)
	CheckTypingRateLimit(ctx context.Context, userID, conversationID string) (bool, error)
}

type rateLimiter struct {
	cache cache.Cache
	log   logger.Logger
}

func NewRateLimiter(cache cache.Cache, log logger.Logger) RateLimiter {
	return &rateLimiter{
		cache: cache,
		log:   log,
	}
}

func (m *rateLimiter) checkCounterRateLimit(ctx context.Context, key string, limit int64, window time.Duration) (bool, int64, error) {
	count, err := m.cache.Increment(ctx, key, 1)
	if err != nil {
		return false, 0, pkgErrors.FromError(err, msgError.CodeCacheError, "failed to increment rate limit counter").
			WithService(msgError.ServiceName).
			WithDetail("key", key)
	}
	if count == 1 {
		if expErr := m.cache.Expire(ctx, key, window); expErr != nil {
			m.log.Warn("Failed to set rate limit TTL", logger.String("key", key), logger.Error(expErr))
		}
	}
	if count > limit {
		return false, count, nil
	}
	return true, count, nil
}
func (m *rateLimiter) CheckSendRateLimit(ctx context.Context, userID string) (bool, int64, error) {
	m.log.Info("Checking send rate limit", logger.String("user_id", userID))
	return m.checkCounterRateLimit(ctx, RLSendPrefix+userID, RLSendLimit, RLSendTTL)
}

func (m *rateLimiter) CheckCreateConvRateLimit(ctx context.Context, userID string) (bool, int64, error) {
	m.log.Info("Checking create conv rate limit", logger.String("user_id", userID))
	return m.checkCounterRateLimit(ctx, RLCreateConvPrefix+userID, RLCreateConvLimit, RLCreateConvTTL)
}

func (m *rateLimiter) CheckReactRateLimit(ctx context.Context, userID string) (bool, int64, error) {
	m.log.Info("Checking react rate limit", logger.String("user_id", userID))
	return m.checkCounterRateLimit(ctx, RLReactPrefix+userID, RLReactLimit, RLReactTTL)
}

func (m *rateLimiter) CheckTypingRateLimit(ctx context.Context, userID, conversationID string) (bool, error) {
	m.log.Info("Checking typing rate limit", logger.String("user_id", userID), logger.String("conversation_id", conversationID))
	key := fmt.Sprintf("%s%s:%s", RLTypingPrefix, userID, conversationID)
	exists, err := m.cache.Exists(ctx, key)
	if err != nil {
		m.log.Error("Failed to check typing rate limit", logger.String("key", key), logger.Error(err))
		return false, pkgErrors.FromError(err, msgError.CodeCacheError, "failed to check typing rate limit").
			WithService(msgError.ServiceName).
			WithDetail("key", key)
	}
	if exists {
		return false, nil
	}
	if setErr := m.cache.SetBool(ctx, key, true, RLTypingTTL); setErr != nil {
		m.log.Warn("Failed to set typing throttle key", logger.String("key", key), logger.Error(setErr))
	}
	return true, nil
}
