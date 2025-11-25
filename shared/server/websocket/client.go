package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"shared/pkg/logger"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Client represents a WebSocket client connection
type Client struct {
	// Identification
	ID       string
	UserID   uuid.UUID
	DeviceID string

	// Connection
	conn     Conn
	connMu   sync.RWMutex
	state    atomic.Value // ClientState
	hub      *Hub

	// Metadata
	metadata ClientMetadata

	// Communication channels
	send     chan interface{}
	sendDone chan struct{}

	// Message handling
	messageHandler MessageHandler

	// Context and cancellation
	ctx    context.Context
	cancel context.CancelFunc

	// Timing and health
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

	// Locks
	writeMu sync.Mutex

	// Lifecycle hooks
	onDisconnect DisconnectHandler
	onError      ErrorHandler

	// Custom data storage
	data *safeMap

	// Reconnection
	reconnectAttempts atomic.Int32
	lastReconnect     atomic.Value // time.Time
}

// NewClient creates a new WebSocket client
func NewClient(
	userID uuid.UUID,
	deviceID string,
	conn Conn,
	hub *Hub,
	metadata ClientMetadata,
	config *Config,
	log logger.Logger,
) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		ID:             generateClientID(),
		UserID:         userID,
		DeviceID:       deviceID,
		conn:           conn,
		hub:            hub,
		metadata:       metadata,
		send:           make(chan interface{}, config.ClientBufferSize),
		sendDone:       make(chan struct{}),
		ctx:            ctx,
		cancel:         cancel,
		createdAt:      time.Now(),
		config:         config,
		log:            log,
		data:           newSafeMap(),
	}

	client.state.Store(StateConnecting)
	client.lastActivity.Store(time.Now())
	client.lastPing.Store(time.Now())

	return client
}

// SetMessageHandler sets the message handler for this client
func (c *Client) SetMessageHandler(handler MessageHandler) {
	c.messageHandler = handler
}

// SetOnDisconnect sets the disconnect handler
func (c *Client) SetOnDisconnect(handler DisconnectHandler) {
	c.onDisconnect = handler
}

// SetOnError sets the error handler
func (c *Client) SetOnError(handler ErrorHandler) {
	c.onError = handler
}

// State returns the current state of the client
func (c *Client) State() ClientState {
	return c.state.Load().(ClientState)
}

// setState sets the state of the client
func (c *Client) setState(state ClientState) {
	c.state.Store(state)
	c.log.Debug("Client state changed",
		logger.String("client_id", c.ID),
		logger.String("user_id", c.UserID.String()),
		logger.String("state", state.String()),
	)
}

// IsConnected returns true if client is connected
func (c *Client) IsConnected() bool {
	state := c.State()
	return state == StateConnected || state == StateConnecting
}

// ReadPump pumps messages from the WebSocket connection to the hub
func (c *Client) ReadPump() {
	c.log.Info("ReadPump started",
		logger.String("client_id", c.ID),
	)
	
	defer func() {
		c.log.Info("ReadPump stopping",
			logger.String("client_id", c.ID),
		)
		c.handleDisconnect()
	}()

	c.connMu.RLock()
	conn := c.conn
	c.connMu.RUnlock()

	if conn == nil {
		c.log.Error("ReadPump called with nil connection",
			logger.String("client_id", c.ID),
		)
		return
	}

	conn.SetReadLimit(c.config.MaxMessageSize)
	if err := conn.SetReadDeadline(time.Now().Add(c.config.PongWait)); err != nil {
		c.handleError(err, "failed to set read deadline")
		return
	}

	conn.SetPongHandler(func(string) error {
		c.lastPing.Store(time.Now())
		c.updateActivity()
		return conn.SetReadDeadline(time.Now().Add(c.config.PongWait))
	})

	conn.SetPingHandler(func(appData string) error {
		c.lastPing.Store(time.Now())
		c.updateActivity()
		err := c.writeControl(websocket.PongMessage, []byte{}, time.Now().Add(c.config.WriteWait))
		if err != nil {
			c.log.Error("Failed to send pong",
				logger.String("client_id", c.ID),
				logger.Error(err),
			)
		}
		return conn.SetReadDeadline(time.Now().Add(c.config.PongWait))
	})

	conn.SetCloseHandler(func(code int, text string) error {
		c.log.Info("Received close message",
			logger.String("client_id", c.ID),
			logger.Int("code", code),
			logger.String("text", text),
		)
		c.setState(StateDisconnecting)
		return nil
	})

	c.setState(StateConnected)

	for {
		select {
		case <-c.ctx.Done():
			c.log.Debug("ReadPump context cancelled",
				logger.String("client_id", c.ID),
			)
			return
		default:
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err,
					websocket.CloseGoingAway,
					websocket.CloseAbnormalClosure,
					websocket.CloseNormalClosure) {
					c.log.Error("Unexpected close error",
						logger.String("client_id", c.ID),
						logger.Error(err),
					)
				}
				c.handleError(err, "read message failed")
				return
			}

			// Log every incoming message
			c.log.Info("Received WebSocket message",
				logger.String("client_id", c.ID),
				logger.String("user_id", c.UserID.String()),
				logger.Int("message_type", messageType),
				logger.Int("message_size", len(message)),
				logger.String("raw_message", string(message)),
			)

			if messageType != websocket.TextMessage && messageType != websocket.BinaryMessage {
				c.log.Warn("Received non-text/binary message",
					logger.String("client_id", c.ID),
					logger.Int("message_type", messageType),
				)
				continue
			}

			c.updateActivity()
			c.messagesReceived.Add(1)
			c.bytesReceived.Add(int64(len(message)))

			// Process message
			if c.messageHandler != nil {
				c.log.Debug("Calling message handler",
					logger.String("client_id", c.ID),
				)
				c.messageHandler(c, message)
			} else {
				c.log.Warn("No message handler set",
					logger.String("client_id", c.ID),
				)
			}
		}
	}
}

