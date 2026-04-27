package ratelimiter

import (
	"context"
	msgError "echo-backend/services/message-service/internal/error"
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
	RLSearchPrefix     = "rl:search:"
	RLReportPrefix     = "rl:report:"
)

const (
	RLSendTTL         = 60 * time.Second
	RLSendLimit       = int64(60)
	RLCreateConvTTL   = 3600 * time.Second
	RLCreateConvLimit = int64(10)
	RLReactTTL        = 60 * time.Second
	RLReactLimit      = int64(100)
	RLTypingTTL       = 3 * time.Second
	RLSearchTTL       = 60 * time.Second
	RLSearchLimit     = int64(10)
	RLReportTTL       = 24 * time.Hour
	RLReportLimit     = int64(5)
)

type RateLimiter interface {
	CheckSendRateLimit(ctx context.Context, userID string) (bool, int64, error)
	CheckCreateConvRateLimit(ctx context.Context, userID string) (bool, int64, error)
	CheckReactRateLimit(ctx context.Context, userID string) (bool, int64, error)
	CheckTypingRateLimit(ctx context.Context, userID, conversationID string) (bool, error)

	CheckReportRateLimit(ctx context.Context, userID string) (bool, int64, error)
	CheckSearchRateLimit(ctx context.Context, userID, conversationID string) (bool, int64, error)
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
	key := RLTypingPrefix + userID + ":" + conversationID
	allowed, _, err := m.checkCounterRateLimit(ctx, key, 1, RLTypingTTL)
	return allowed, err
}

func (m *rateLimiter) CheckReportRateLimit(ctx context.Context, userID string) (bool, int64, error) {
	m.log.Info("Checking report rate limit", logger.String("user_id", userID))
	return m.checkCounterRateLimit(ctx, RLReportPrefix+userID, RLReportLimit, RLReportTTL)
}

func (m *rateLimiter) CheckSearchRateLimit(ctx context.Context, userID, conversationID string) (bool, int64, error) {
	m.log.Info("Checking search rate limit", logger.String("user_id", userID), logger.String("conversation_id", conversationID))
	key := RLSearchPrefix + userID + ":" + conversationID
	return m.checkCounterRateLimit(ctx, key, RLSearchLimit, RLSearchTTL)
}
