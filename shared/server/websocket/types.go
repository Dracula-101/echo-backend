package websocket

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// MessageHandler is a function that processes incoming WebSocket messages
type MessageHandler func(client *Client, message []byte)

// ConnectHandler is called when a client connects
type ConnectHandler func(client *Client)

// DisconnectHandler is called when a client disconnects
type DisconnectHandler func(client *Client)

// ErrorHandler is called when an error occurs
type ErrorHandler func(client *Client, err error)

// MessageValidator validates messages before processing
type MessageValidator interface {
	Validate(message []byte) error
}

// ClientMetadata holds metadata about a WebSocket client
type ClientMetadata struct {
	UserID      uuid.UUID         `json:"user_id"`
	DeviceID    string            `json:"device_id"`
	IPAddress   string            `json:"ip_address"`
	UserAgent   string            `json:"user_agent"`
	Platform    string            `json:"platform"`
	AppVersion  string            `json:"app_version"`
	DeviceName  string            `json:"device_name"`
	DeviceType  string            `json:"device_type"`
	ConnectedAt time.Time         `json:"connected_at"`
	LastPingAt  time.Time         `json:"last_ping_at"`
	CustomData  map[string]string `json:"custom_data,omitempty"`
}

// ClientState represents the state of a WebSocket client
type ClientState int

const (
	// StateConnecting is the initial state when client is connecting
	StateConnecting ClientState = iota
	// StateConnected is when client is fully connected
	StateConnected
	// StateDisconnecting is when client is in the process of disconnecting
	StateDisconnecting
	// StateDisconnected is when client has disconnected
	StateDisconnected
	// StateReconnecting is when client is attempting to reconnect
	StateReconnecting
)

// String returns the string representation of ClientState
func (s ClientState) String() string {
	switch s {
	case StateConnecting:
		return "connecting"
	case StateConnected:
		return "connected"
	case StateDisconnecting:
		return "disconnecting"
	case StateDisconnected:
		return "disconnected"
	case StateReconnecting:
		return "reconnecting"
	default:
		return "unknown"
	}
}

// HubStats represents statistics about the hub
type HubStats struct {
	TotalClients      int            `json:"total_clients"`
	TotalConnections  int            `json:"total_connections"`
	OnlineUsers       int            `json:"online_users"`
	MessagesSent      int64          `json:"messages_sent"`
	MessagesReceived  int64          `json:"messages_received"`
	BytesSent         int64          `json:"bytes_sent"`
	BytesReceived     int64          `json:"bytes_received"`
	Uptime            time.Duration  `json:"uptime"`
	UserDeviceCount   map[uuid.UUID]int `json:"user_device_count,omitempty"`
}

// BroadcastMessage represents a message to be broadcast
type BroadcastMessage struct {
	Data      interface{}
	Target    BroadcastTarget
	Exclude   []string // Client IDs to exclude
	UserIDs   []uuid.UUID
	Timestamp time.Time
}

// BroadcastTarget defines the target for broadcasting
type BroadcastTarget int

const (
	// BroadcastAll sends to all connected clients
	BroadcastAll BroadcastTarget = iota
	// BroadcastUser sends to all devices of specific users
	BroadcastUser
	// BroadcastExcept sends to all except specified clients
	BroadcastExcept
)

// Config holds WebSocket configuration
type Config struct {
	// Connection timeouts
	WriteWait          time.Duration `json:"write_wait"`
	PongWait           time.Duration `json:"pong_wait"`
	PingPeriod         time.Duration `json:"ping_period"`
	CloseGracePeriod   time.Duration `json:"close_grace_period"`
	HandshakeTimeout   time.Duration `json:"handshake_timeout"`

	// Buffer sizes
	ReadBufferSize     int           `json:"read_buffer_size"`
	WriteBufferSize    int           `json:"write_buffer_size"`
	MaxMessageSize     int64         `json:"max_message_size"`
	ClientBufferSize   int           `json:"client_buffer_size"`

	// Cleanup and maintenance
	CleanupInterval           time.Duration `json:"cleanup_interval"`
	StaleConnectionTimeout    time.Duration `json:"stale_connection_timeout"`
	MaxConnectionsPerUser     int           `json:"max_connections_per_user"`
	MaxReconnectAttempts      int           `json:"max_reconnect_attempts"`
	ReconnectBackoff          time.Duration `json:"reconnect_backoff"`

	// Hub channels
	RegisterBuffer     int           `json:"register_buffer"`
	UnregisterBuffer   int           `json:"unregister_buffer"`
	BroadcastBuffer    int           `json:"broadcast_buffer"`

	// Security
	CheckOrigin        bool          `json:"check_origin"`
	AllowedOrigins     []string      `json:"allowed_origins"`
	EnableCompression  bool          `json:"enable_compression"`
	CompressionLevel   int           `json:"compression_level"`

	// Rate limiting
	MaxMessagesPerSecond int         `json:"max_messages_per_second"`
	BurstSize            int         `json:"burst_size"`

	// Monitoring
	EnableMetrics      bool          `json:"enable_metrics"`
	MetricsInterval    time.Duration `json:"metrics_interval"`
}