// WritePump pumps messages from the hub to the WebSocket connection
func (c *Client) WritePump() {
	c.log.Info("WritePump started",
		logger.String("client_id", c.ID),
	)
	
	ticker := time.NewTicker(c.config.PingPeriod)
	defer func() {
		ticker.Stop()
		c.log.Info("WritePump stopping",
			logger.String("client_id", c.ID),
		)
		c.handleDisconnect()
	}()

	for {
		select {
		case <-c.ctx.Done():
			c.log.Debug("WritePump context cancelled",
				logger.String("client_id", c.ID),
			)
			return

		case message, ok := <-c.send:
			if !ok {
				// Channel closed
				c.writeClose()
				return
			}

			c.log.Info("WritePump sending message",
				logger.String("client_id", c.ID),
				logger.String("message_type", fmt.Sprintf("%T", message)),
			)

			if err := c.writeMessage(message); err != nil {
				c.log.Error("Write message failed",
					logger.String("client_id", c.ID),
					logger.Error(err),
				)
				c.handleError(err, "write message failed")
				return
			}

			c.log.Info("Message sent successfully",
				logger.String("client_id", c.ID),
			)

		case <-ticker.C:
			if err := c.writePing(); err != nil {
				c.handleError(err, "write ping failed")
				return
			}
		}
	}
}

// SendMessage sends a message to the client
func (c *Client) SendMessage(message interface{}) error {
	c.log.Info("SendMessage called",
		logger.String("client_id", c.ID),
		logger.String("message_type", fmt.Sprintf("%T", message)),
		logger.Bool("is_connected", c.IsConnected()),
	)

	if !c.IsConnected() {
		c.log.Warn("Client not connected, cannot send message",
			logger.String("client_id", c.ID),
		)
		return ErrClientDisconnected
	}

	select {
	case c.send <- message:
		c.log.Info("Message queued successfully",
			logger.String("client_id", c.ID),
		)
		return nil
	case <-time.After(c.config.WriteWait):
		c.log.Error("Message timeout",
			logger.String("client_id", c.ID),
		)
		return ErrMessageTimeout
	case <-c.ctx.Done():
		c.log.Warn("Context cancelled while sending message",
			logger.String("client_id", c.ID),
		)
		return ErrConnectionClosed
	}
}

// SendMessageSync sends a message synchronously
func (c *Client) SendMessageSync(message interface{}) error {
	if !c.IsConnected() {
		return ErrClientDisconnected
	}
	return c.writeMessage(message)
}

// writeMessage writes a message to the WebSocket connection
func (c *Client) writeMessage(message interface{}) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	c.connMu.RLock()
	conn := c.conn
	c.connMu.RUnlock()

	if conn == nil {
		return ErrInvalidConnection
	}

	if err := conn.SetWriteDeadline(time.Now().Add(c.config.WriteWait)); err != nil {
		return NewConnectionError(c.ID, err, "failed to set write deadline")
	}

	var data []byte
	var err error

	switch v := message.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		data, err = json.Marshal(message)
		if err != nil {
			return NewMessageError(c.ID, "unknown", err, "failed to marshal message")
		}
	}

	if int64(len(data)) > c.config.MaxMessageSize {
		return ErrMessageTooLarge
	}

	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return NewConnectionError(c.ID, err, "failed to write message")
	}

	c.messagesSent.Add(1)
	c.bytesSent.Add(int64(len(data)))
	c.updateActivity()

	return nil
}

