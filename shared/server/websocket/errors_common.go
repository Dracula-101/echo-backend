package websocket

import "errors"

var (
	// ErrEngineAlreadyRunning is returned when engine is already running
	ErrEngineAlreadyRunning = errors.New("engine is already running")

	// ErrEngineNotRunning is returned when engine is not running
	ErrEngineNotRunning = errors.New("engine is not running")
)
