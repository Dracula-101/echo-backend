package cache

import "errors"

var (
	ErrNotFound        = errors.New("cache: key not found")
	ErrNotSupported    = errors.New("cache: operation not supported")
	ErrConnection      = errors.New("cache: connection error")
	ErrTimeout         = errors.New("cache: operation timeout")
	ErrInvalidData     = errors.New("cache: invalid data")
	ErrCacheError      = errors.New("cache: general error")
	ErrSerialization   = errors.New("cache: serialization error")
	ErrDeserialization = errors.New("cache: deserialization error")
	ErrUnknown         = errors.New("cache: unknown error")
)
