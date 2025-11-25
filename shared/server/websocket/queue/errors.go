package queue

import "errors"

var (
	// ErrQueueEmpty is returned when queue is empty
	ErrQueueEmpty = errors.New("queue is empty")

	// ErrQueueFull is returned when queue is full
	ErrQueueFull = errors.New("queue is full")
)
