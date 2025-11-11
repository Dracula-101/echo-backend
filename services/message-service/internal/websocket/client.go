package websocket

import (
	"echo-backend/services/message-service/internal/models"
	"encoding/json"
	"time"

	"shared/pkg/logger"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait = 10 * time.Second

	pongWait = 60 * time.Second

	pingPeriod = (pongWait * 9) / 10

	maxMessageSize = 10 * 1024 * 1024

	clientBufferSize = 256
)

// Client represents a WebSocket connection from a specific device
type Client struct {
	ID       string
	UserID   uuid.UUID
	DeviceID string
	conn     *websocket.Conn
	send     chan []byte
	hub      *Hub
	log      logger.Logger
	lastPong time.Time
	metadata ClientMetadata
}

// ClientMetadata contains additional client information
type ClientMetadata struct {
	IPAddress   string
	UserAgent   string
	Platform    string // ios, android, web
	AppVersion  string
	ConnectedAt time.Time
}

// NewClient creates a new client instance
func NewClient(userID uuid.UUID, deviceID string, conn *websocket.Conn, hub *Hub, log logger.Logger, metadata ClientMetadata) *Client {
	return &Client{
		ID:       uuid.New().String(),
		UserID:   userID,
		DeviceID: deviceID,
		conn:     conn,
		send:     make(chan []byte, clientBufferSize),
		hub:      hub,
		log:      log,
		lastPong: time.Now(),
		metadata: metadata,
	}
}

// ReadPump reads messages from the WebSocket connection
// It handles incoming messages like read receipts, typing indicators, etc.
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
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
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

func (c *Client) SendMessage(message interface{}) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}

	select {
	case c.send <- payload:
		return nil
	default:
		c.log.Warn("Client send buffer full",
			logger.String("client_id", c.ID),
			logger.String("user_id", c.UserID.String()),
		)
		return ErrClientBufferFull
	}
}

type IncomingMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type MessageHandler func(*Client, []byte)

func CreateMessageHandler(
	log logger.Logger,
	onReadReceipt func(userID, messageID uuid.UUID),
	onTyping func(userID, conversationID uuid.UUID, isTyping bool),
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
		case "read_receipt":
			handleReadReceipt(client, msg.Payload, log, onReadReceipt)

		case "typing":
			handleTypingIndicator(client, msg.Payload, log, onTyping)

		case "ping":
			handlePing(client, log)

		default:
			log.Fatal("Unknown message type",
				logger.String("type", msg.Type),
				logger.String("client_id", client.ID),
			)
		}
	}
}

func handleReadReceipt(client *Client, payload json.RawMessage, log logger.Logger, callback func(uuid.UUID, uuid.UUID)) {
	var receipt models.ReadReceipt
	if err := json.Unmarshal(payload, &receipt); err != nil {
		log.Error("Failed to unmarshal read receipt",
			logger.String("client_id", client.ID),
			logger.Error(err),
		)
		return
	}

	if callback != nil {
		callback(client.UserID, receipt.MessageID)
	}

	log.Debug("Read receipt received",
		logger.String("client_id", client.ID),
		logger.String("message_id", receipt.MessageID.String()),
	)
}

// handleTypingIndicator processes typing indicator messages
func handleTypingIndicator(client *Client, payload json.RawMessage, log logger.Logger, callback func(uuid.UUID, uuid.UUID, bool)) {
	var typing models.TypingIndicator
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
		logger.String("conversation_id", typing.ConversationID.String()),
		logger.Bool("is_typing", typing.IsTyping),
	)
}

// handlePing responds to ping messages
func handlePing(client *Client, log logger.Logger) {
	pong := models.WebSocketMessage{
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
