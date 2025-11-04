package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"echo-backend/services/message-service/internal/model"

	"github.com/google/uuid"
	"shared/pkg/logger"
)

// Hub maintains active WebSocket connections and handles message broadcasting
// Similar to WhatsApp/Telegram, it supports multiple devices per user
type Hub struct {
	// Map of user ID to their active connections (multi-device support)
	clients map[uuid.UUID]map[*Client]bool

	// Map of device/connection ID to client
	connections map[string]*Client

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Broadcast message to specific user (all devices)
	broadcast chan *BroadcastMessage

	// Broadcast to multiple users
	broadcastMulti chan *MultiBroadcastMessage

	// Mutex for thread-safe operations
	mu sync.RWMutex

	// Logger
	logger logger.Logger

	// Metrics
	totalConnections    int64
	totalMessages       int64
	totalBroadcasts     int64
	connectionsDuration map[string]time.Time
}

// BroadcastMessage represents a message to broadcast to a specific user
type BroadcastMessage struct {
	UserID  uuid.UUID
	Payload []byte
}

// MultiBroadcastMessage represents a message to broadcast to multiple users
type MultiBroadcastMessage struct {
	UserIDs      []uuid.UUID
	Payload      []byte
	ExcludeUsers []uuid.UUID // Users to exclude from broadcast (e.g., sender)
}

// NewHub creates a new WebSocket hub
func NewHub(logger logger.Logger) *Hub {
	return &Hub{
		clients:             make(map[uuid.UUID]map[*Client]bool),
		connections:         make(map[string]*Client),
		register:            make(chan *Client, 256),
		unregister:          make(chan *Client, 256),
		broadcast:           make(chan *BroadcastMessage, 1024),
		broadcastMulti:      make(chan *MultiBroadcastMessage, 1024),
		logger:              logger,
		connectionsDuration: make(map[string]time.Time),
	}
}

// Run starts the hub's main event loop
// This should be called as a goroutine
func (h *Hub) Run() {
	h.logger.Info("WebSocket hub starting")

	// Start cleanup goroutine for stale connections
	go h.cleanupStaleConnections()

	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastToUser(message)

		case message := <-h.broadcastMulti:
			h.broadcastToMultipleUsers(message)
		}
	}
}

// registerClient registers a new client connection
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Initialize user's client map if doesn't exist
	if h.clients[client.UserID] == nil {
		h.clients[client.UserID] = make(map[*Client]bool)
	}

	// Add client to user's connections
	h.clients[client.UserID][client] = true

	// Add to connections map
	h.connections[client.ID] = client

	// Track connection time
	h.connectionsDuration[client.ID] = time.Now()

	h.totalConnections++

	h.logger.Info("Client connected",
		logger.String("client_id", client.ID),
		logger.String("user_id", client.UserID.String()),
		logger.String("device_id", client.DeviceID),
		logger.Int("total_devices", len(h.clients[client.UserID])),
		logger.Int64("total_connections", h.totalConnections),
	)

	// Send welcome message
	welcomeMsg := model.WebSocketMessage{
		Type: "connection_ack",
		Payload: map[string]interface{}{
			"status":    "connected",
			"timestamp": time.Now(),
			"client_id": client.ID,
		},
	}
	h.sendToClient(client, welcomeMsg)
}

// unregisterClient removes a client connection
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Remove from user's clients map
	if clients, ok := h.clients[client.UserID]; ok {
		if _, exists := clients[client]; exists {
			delete(clients, client)
			close(client.send)

			// If user has no more connections, remove user entry
			if len(clients) == 0 {
				delete(h.clients, client.UserID)
			}
		}
	}

	// Remove from connections map
	delete(h.connections, client.ID)

	// Calculate connection duration
	if startTime, ok := h.connectionsDuration[client.ID]; ok {
		duration := time.Since(startTime)
		delete(h.connectionsDuration, client.ID)

		h.logger.Info("Client disconnected",
			logger.String("client_id", client.ID),
			logger.String("user_id", client.UserID.String()),
			logger.Duration("duration", duration),
			logger.Int("remaining_devices", len(h.clients[client.UserID])),
		)
	}
}

// broadcastToUser sends a message to all devices of a specific user
func (h *Hub) broadcastToUser(message *BroadcastMessage) {
	h.mu.RLock()
	clients := h.clients[message.UserID]
	h.mu.RUnlock()

	if len(clients) == 0 {
		h.logger.Debug("User not online",
			logger.String("user_id", message.UserID.String()),
		)
		return
	}

	h.totalBroadcasts++
	successCount := 0

	// Send to all devices
	for client := range clients {
		select {
		case client.send <- message.Payload:
			h.totalMessages++
			successCount++
		default:
			// Client's send buffer is full, disconnect
			h.logger.Warn("Client buffer full, disconnecting",
				logger.String("client_id", client.ID),
				logger.String("user_id", client.UserID.String()),
			)
			h.unregister <- client
		}
	}

	h.logger.Debug("Message broadcasted to user",
		logger.String("user_id", message.UserID.String()),
		logger.Int("devices", len(clients)),
		logger.Int("success", successCount),
	)
}

