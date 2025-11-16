package websocket

import (
	"presence-service/internal/model"
	"sync"
	"time"

	"shared/pkg/logger"

	"github.com/google/uuid"
)

// Hub maintains active WebSocket connections for presence tracking
// Similar to message-service, it supports multiple devices per user
type Hub struct {
	// Map of user ID to their active connections (multi-device support)
	clients map[uuid.UUID]map[*Client]bool

	// Map of device/connection ID to client
	connections map[string]*Client

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Presence updates to broadcast
	presenceUpdates chan *HubPresenceUpdate

	// Typing indicators to broadcast
	typingIndicators chan *HubTypingBroadcast

	// Mutex for thread-safe operations
	mu sync.RWMutex

	// Disconnect callback
	onDisconnect func(*Client)

	// Logger
	logger logger.Logger

	// Metrics
	totalConnections    int64
	connectionsDuration map[string]time.Time
}

// NewHub creates a new WebSocket hub for presence tracking
func NewHub(logger logger.Logger) *Hub {
	return &Hub{
		clients:             make(map[uuid.UUID]map[*Client]bool),
		connections:         make(map[string]*Client),
		register:            make(chan *Client, 256),
		unregister:          make(chan *Client, 256),
		presenceUpdates:     make(chan *HubPresenceUpdate, 1024),
		typingIndicators:    make(chan *HubTypingBroadcast, 1024),
		logger:              logger,
		connectionsDuration: make(map[string]time.Time),
	}
}

// Run starts the hub's main event loop
func (h *Hub) Run() {
	h.logger.Info("Presence WebSocket hub starting")

	// Start cleanup goroutine for stale connections
	go h.cleanupStaleConnections()

	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case update := <-h.presenceUpdates:
			h.broadcastPresenceUpdate(update)

		case typing := <-h.typingIndicators:
			h.broadcastTypingIndicator(typing)
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

	h.logger.Info("Presence client connected",
		logger.String("client_id", client.ID),
		logger.String("user_id", client.UserID.String()),
		logger.String("device_id", client.DeviceID),
		logger.Int("total_devices", len(h.clients[client.UserID])),
		logger.Int64("total_connections", h.totalConnections),
	)

	// Send welcome message
	welcomeMsg := model.PresenceEvent{
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

	// Check if client exists before proceeding
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
	duration := time.Duration(0)
	if startTime, ok := h.connectionsDuration[client.ID]; ok {
		duration = time.Since(startTime)
		delete(h.connectionsDuration, client.ID)
	}

	h.mu.Unlock()

	// Call disconnect callback after unlocking to avoid potential deadlocks
	if h.onDisconnect != nil {
		h.onDisconnect(client)
	}

	h.logger.Info("Presence client disconnected",
		logger.String("client_id", client.ID),
		logger.String("user_id", client.UserID.String()),
		logger.Duration("duration", duration),
		logger.Int("remaining_devices", len(h.clients[client.UserID])),
	)
}

// broadcastPresenceUpdate broadcasts presence updates to relevant users
func (h *Hub) broadcastPresenceUpdate(update *HubPresenceUpdate) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	message := model.PresenceEvent{
		Type: "presence_update",
		Payload: map[string]interface{}{
			"user_id":       update.UserID,
			"online_status": update.OnlineStatus,
			"custom_status": update.CustomStatus,
			"timestamp":     time.Now(),
		},
	}

	// Broadcast to specified users
	for _, userID := range update.BroadcastTo {
		if clients, ok := h.clients[userID]; ok {
			for client := range clients {
				h.sendToClient(client, message)
			}
		}
	}
}

// broadcastTypingIndicator broadcasts typing indicators to conversation participants
func (h *Hub) broadcastTypingIndicator(typing *HubTypingBroadcast) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	message := model.PresenceEvent{
		Type: "typing_indicator",
		Payload: map[string]interface{}{
			"conversation_id": typing.ConversationID,
			"user_id":         typing.UserID,
			"is_typing":       typing.IsTyping,
			"timestamp":       time.Now(),
		},
	}

	// Broadcast to participants (except the sender)
	for _, userID := range typing.Participants {
		if userID == typing.UserID {
			continue // Don't send to sender
		}

		if clients, ok := h.clients[userID]; ok {
			for client := range clients {
				h.sendToClient(client, message)
			}
		}
	}
}

// sendToClient sends a message to a specific client
func (h *Hub) sendToClient(client *Client, message interface{}) {
	select {
	case client.send <- message:
	default:
		// Buffer full, disconnect client
		h.logger.Warn("Client buffer full, will disconnect",
			logger.String("client_id", client.ID),
			logger.String("user_id", client.UserID.String()),
		)
		go func() {
			h.unregister <- client
		}()
	}
}

// BroadcastPresenceUpdate queues a presence update for broadcasting
func (h *Hub) BroadcastPresenceUpdate(update *HubPresenceUpdate) {
	select {
	case h.presenceUpdates <- update:
	default:
		h.logger.Warn("Presence update channel full, dropping update",
			logger.String("user_id", update.UserID.String()),
		)
	}
}

// BroadcastTypingIndicator queues a typing indicator for broadcasting
func (h *Hub) BroadcastTypingIndicator(typing *HubTypingBroadcast) {
	select {
	case h.typingIndicators <- typing:
	default:
		h.logger.Warn("Typing indicator channel full, dropping indicator")
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

// GetActiveDevices returns active device information for a user
func (h *Hub) GetActiveDevices(userID uuid.UUID) []*model.Device {
	h.mu.RLock()
	defer h.mu.RUnlock()

	devices := make([]*model.Device, 0)

	if clients, ok := h.clients[userID]; ok {
		for client := range clients {
			device := &model.Device{
				ID:           uuid.MustParse(client.ID),
				UserID:       client.UserID,
				DeviceID:     client.DeviceID,
				DeviceName:   client.metadata.DeviceName,
				Platform:     client.metadata.Platform,
				AppVersion:   client.metadata.AppVersion,
				IsActive:     true,
				LastActiveAt: time.Now(),
				RegisteredAt: client.metadata.ConnectedAt,
			}
			devices = append(devices, device)
		}
	}

	return devices
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
			h.logger.Warn("Disconnecting stale presence client",
				logger.String("client_id", client.ID),
				logger.Duration("since_last_pong", time.Since(client.lastPong)),
			)
			h.unregister <- client
		}

		if len(staleClients) > 0 {
			h.logger.Info("Cleaned up stale presence connections",
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
	h.logger.Info("Shutting down Presence WebSocket hub")

	h.mu.Lock()
	defer h.mu.Unlock()

	// Close all client connections
	for _, client := range h.connections {
		close(client.send)
	}

	// Clear maps
	h.clients = make(map[uuid.UUID]map[*Client]bool)
	h.connections = make(map[string]*Client)

	h.logger.Info("Presence WebSocket hub shutdown complete")
}
