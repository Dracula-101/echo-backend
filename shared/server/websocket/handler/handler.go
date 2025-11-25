package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"shared/pkg/logger"
	"shared/server/websocket/connection"
	"shared/server/websocket/state"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Connection is an alias for connection.Connection for easier usage
type Connection = connection.Connection

// UserValidator validates if a user can connect
type UserValidator func(ctx context.Context, userID uuid.UUID) (bool, error)

// UserIDExtractor extracts user ID from HTTP request
type UserIDExtractor func(r *http.Request) (uuid.UUID, error)

// MessageHandler handles incoming WebSocket messages
type MessageHandler func(ctx context.Context, conn *Connection, message []byte) error

// ConnectionMetadataFunc extracts metadata from HTTP request
type ConnectionMetadataFunc func(r *http.Request) map[string]any

// Engine interface to avoid circular dependency
type Engine interface {
	ConnectionManager() *connection.Manager
}

// Config holds handler configuration
type Config struct {
	// Connection configuration
	SendBufferSize int
	MaxMessageSize int64
	PingInterval   time.Duration
	WriteTimeout   time.Duration
	ReadTimeout    time.Duration
	StaleTimeout   time.Duration

	// Upgrader configuration
	CheckOrigin       func(r *http.Request) bool
	ReadBufferSize    int
	WriteBufferSize   int
	EnableCompression bool

	// Callbacks
	ValidateUser    UserValidator
	ExtractUserID   UserIDExtractor
	HandleMessage   MessageHandler
	ExtractMetadata ConnectionMetadataFunc
	OnConnected     func(conn *Connection)
	OnDisconnected  func(conn *Connection)

	// Validation
	MessageValidator   ErrorValidator
	SendErrorsToClient bool
}

// DefaultConfig returns default handler configuration
func DefaultConfig() *Config {
	return &Config{
		SendBufferSize:     256,
		MaxMessageSize:     10 * 1024 * 1024, // 10MB
		PingInterval:       54 * time.Second,
		WriteTimeout:       10 * time.Second,
		ReadTimeout:        60 * time.Second,
		StaleTimeout:       90 * time.Second,
		CheckOrigin:        func(r *http.Request) bool { return true },
		ReadBufferSize:     1024,
		WriteBufferSize:    1024,
		EnableCompression:  false,
		ExtractUserID:      DefaultUserIDExtractor,
		SendErrorsToClient: true,
	}
}

// Handler handles WebSocket upgrade requests
type Handler struct {
	engine   Engine
	upgrader websocket.Upgrader
	config   *Config
	log      logger.Logger
}

// New creates a new WebSocket handler
func New(engine Engine, config *Config, log logger.Logger) *Handler {
	if config == nil {
		config = DefaultConfig()
	}

	return &Handler{
		engine: engine,
		upgrader: websocket.Upgrader{
			CheckOrigin:       config.CheckOrigin,
			ReadBufferSize:    config.ReadBufferSize,
			WriteBufferSize:   config.WriteBufferSize,
			EnableCompression: config.EnableCompression,
		},
		config: config,
		log:    log,
	}
}