// broadcastToMultipleUsers sends a message to multiple users
func (h *Hub) broadcastToMultipleUsers(message *MultiBroadcastMessage) {
	excludeMap := make(map[uuid.UUID]bool)
	for _, userID := range message.ExcludeUsers {
		excludeMap[userID] = true
	}

	for _, userID := range message.UserIDs {
		// Skip excluded users
		if excludeMap[userID] {
			continue
		}

		// Broadcast to this user
		h.broadcastToUser(&BroadcastMessage{
			UserID:  userID,
			Payload: message.Payload,
		})
	}
}

// SendToUser sends a message to all devices of a specific user
func (h *Hub) SendToUser(userID uuid.UUID, message interface{}) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}

	h.broadcast <- &BroadcastMessage{
		UserID:  userID,
		Payload: payload,
	}

	return nil
}

// SendToUsers sends a message to multiple users
func (h *Hub) SendToUsers(userIDs []uuid.UUID, message interface{}, excludeUsers []uuid.UUID) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}

	h.broadcastMulti <- &MultiBroadcastMessage{
		UserIDs:      userIDs,
		Payload:      payload,
		ExcludeUsers: excludeUsers,
	}

	return nil
}

// sendToClient sends a message to a specific client
func (h *Hub) sendToClient(client *Client, message interface{}) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}

	select {
	case client.send <- payload:
		return nil
	default:
		// Buffer full, disconnect client
		h.unregister <- client
		return ErrClientBufferFull
	}
}

// IsUserOnline checks if a user has any active connections
func (h *Hub) IsUserOnline(userID uuid.UUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.clients[userID]
	return ok && len(clients) > 0
}

// GetOnlineUsers returns all currently online user IDs
func (h *Hub) GetOnlineUsers() []uuid.UUID {
	h.mu.RLock()
	defer h.mu.RUnlock()

	users := make([]uuid.UUID, 0, len(h.clients))
	for userID := range h.clients {
		users = append(users, userID)
	}

	return users
}

// GetUserDeviceCount returns the number of active devices for a user
func (h *Hub) GetUserDeviceCount(userID uuid.UUID) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.clients[userID]; ok {
		return len(clients)
	}

	return 0
}

// GetStats returns hub statistics
func (h *Hub) GetStats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	totalUsers := len(h.clients)
	totalDevices := 0
	for _, clients := range h.clients {
		totalDevices += len(clients)
	}

	return map[string]interface{}{
		"total_users":       totalUsers,
		"total_devices":     totalDevices,
		"total_connections": h.totalConnections,
		"total_messages":    h.totalMessages,
		"total_broadcasts":  h.totalBroadcasts,
	}
}

// cleanupStaleConnections periodically removes stale connections
func (h *Hub) cleanupStaleConnections() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		h.mu.RLock()
		staleClients := make([]*Client, 0)

		for _, client := range h.connections {
			// Check if client is stale (no ping/pong for 90 seconds)
			if time.Since(client.lastPong) > 90*time.Second {
				staleClients = append(staleClients, client)
			}
		}
		h.mu.RUnlock()

		// Disconnect stale clients
		for _, client := range staleClients {
			h.logger.Warn("Disconnecting stale client",
				logger.String("client_id", client.ID),
				logger.Duration("since_last_pong", time.Since(client.lastPong)),
			)
			h.unregister <- client
		}

		if len(staleClients) > 0 {
			h.logger.Info("Cleaned up stale connections",
				logger.Int("count", len(staleClients)),
			)
		}
	}
}

// Register registers a new client with the hub
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister removes a client from the hub
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Shutdown gracefully shuts down the hub
func (h *Hub) Shutdown() {
	h.logger.Info("Shutting down WebSocket hub")

	h.mu.Lock()
	defer h.mu.Unlock()

	// Close all client connections
	for _, client := range h.connections {
		close(client.send)
	}

	// Clear maps
	h.clients = make(map[uuid.UUID]map[*Client]bool)
	h.connections = make(map[string]*Client)

	h.logger.Info("WebSocket hub shutdown complete")
}

// Custom errors
var (
	ErrClientBufferFull = &HubError{Message: "client send buffer is full"}
)

type HubError struct {
	Message string
}

func (e *HubError) Error() string {
	return e.Message
}
