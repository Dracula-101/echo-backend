package hashing

import "errors"

var (
	ErrInvalidConfig    = errors.New("hashing: invalid config")
	ErrUnknownAlgorithm = errors.New("hashing: unknown algorithm")
	ErrMalformedHash    = errors.New("hashing: malformed hash")
)
