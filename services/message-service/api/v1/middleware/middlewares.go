package middleware

import (
	ratelimiter "shared/pkg/ratelimiter"
	"context"
	"net/http"
	"shared/server/headers"
	"shared/server/middleware"
	"shared/server/request"
	req "shared/server/request"
	"shared/server/response"
	"strconv"
)

// middleware/ratelimit.go
type RateLimitCheck struct {
	Allow func(ctx context.Context, key string, limit int64) (bool, int64, error)
	Key   func(req request.RequestHandler) string
	Limit int64
}

func RateLimit(
	allow func(ctx context.Context, key string, limit int64) (bool, int64, error),
	key func(req request.RequestHandler) string,
	limit int64,
) middleware.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			requestHandler := req.NewHandler(r, w)
			allowed, retryAfter, err := allow(ctx, key(*requestHandler), limit)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			if !allowed {
				w.Header().Set(headers.RetryAfter, strconv.FormatInt(retryAfter, 10))
				w.WriteHeader(http.StatusTooManyRequests)
				response.TooManyRequestsError(ctx, r, w, "Rate limit exceeded", int(retryAfter))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func RateLimitMulti(limiter ratelimiter.RateLimiter, checks ...RateLimitCheck) middleware.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			for _, check := range checks {
				requestHandler := req.NewHandler(r, w)
				allowed, retryAfter, err := check.Allow(ctx, check.Key(*requestHandler), check.Limit)
				if err != nil {
					next.ServeHTTP(w, r)
					return
				}
				if !allowed {
					w.Header().Set(headers.RetryAfter, strconv.FormatInt(retryAfter, 10))
					w.WriteHeader(http.StatusTooManyRequests)
					response.TooManyRequestsError(ctx, r, w, "Rate limit exceeded", int(retryAfter))
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
