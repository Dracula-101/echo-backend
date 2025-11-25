package session

import "errors"

var (
	// ErrSessionNotFound is returned when session is not found
	ErrSessionNotFound = errors.New("session not found")

	// ErrSessionExpired is returned when session is expired
	ErrSessionExpired = errors.New("session expired")
)