// DefaultConfig returns default WebSocket configuration
func DefaultConfig() *Config {
	return &Config{
		WriteWait:                 10 * time.Second,
		PongWait:                  60 * time.Second,
		PingPeriod:                54 * time.Second,
		CloseGracePeriod:          5 * time.Second,
		HandshakeTimeout:          10 * time.Second,
		ReadBufferSize:            1024,
		WriteBufferSize:           1024,
		MaxMessageSize:            10 * 1024 * 1024, // 10MB
		ClientBufferSize:          256,
		CleanupInterval:           30 * time.Second,
		StaleConnectionTimeout:    90 * time.Second,
		MaxConnectionsPerUser:     5,
		MaxReconnectAttempts:      3,
		ReconnectBackoff:          time.Second,
		RegisterBuffer:            256,
		UnregisterBuffer:          256,
		BroadcastBuffer:           1024,
		CheckOrigin:               false,
		AllowedOrigins:            []string{},
		EnableCompression:         false,
		CompressionLevel:          -1,
		MaxMessagesPerSecond:      100,
		BurstSize:                 10,
		EnableMetrics:             true,
		MetricsInterval:           time.Minute,
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.WriteWait <= 0 {
		return ErrInvalidConfig("write_wait must be positive")
	}
	if c.PongWait <= 0 {
		return ErrInvalidConfig("pong_wait must be positive")
	}
	if c.PingPeriod <= 0 || c.PingPeriod >= c.PongWait {
		return ErrInvalidConfig("ping_period must be positive and less than pong_wait")
	}
	if c.MaxMessageSize <= 0 {
		return ErrInvalidConfig("max_message_size must be positive")
	}
	if c.ClientBufferSize <= 0 {
		return ErrInvalidConfig("client_buffer_size must be positive")
	}
	return nil
}

// RateLimiter interface for rate limiting
type RateLimiter interface {
	Allow(clientID string) bool
	Reset(clientID string)
}

// ClientRegistry manages client lookups
type ClientRegistry interface {
	Register(client *Client)
	Unregister(client *Client)
	GetByID(clientID string) (*Client, bool)
	GetByUserID(userID uuid.UUID) []*Client
	GetAll() []*Client
	Count() int
}

// MessageQueue interface for message queuing
type MessageQueue interface {
	Enqueue(clientID string, message interface{}) error
	Dequeue(clientID string) (interface{}, error)
	Size(clientID string) int
	Clear(clientID string)
}

// Mutex-protected map for safe concurrent access
type safeMap struct {
	mu sync.RWMutex
	m  map[string]interface{}
}

func newSafeMap() *safeMap {
	return &safeMap{
		m: make(map[string]interface{}),
	}
}

func (sm *safeMap) Set(key string, value interface{}) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.m[key] = value
}

func (sm *safeMap) Get(key string) (interface{}, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	val, ok := sm.m[key]
	return val, ok
}

func (sm *safeMap) Delete(key string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.m, key)
}

func (sm *safeMap) Len() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.m)
}

// Context keys for WebSocket context values
type contextKey string

const (
	// ContextKeyClient is the key for storing client in context
	ContextKeyClient contextKey = "websocket_client"
	// ContextKeyUserID is the key for storing user ID in context
	ContextKeyUserID contextKey = "user_id"
	// ContextKeyDeviceID is the key for storing device ID in context
	ContextKeyDeviceID contextKey = "device_id"
	// ContextKeyRequestID is the key for storing request ID in context
	ContextKeyRequestID contextKey = "request_id"
)

// ClientFromContext retrieves client from context
func ClientFromContext(ctx context.Context) (*Client, bool) {
	client, ok := ctx.Value(ContextKeyClient).(*Client)
	return client, ok
}

// ContextWithClient returns a new context with client
func ContextWithClient(ctx context.Context, client *Client) context.Context {
	return context.WithValue(ctx, ContextKeyClient, client)
}

// Conn is an interface wrapper around gorilla/websocket.Conn for testing
type Conn interface {
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, data []byte) error
	WriteControl(messageType int, data []byte, deadline time.Time) error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
	SetReadLimit(limit int64)
	SetPongHandler(h func(appData string) error)
	SetPingHandler(h func(appData string) error)
	SetCloseHandler(h func(code int, text string) error)
	Close() error
	LocalAddr() string
	RemoteAddr() string
	Subprotocol() string
}

// gorilla websocket connection wrapper
type wsConn struct {
	*websocket.Conn
}

func (w *wsConn) LocalAddr() string {
	return w.Conn.LocalAddr().String()
}

func (w *wsConn) RemoteAddr() string {
	return w.Conn.RemoteAddr().String()
}

// WrapConn wraps a gorilla websocket connection
func WrapConn(conn *websocket.Conn) Conn {
	return &wsConn{Conn: conn}
}
