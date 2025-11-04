package token

import "errors"

var (
	ErrInvalidConfig = errors.New("token: invalid manager config")
	ErrInvalidToken  = errors.New("token: invalid token")
	ErrExpiredToken  = errors.New("token: token expired")
	ErrKeyNotFound   = errors.New("token: signing key not found")
)
