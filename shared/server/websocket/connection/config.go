package connection

import "time"

// Config holds connection configuration
type Config struct {
	// Timeouts
	WriteTimeout  time.Duration
	ReadTimeout   time.Duration
	PingInterval  time.Duration
	PongTimeout   time.Duration
	StaleTimeout  time.Duration
	HandshakeTimeout time.Duration

	// Buffer sizes
	SendBufferSize  int
	ReadBufferSize  int
	WriteBufferSize int

	// Message limits
	MaxMessageSize int64

	// Compression
	EnableCompression bool
	CompressionLevel  int
}

// DefaultConfig returns default connection configuration
func DefaultConfig() *Config {
	return &Config{
		WriteTimeout:      10 * time.Second,
		ReadTimeout:       60 * time.Second,
		PingInterval:      54 * time.Second,
		PongTimeout:       60 * time.Second,
		StaleTimeout:      90 * time.Second,
		HandshakeTimeout:  10 * time.Second,
		SendBufferSize:    256,
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		MaxMessageSize:    10 * 1024 * 1024, // 10MB
		EnableCompression: false,
		CompressionLevel:  -1,
	}
}
