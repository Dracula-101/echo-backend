package websocket

import (
	"encoding/json"
	"presence-service/internal/model"
	"time"

	"shared/pkg/logger"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 10 * 1024 * 1024

	// Client send buffer size
	clientBufferSize = 256
)

// Client represents a WebSocket connection from a specific device for presence tracking
type Client struct {
	ID       string
	UserID   uuid.UUID
	DeviceID string
	conn     *websocket.Conn
	send     chan interface{}
	hub      *Hub
	log      logger.Logger
	lastPong time.Time
	metadata ClientMetadata
}

// NewClient creates a new client instance
func NewClient(userID uuid.UUID, deviceID string, conn *websocket.Conn, hub *Hub, log logger.Logger, metadata ClientMetadata) *Client {
	return &Client{
		ID:       uuid.New().String(),
		UserID:   userID,
		DeviceID: deviceID,
		conn:     conn,
		send:     make(chan interface{}, clientBufferSize),
		hub:      hub,
		log:      log,
		lastPong: time.Now(),
		metadata: metadata,
	}
}

// ReadPump reads messages from the WebSocket connection
// It handles incoming presence events like heartbeat, status updates, typing indicators
func (c *Client) ReadPump(messageHandler func(*Client, []byte)) {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.lastPong = time.Now()
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.log.Error("WebSocket read error",
					logger.String("client_id", c.ID),
					logger.Error(err),
				)
			}
			break
		}

		// Handle incoming message
		if messageHandler != nil {
			go messageHandler(c, message)
		}
	}
}

// WritePump writes messages to the WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Marshal message to JSON
			payload, err := json.Marshal(message)
			if err != nil {
				c.log.Error("Failed to marshal message",
					logger.String("client_id", c.ID),
					logger.Error(err),
				)
				continue
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(payload)

			// Add queued messages to current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				nextMsg := <-c.send
				nextPayload, err := json.Marshal(nextMsg)
				if err != nil {
					continue
				}
				w.Write([]byte{'\n'})
				w.Write(nextPayload)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// SendMessage sends a message to the client
func (c *Client) SendMessage(message interface{}) error {
	select {
	case c.send <- message:
		return nil
	default:
		c.log.Warn("Client send buffer full",
			logger.String("client_id", c.ID),
			logger.String("user_id", c.UserID.String()),
		)
		return ErrClientBufferFull
	}
}

// MessageHandler is a function that handles incoming WebSocket messages
type MessageHandler func(*Client, []byte)

// CreateMessageHandler creates a handler for incoming WebSocket messages
func CreateMessageHandler(
	log logger.Logger,
	onPresenceUpdate func(userID uuid.UUID, status string, customStatus string),
	onHeartbeat func(userID uuid.UUID, deviceID string),
	onTyping func(userID uuid.UUID, conversationID uuid.UUID, isTyping bool),
) MessageHandler {
	return func(client *Client, messageData []byte) {
		var msg IncomingMessage
		if err := json.Unmarshal(messageData, &msg); err != nil {
			log.Error("Failed to unmarshal incoming message",
				logger.String("client_id", client.ID),
				logger.Error(err),
			)
			return
		}

		switch msg.Type {
		case "presence_update":
			handlePresenceUpdate(client, msg.Payload, log, onPresenceUpdate)

		case "heartbeat":
			handleHeartbeat(client, msg.Payload, log, onHeartbeat)

		case "typing":
			handleTyping(client, msg.Payload, log, onTyping)

		case "ping":
			handlePing(client, log)

		default:
			log.Warn("Unknown message type",
				logger.String("type", msg.Type),
				logger.String("client_id", client.ID),
			)
		}
	}
}

// handlePresenceUpdate processes presence status updates
func handlePresenceUpdate(client *Client, payload json.RawMessage, log logger.Logger, callback func(uuid.UUID, string, string)) {
	var update struct {
		OnlineStatus string `json:"online_status"`
		CustomStatus string `json:"custom_status"`
	}

	if err := json.Unmarshal(payload, &update); err != nil {
		log.Error("Failed to unmarshal presence update",
			logger.String("client_id", client.ID),
			logger.Error(err),
		)
		return
	}

	if callback != nil {
		callback(client.UserID, update.OnlineStatus, update.CustomStatus)
	}

	log.Debug("Presence update received",
		logger.String("client_id", client.ID),
		logger.String("user_id", client.UserID.String()),
		logger.String("status", update.OnlineStatus),
	)
}

// handleHeartbeat processes heartbeat messages
func handleHeartbeat(client *Client, payload json.RawMessage, log logger.Logger, callback func(uuid.UUID, string)) {
	if callback != nil {
		callback(client.UserID, client.DeviceID)
	}

	log.Debug("Heartbeat received",
		logger.String("client_id", client.ID),
		logger.String("user_id", client.UserID.String()),
		logger.String("device_id", client.DeviceID),
	)
}

// handleTyping processes typing indicator messages
func handleTyping(client *Client, payload json.RawMessage, log logger.Logger, callback func(uuid.UUID, uuid.UUID, bool)) {
	var typing model.TypingIndicator
	if err := json.Unmarshal(payload, &typing); err != nil {
		log.Error("Failed to unmarshal typing indicator",
			logger.String("client_id", client.ID),
			logger.Error(err),
		)
		return
	}

	if callback != nil {
		callback(client.UserID, typing.ConversationID, typing.IsTyping)
	}

	log.Debug("Typing indicator received",
		logger.String("client_id", client.ID),
		logger.String("user_id", client.UserID.String()),
		logger.String("conversation_id", typing.ConversationID.String()),
		logger.Bool("is_typing", typing.IsTyping),
	)
}

// handlePing responds to ping messages
func handlePing(client *Client, log logger.Logger) {
	pong := model.PresenceEvent{
		Type: "pong",
		Payload: map[string]interface{}{
			"timestamp": time.Now(),
		},
	}

	if err := client.SendMessage(pong); err != nil {
		log.Error("Failed to send pong",
			logger.String("client_id", client.ID),
			logger.Error(err),
		)
	}
}

// Custom errors
var (
	ErrClientBufferFull = &ClientError{Message: "client send buffer is full"}
)

type ClientError struct {
	Message string
}

func (e *ClientError) Error() string {
	return e.Message
}
