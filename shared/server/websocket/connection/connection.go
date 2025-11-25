package connection

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	"shared/pkg/logger"
	"shared/server/websocket/state"

	"github.com/gorilla/websocket"
)

// Connection represents a WebSocket connection
type Connection struct {
	id       string
	conn     *websocket.Conn
	connMu   sync.RWMutex

	// State management
	stateMachine *state.Machine

	// Communication channels
	send     chan []byte
	sendDone chan struct{}

	// Context
	ctx    context.Context
	cancel context.CancelFunc

	// Metadata
	metadata  map[string]interface{}
	metadataMu sync.RWMutex

	// Timing
	createdAt    time.Time
	lastActivity atomic.Value // time.Time
	lastPing     atomic.Value // time.Time

	// Statistics
	messagesSent     atomic.Int64
	messagesReceived atomic.Int64
	bytesSent        atomic.Int64
	bytesReceived    atomic.Int64

	// Configuration
	config *Config

	// Logger
	log logger.Logger

	// Write mutex
	writeMu sync.Mutex
}

// New creates a new WebSocket connection
func New(id string, conn *websocket.Conn, config *Config, log logger.Logger) *Connection {
	ctx, cancel := context.WithCancel(context.Background())

	c := &Connection{
		id:           id,
		conn:         conn,
		stateMachine: state.NewMachine(state.StateConnecting),
		send:         make(chan []byte, config.SendBufferSize),
		sendDone:     make(chan struct{}),
		ctx:          ctx,
		cancel:       cancel,
		metadata:     make(map[string]interface{}),
		createdAt:    time.Now(),
		config:       config,
		log:          log,
	}

	c.lastActivity.Store(time.Now())
	c.lastPing.Store(time.Now())

	return c
}

// ID returns the connection ID
func (c *Connection) ID() string {
	return c.id
}

// State returns the current state
func (c *Connection) State() state.State {
	return c.stateMachine.Current()
}

// TransitionTo transitions to a new state
func (c *Connection) TransitionTo(newState state.State) error {
	return c.stateMachine.Transition(newState)
}

// Send queues a message for sending
func (c *Connection) Send(data []byte) error {
	if !c.IsConnected() {
		return ErrConnectionClosed
	}

	select {
	case c.send <- data:
		return nil
	case <-time.After(c.config.WriteTimeout):
		return ErrSendTimeout
	case <-c.ctx.Done():
		return ErrConnectionClosed
	}
}

// SendJSON sends a JSON-encoded message
func (c *Connection) SendJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.Send(data)
}

// IsConnected returns true if the connection is active
func (c *Connection) IsConnected() bool {
	s := c.State()
	return s == state.StateConnected || s == state.StateConnecting
}

// IsStale returns true if the connection is stale
func (c *Connection) IsStale() bool {
	lastAct := c.lastActivity.Load().(time.Time)
	return time.Since(lastAct) > c.config.StaleTimeout
}

// LastActivity returns the last activity time
func (c *Connection) LastActivity() time.Time {
	return c.lastActivity.Load().(time.Time)
}

// UpdateActivity updates the last activity timestamp
func (c *Connection) UpdateActivity() {
	c.lastActivity.Store(time.Now())
}

// SetMetadata sets metadata for the connection
func (c *Connection) SetMetadata(key string, value interface{}) {
	c.metadataMu.Lock()
	defer c.metadataMu.Unlock()
	c.metadata[key] = value
}

// GetMetadata retrieves metadata
func (c *Connection) GetMetadata(key string) (interface{}, bool) {
	c.metadataMu.RLock()
	defer c.metadataMu.RUnlock()
	val, ok := c.metadata[key]
	return val, ok
}

// Stats returns connection statistics
func (c *Connection) Stats() Stats {
	return Stats{
		ID:               c.id,
		State:            c.State().String(),
		CreatedAt:        c.createdAt,
		LastActivity:     c.LastActivity(),
		MessagesSent:     c.messagesSent.Load(),
		MessagesReceived: c.messagesReceived.Load(),
		BytesSent:        c.bytesSent.Load(),
		BytesReceived:    c.bytesReceived.Load(),
		Uptime:           time.Since(c.createdAt),
	}
}

// Context returns the connection context
func (c *Connection) Context() context.Context {
	return c.ctx
}

// Close closes the connection gracefully
func (c *Connection) Close() error {
	if err := c.TransitionTo(state.StateDisconnecting); err != nil {
		return err
	}

	c.cancel()

	// Close send channel
	select {
	case <-c.sendDone:
	default:
		close(c.send)
		close(c.sendDone)
	}

	// Close websocket connection
	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.conn != nil {
		c.conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(time.Second),
		)
		err := c.conn.Close()
		c.conn = nil
		c.TransitionTo(state.StateDisconnected)
		return err
	}

	c.TransitionTo(state.StateDisconnected)
	return nil
}

// Conn returns the underlying websocket connection (for internal use)
func (c *Connection) Conn() *websocket.Conn {
	c.connMu.RLock()
	defer c.connMu.RUnlock()
	return c.conn
}

// IncrementMessagesSent increments sent message counter
func (c *Connection) IncrementMessagesSent() {
	c.messagesSent.Add(1)
}

// IncrementMessagesReceived increments received message counter
func (c *Connection) IncrementMessagesReceived() {
	c.messagesReceived.Add(1)
}

// AddBytesSent adds to bytes sent counter
func (c *Connection) AddBytesSent(n int64) {
	c.bytesSent.Add(n)
}

// AddBytesReceived adds to bytes received counter
func (c *Connection) AddBytesReceived(n int64) {
	c.bytesReceived.Add(n)
}

// SendChan returns the send channel for reading messages
func (c *Connection) SendChan() <-chan []byte {
	return c.send
}
