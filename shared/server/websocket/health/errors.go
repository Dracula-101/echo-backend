package health

import "errors"

var (
	// ErrCheckNotFound is returned when health check is not found
	ErrCheckNotFound = errors.New("health check not found")
)
