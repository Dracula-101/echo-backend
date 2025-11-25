package buffer

import "errors"

var (
	// ErrBufferFull is returned when buffer is full
	ErrBufferFull = errors.New("buffer is full")

	// ErrBufferEmpty is returned when buffer is empty
	ErrBufferEmpty = errors.New("buffer is empty")
)
