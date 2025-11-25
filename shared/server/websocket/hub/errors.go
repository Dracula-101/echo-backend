package hub

import "errors"

var (
	ErrClientNotFound = errors.New("hub: client not found")
	ErrHubClosed      = errors.New("hub: hub closed")
)
