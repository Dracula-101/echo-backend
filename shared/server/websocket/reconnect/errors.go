package reconnect

import "errors"

var (
	// ErrMaxAttemptsReached is returned when max reconnection attempts reached
	ErrMaxAttemptsReached = errors.New("maximum reconnection attempts reached")
)
