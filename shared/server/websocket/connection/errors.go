package connection

import "errors"

var (
	// ErrConnectionClosed is returned when the connection is closed
	ErrConnectionClosed = errors.New("connection closed")

	// ErrSendTimeout is returned when send times out
	ErrSendTimeout = errors.New("send timeout")

	// ErrMaxConnectionsReached is returned when max connections limit is reached
	ErrMaxConnectionsReached = errors.New("maximum connections reached")

	// ErrInvalidState is returned when operation is invalid for current state
	ErrInvalidState = errors.New("invalid state for operation")
)