// writePing writes a ping message
func (c *Client) writePing() error {
	return c.writeControl(websocket.PingMessage, []byte{}, time.Now().Add(c.config.WriteWait))
}

// writeClose writes a close message
func (c *Client) writeClose() {
	c.writeControl(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		time.Now().Add(c.config.WriteWait))
}

// writeControl writes a control message
func (c *Client) writeControl(messageType int, data []byte, deadline time.Time) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	c.connMu.RLock()
	conn := c.conn
	c.connMu.RUnlock()

	if conn == nil {
		return ErrInvalidConnection
	}

	return conn.WriteControl(messageType, data, deadline)
}

// Close closes the client connection gracefully
func (c *Client) Close() error {
	if c.State() == StateDisconnected {
		return nil
	}

	c.setState(StateDisconnecting)

	// Cancel context
	c.cancel()

	// Send close message
	c.writeClose()

	// Wait a bit for graceful close
	time.Sleep(c.config.CloseGracePeriod)

	// Close connection
	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		c.setState(StateDisconnected)
		return err
	}

	return nil
}

// handleDisconnect handles client disconnection
func (c *Client) handleDisconnect() {
	if c.State() == StateDisconnected {
		return
	}

	c.log.Info("Client disconnecting",
		logger.String("client_id", c.ID),
		logger.String("user_id", c.UserID.String()),
		logger.String("device_id", c.DeviceID),
	)

	// Mark as disconnecting
	c.setState(StateDisconnecting)

	// Close send channel
	select {
	case <-c.sendDone:
		// Already closed
	default:
		close(c.send)
		close(c.sendDone)
	}

	// Unregister from hub
	if c.hub != nil {
		c.hub.Unregister(c)
	}

	// Call disconnect handler
	if c.onDisconnect != nil {
		c.onDisconnect(c)
	}

	// Close connection
	c.connMu.Lock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.connMu.Unlock()

	// Cancel context
	c.cancel()

	c.setState(StateDisconnected)

	c.log.Info("Client disconnected",
		logger.String("client_id", c.ID),
		logger.String("user_id", c.UserID.String()),
		logger.Int64("messages_sent", c.messagesSent.Load()),
		logger.Int64("messages_received", c.messagesReceived.Load()),
		logger.Duration("session_duration", time.Since(c.createdAt)),
	)
}

// handleError handles client errors
func (c *Client) handleError(err error, message string) {
	if err == nil {
		return
	}

	c.log.Error(message,
		logger.String("client_id", c.ID),
		logger.String("user_id", c.UserID.String()),
		logger.Error(err),
	)

	if c.onError != nil {
		c.onError(c, err)
	}
}

// updateActivity updates the last activity timestamp
func (c *Client) updateActivity() {
	c.lastActivity.Store(time.Now())
}

// LastActivity returns the last activity time
func (c *Client) LastActivity() time.Time {
	return c.lastActivity.Load().(time.Time)
}

// LastPing returns the last ping time
func (c *Client) LastPing() time.Time {
	return c.lastPing.Load().(time.Time)
}

// IsStale returns true if the connection is stale
func (c *Client) IsStale() bool {
	return time.Since(c.LastActivity()) > c.config.StaleConnectionTimeout
}

// Metadata returns client metadata
func (c *Client) Metadata() ClientMetadata {
	return c.metadata
}

// Stats returns client statistics
func (c *Client) Stats() map[string]interface{} {
	return map[string]interface{}{
		"client_id":         c.ID,
		"user_id":           c.UserID.String(),
		"device_id":         c.DeviceID,
		"state":             c.State().String(),
		"connected_at":      c.createdAt,
		"last_activity":     c.LastActivity(),
		"last_ping":         c.LastPing(),
		"messages_sent":     c.messagesSent.Load(),
		"messages_received": c.messagesReceived.Load(),
		"bytes_sent":        c.bytesSent.Load(),
		"bytes_received":    c.bytesReceived.Load(),
		"uptime":            time.Since(c.createdAt),
		"is_stale":          c.IsStale(),
	}
}

// Set stores custom data
func (c *Client) Set(key string, value interface{}) {
	c.data.Set(key, value)
}

// Get retrieves custom data
func (c *Client) Get(key string) (interface{}, bool) {
	return c.data.Get(key)
}

// Delete removes custom data
func (c *Client) Delete(key string) {
	c.data.Delete(key)
}

// Context returns the client context
func (c *Client) Context() context.Context {
	return c.ctx
}

// generateClientID generates a unique client ID
func generateClientID() string {
	return uuid.New().String()
}
