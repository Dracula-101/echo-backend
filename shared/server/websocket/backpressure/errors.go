package backpressure

import "errors"

var (
	// ErrBackpressureApplied is returned when backpressure is applied
	ErrBackpressureApplied = errors.New("backpressure applied")

	// ErrMessageDropped is returned when message is dropped
	ErrMessageDropped = errors.New("message dropped due to backpressure")

	// ErrDropOldest signals to drop oldest message
	ErrDropOldest = errors.New("drop oldest message")
)
