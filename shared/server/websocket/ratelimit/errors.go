package ratelimit

import "errors"

var (
	// ErrRateLimited is returned when rate limit is exceeded
	ErrRateLimited = errors.New("rate limit exceeded")

	// ErrWaitNotSupported is returned when Wait is not supported
	ErrWaitNotSupported = errors.New("wait not supported for this limiter")
)
