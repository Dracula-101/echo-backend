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
	CheckSendRateLimit(ctx context.Context, userID string) *msgError.MsgError
	CheckCreateConvRateLimit(ctx context.Context, userID string) *msgError.MsgError
	CheckReactRateLimit(ctx context.Context, userID string) *msgError.MsgError
	CheckTypingRateLimit(ctx context.Context, userID, conversationID string) *msgError.MsgError
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

func (m *rateLimiter) checkCounterRateLimit(ctx context.Context, key string, limit int64, window time.Duration) *msgError.MsgError {
	count, err := m.cache.Increment(ctx, key, 1)
	if err != nil {
		return &msgError.MsgError{
			Message: "Failed to check rate limit",
			Code:    msgError.CodeCacheError,
			Error: pkgErrors.FromError(err, msgError.CodeCacheError, "failed to increment rate limit counter").
				WithService(msgError.ServiceName).
				WithDetail("key", key),
		}
	}
	if count == 1 {
		if expErr := m.cache.Expire(ctx, key, window); expErr != nil {
			m.log.Warn("Failed to set rate limit TTL", logger.String("key", key), logger.Error(expErr))
		}
	}
	if count > limit {
		return &msgError.MsgError{
			Message: "Rate limit exceeded",
			Code:    msgError.CodeRateLimited,
			Error: pkgErrors.New(msgError.CodeRateLimited, "rate limit exceeded").
				WithService(msgError.ServiceName).
				WithDetail("key", key),
		}
	}
	return nil
}
func (m *rateLimiter) CheckSendRateLimit(ctx context.Context, userID string) *msgError.MsgError {
	m.log.Info("Checking send rate limit", logger.String("user_id", userID))
	return m.checkCounterRateLimit(ctx, RLSendPrefix+userID, RLSendLimit, RLSendTTL)
}

func (m *rateLimiter) CheckCreateConvRateLimit(ctx context.Context, userID string) *msgError.MsgError {
	m.log.Info("Checking create conv rate limit", logger.String("user_id", userID))
	return m.checkCounterRateLimit(ctx, RLCreateConvPrefix+userID, RLCreateConvLimit, RLCreateConvTTL)
}

func (m *rateLimiter) CheckReactRateLimit(ctx context.Context, userID string) *msgError.MsgError {
	m.log.Info("Checking react rate limit", logger.String("user_id", userID))
	return m.checkCounterRateLimit(ctx, RLReactPrefix+userID, RLReactLimit, RLReactTTL)
}

func (m *rateLimiter) CheckTypingRateLimit(ctx context.Context, userID, conversationID string) *msgError.MsgError {
	m.log.Info("Checking typing rate limit", logger.String("user_id", userID), logger.String("conversation_id", conversationID))
	key := fmt.Sprintf("%s%s:%s", RLTypingPrefix, userID, conversationID)
	exists, err := m.cache.Exists(ctx, key)
	if err != nil {
		m.log.Error("Failed to check typing rate limit", logger.String("key", key), logger.Error(err))
		return &msgError.MsgError{
			Message: "Failed to check typing rate limit",
			Code:    msgError.CodeCacheError,
			Error: pkgErrors.FromError(err, msgError.CodeCacheError, "failed to check typing throttle key").
				WithService(msgError.ServiceName).
				WithDetail("user_id", userID).
				WithDetail("conversation_id", conversationID),
		}
	}
	if exists {
		return &msgError.MsgError{
			Message: "Typing update throttled",
			Code:    msgError.CodeRateLimited,
			Error: pkgErrors.New(msgError.CodeRateLimited, "typing update throttled").
				WithService(msgError.ServiceName).
				WithDetail("user_id", userID).
				WithDetail("conversation_id", conversationID),
		}
	}
	if setErr := m.cache.SetBool(ctx, key, true, RLTypingTTL); setErr != nil {
		m.log.Warn("Failed to set typing throttle key", logger.String("key", key), logger.Error(setErr))
	}
	return nil
}
