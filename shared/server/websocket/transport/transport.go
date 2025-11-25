package transport

import (
	"context"
	"io"
)

// Transport is an interface for WebSocket transport layer
type Transport interface {
	// Read reads data from the transport
	Read(ctx context.Context) ([]byte, error)

	// Write writes data to the transport
	Write(ctx context.Context, data []byte) error

	// Close closes the transport
	Close() error

	// LocalAddr returns the local address
	LocalAddr() string

	// RemoteAddr returns the remote address
	RemoteAddr() string
}

// Conn represents a transport connection
type Conn interface {
	Transport
	io.Closer
}
