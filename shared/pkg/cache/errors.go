package cache

import "errors"

var (
	ErrNotFound     = errors.New("cache: key not found")
	ErrNotSupported = errors.New("cache: operation not supported")
)
