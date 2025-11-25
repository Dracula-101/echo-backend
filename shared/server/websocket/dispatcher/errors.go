package dispatcher

import "errors"

var (
	// ErrQueueFull is returned when the dispatch queue is full
	ErrQueueFull = errors.New("dispatch queue is full")

	// ErrDispatcherStopped is returned when dispatcher is stopped
	ErrDispatcherStopped = errors.New("dispatcher is stopped")
)