// HandleUpgrade handles the WebSocket upgrade request
func (h *Handler) HandleUpgrade(w http.ResponseWriter, r *http.Request) {
	h.log.Info("WebSocket upgrade request",
		logger.String("remote_addr", r.RemoteAddr),
		logger.String("user_agent", r.UserAgent()),
	)

	// Extract user ID
	userID, err := h.config.ExtractUserID(r)
	if err != nil {
		h.log.Warn("Failed to extract user ID", logger.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate user if validator is provided
	if h.config.ValidateUser != nil {
		exists, err := h.config.ValidateUser(r.Context(), userID)
		if err != nil {
			h.log.Error("Failed to validate user", logger.Error(err))
			http.Error(w, "failed to validate user", http.StatusInternalServerError)
			return
		}
		if !exists {
			h.log.Warn("User not found", logger.String("user_id", userID.String()))
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
	}

	// Upgrade connection
	wsConn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Error("Failed to upgrade connection", logger.Error(err))
		return
	}

	// Create connection configuration
	connConfig := &connection.Config{
		SendBufferSize: h.config.SendBufferSize,
		MaxMessageSize: h.config.MaxMessageSize,
		PingInterval:   h.config.PingInterval,
		WriteTimeout:   h.config.WriteTimeout,
		ReadTimeout:    h.config.ReadTimeout,
		StaleTimeout:   h.config.StaleTimeout,
	}

	// Create connection instance
	connID := uuid.New().String()
	conn := connection.New(connID, wsConn, connConfig, h.log)

	// Set user ID metadata
	conn.SetMetadata("user_id", userID)

	// Extract additional metadata if provided
	if h.config.ExtractMetadata != nil {
		metadata := h.config.ExtractMetadata(r)
		for key, value := range metadata {
			conn.SetMetadata(key, value)
		}
	}

	// Add to connection manager
	if err := h.engine.ConnectionManager().Add(conn); err != nil {
		h.log.Error("Failed to add connection", logger.Error(err))
		conn.Close()
		return
	}

	// Transition to connected state
	if err := conn.TransitionTo(state.StateConnected); err != nil {
		h.log.Error("Failed to transition to connected state", logger.Error(err))
		h.engine.ConnectionManager().Remove(conn.ID())
		conn.Close()
		return
	}

	h.log.Info("WebSocket connection established",
		logger.String("conn_id", conn.ID()),
		logger.String("user_id", userID.String()),
	)

	// Call onConnected callback
	if h.config.OnConnected != nil {
		h.config.OnConnected(conn)
	}

	// Start connection pumps
	go h.startReadPump(conn, wsConn, r)
	go h.startWritePump(conn, wsConn, connConfig)
}

func (h *Handler) startReadPump(conn *Connection, wsConn *websocket.Conn, r *http.Request) {
	defer func() {
		if h.config.OnDisconnected != nil {
			h.config.OnDisconnected(conn)
		}
		conn.Close()
		h.engine.ConnectionManager().Remove(conn.ID())
	}()

	for {
		_, message, err := wsConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
				websocket.CloseNormalClosure) {
				h.log.Error("Unexpected close",
					logger.String("conn_id", conn.ID()),
					logger.Error(err),
				)
			}
			return
		}

		// Check message size
		if int64(len(message)) > h.config.MaxMessageSize {
			h.log.Warn("Message too large",
				logger.String("conn_id", conn.ID()),
				logger.Int("size", len(message)),
				logger.Int64("max_size", h.config.MaxMessageSize),
			)
			h.sendError(conn, ErrMessageTooLarge)
			continue
		}

		// Update connection metrics
		conn.UpdateActivity()
		conn.IncrementMessagesReceived()
		conn.AddBytesReceived(int64(len(message)))

		// Validate message if validator is configured
		if h.config.MessageValidator != nil {
			if err := h.config.MessageValidator.Validate(message); err != nil {
				h.log.Warn("Message validation failed",
					logger.String("conn_id", conn.ID()),
					logger.Error(err),
				)
				if errResp, ok := err.(*ErrorResponse); ok {
					h.sendError(conn, errResp)
				} else {
					h.sendError(conn, NewErrorResponse(
						ErrCodeInvalidField,
						err.Error(),
						nil,
					))
				}
				continue
			}
		}

		// Handle message
		if h.config.HandleMessage != nil {
			if err := h.config.HandleMessage(r.Context(), conn, message); err != nil {
				h.log.Error("Message handling failed",
					logger.String("conn_id", conn.ID()),
					logger.Error(err),
				)
				if h.config.SendErrorsToClient {
					h.sendError(conn, NewErrorResponse(
						ErrCodeInternalError,
						"Failed to process message",
						nil,
					))
				}
			} 
		}
	}
}

func (h *Handler) sendError(conn *Connection, errResp *ErrorResponse) {
	if !h.config.SendErrorsToClient {
		return
	}

	errJSON, err := errResp.ToJSON()
	if err != nil {
		h.log.Error("Failed to marshal error response",
			logger.String("conn_id", conn.ID()),
			logger.Error(err),
		)
		return
	}

	if err := conn.Send(errJSON); err != nil {
		h.log.Error("Failed to send error to client",
			logger.String("conn_id", conn.ID()),
			logger.Error(err),
		)
	}
}

func (h *Handler) startWritePump(conn *Connection, wsConn *websocket.Conn, cfg *connection.Config) {
	ticker := time.NewTicker(cfg.PingInterval)
	defer ticker.Stop()
	defer conn.Close()

	for {
		select {
		case <-conn.Context().Done():
			return
		case message, ok := <-conn.SendChan():
			if !ok {
				return
			}
			wsConn.SetWriteDeadline(time.Now().Add(cfg.WriteTimeout))
			if err := wsConn.WriteMessage(websocket.TextMessage, message); err != nil {
				h.log.Error("Failed to write message",
					logger.String("conn_id", conn.ID()),
					logger.Error(err),
				)
				return
			}
			conn.IncrementMessagesSent()
			conn.AddBytesSent(int64(len(message)))
		case <-ticker.C:
			deadline := time.Now().Add(cfg.WriteTimeout)
			if err := wsConn.WriteControl(websocket.PingMessage, []byte{}, deadline); err != nil {
				h.log.Debug("Failed to send ping",
					logger.String("conn_id", conn.ID()),
					logger.Error(err),
				)
				return
			}
		}
	}
}

// DefaultUserIDExtractor extracts user ID from header or query param
func DefaultUserIDExtractor(r *http.Request) (uuid.UUID, error) {
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		userIDStr = r.URL.Query().Get("user_id")
	}

	if userIDStr == "" {
		return uuid.Nil, errors.New("user_id is required")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, errors.New("invalid user_id format")
	}

	return userID, nil
}

// DefaultMetadataExtractor extracts common metadata from request
func DefaultMetadataExtractor(r *http.Request) map[string]any {
	return map[string]any{
		"device_id":  r.Header.Get("X-Device-ID"),
		"platform":   r.Header.Get("X-Platform"),
		"ip_address": r.RemoteAddr,
		"user_agent": r.UserAgent(),
	}
}
