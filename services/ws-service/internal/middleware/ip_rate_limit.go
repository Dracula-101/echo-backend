package middleware

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"time"

	"shared/pkg/logger"
	internalRedis "ws-service/internal/redis"

	"github.com/redis/go-redis/v9"
)

// IPRateLimitConfig parameters for the per-IP connection-attempt limiter.
type IPRateLimitConfig struct {
	MaxAttempts int           // requests allowed inside Window
	Window      time.Duration // sliding window length
}

// DefaultIPRateLimitConfig: 10 connection attempts per minute per IP.
// Tuned for reconnection storms — 10/min lets a misbehaving client recover
// without DoS'ing the auth path.
func DefaultIPRateLimitConfig() IPRateLimitConfig {
	return IPRateLimitConfig{
		MaxAttempts: 10,
		Window:      time.Minute,
	}
}

// IPRateLimit returns a Redis-backed sliding-window middleware that throttles
// HTTP upgrade attempts per client IP. Disabled if rdb is nil (single-instance
// dev mode). Trusts X-Real-IP / X-Forwarded-For when present.
func IPRateLimit(rdb *redis.Client, cfg IPRateLimitConfig, log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rdb == nil {
				next.ServeHTTP(w, r)
				return
			}

			ip := clientIP(r)
			if ip == "" {
				next.ServeHTTP(w, r)
				return
			}

			allowed, retryAfter, err := checkRateLimit(r.Context(), rdb, ip, cfg)
			if err != nil {
				log.Warn("ip rate-limit check failed; allowing through",
					logger.String("ip", ip),
					logger.Error(err),
				)
				next.ServeHTTP(w, r)
				return
			}
			if !allowed {
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				http.Error(w, "too many connection attempts", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// checkRateLimit increments the sliding-window counter via a sorted set keyed
// by IP. Returns (allowed, retryAfterSeconds, err).
func checkRateLimit(ctx context.Context, rdb *redis.Client, ip string, cfg IPRateLimitConfig) (bool, int, error) {
	key := internalRedis.RLAuthKey(ip)
	now := time.Now()
	cutoff := now.Add(-cfg.Window)

	pipe := rdb.Pipeline()
	pipe.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(cutoff.UnixNano(), 10))
	countCmd := pipe.ZCard(ctx, key)
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now.UnixNano()), Member: now.UnixNano()})
	pipe.Expire(ctx, key, cfg.Window+10*time.Second)
	if _, err := pipe.Exec(ctx); err != nil {
		return true, 0, err
	}

	if countCmd.Val() >= int64(cfg.MaxAttempts) {
		return false, int(cfg.Window.Seconds()), nil
	}
	return true, 0, nil
}

func clientIP(r *http.Request) string {
	if v := r.Header.Get("X-Real-IP"); v != "" {
		return v
	}
	if v := r.Header.Get("X-Forwarded-For"); v != "" {
		// First entry is the original client.
		for i := 0; i < len(v); i++ {
			if v[i] == ',' {
				return v[:i]
			}
		}
		return v
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
