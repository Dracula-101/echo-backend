package websocket

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"shared/pkg/logger"

	"github.com/google/uuid"
)

// Hub maintains the set of active clients and broadcasts messages to the clients
type Hub struct {
	// Client management
	clients   map[uuid.UUID]map[*Client]bool // userID -> clients
	clientsMu sync.RWMutex

	// Client lookup by ID
	clientsByID   map[string]*Client
	clientsByIDMu sync.RWMutex

	// Registration requests from clients
	register chan *Client

	// Unregistration requests from clients
	unregister chan *Client

	// Broadcast messages to clients
	broadcast chan *BroadcastMessage

	// Configuration
	config *Config

	// Logger
	log logger.Logger

	// State
	running    atomic.Bool
	startTime  time.Time
	shutdownCh chan struct{}

	// Statistics
	totalConnections   atomic.Int64
	totalDisconnections atomic.Int64
	messagesSent       atomic.Int64
	messagesReceived   atomic.Int64
	bytesSent          atomic.Int64
	bytesReceived      atomic.Int64

	// Lifecycle hooks
	onConnect    ConnectHandler
	onDisconnect DisconnectHandler

	// Context
	ctx    context.Context
	cancel context.CancelFunc

	// Cleanup ticker
	cleanupTicker *time.Ticker

	// Rate limiter (optional)
	rateLimiter RateLimiter

	// Metrics
	metricsEnabled bool
	metricsTicker  *time.Ticker
}

// NewHub creates a new WebSocket hub
func NewHub(config *Config, log logger.Logger) *Hub {
	if config == nil {
		config = DefaultConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	hub := &Hub{
		clients:        make(map[uuid.UUID]map[*Client]bool),
		clientsByID:    make(map[string]*Client),
		register:       make(chan *Client, config.RegisterBuffer),
		unregister:     make(chan *Client, config.UnregisterBuffer),
		broadcast:      make(chan *BroadcastMessage, config.BroadcastBuffer),
		config:         config,
		log:            log,
		startTime:      time.Now(),
		shutdownCh:     make(chan struct{}),
		ctx:            ctx,
		cancel:         cancel,
		metricsEnabled: config.EnableMetrics,
	}

	return hub
}

// SetOnConnect sets the connect handler
func (h *Hub) SetOnConnect(handler ConnectHandler) {
	h.onConnect = handler
}

// SetOnDisconnect sets the disconnect handler
func (h *Hub) SetOnDisconnect(handler DisconnectHandler) {
	h.onDisconnect = handler
}

// SetRateLimiter sets the rate limiter
func (h *Hub) SetRateLimiter(limiter RateLimiter) {
	h.rateLimiter = limiter
}

// Run starts the hub
func (h *Hub) Run() {
	if h.running.Load() {
		h.log.Warn("Hub already running")
		return
	}

	h.running.Store(true)
	h.startTime = time.Now()

	// Start cleanup ticker
	h.cleanupTicker = time.NewTicker(h.config.CleanupInterval)

	// Start metrics ticker if enabled
	if h.metricsEnabled {
		h.metricsTicker = time.NewTicker(h.config.MetricsInterval)
	}

	h.log.Info("WebSocket Hub started",
		logger.Int("register_buffer", h.config.RegisterBuffer),
		logger.Int("unregister_buffer", h.config.UnregisterBuffer),
		logger.Int("broadcast_buffer", h.config.BroadcastBuffer),
	)

	for {
		select {
		case client := <-h.register:
			h.handleRegister(client)

		case client := <-h.unregister:
			h.handleUnregister(client)

		case message := <-h.broadcast:
			h.handleBroadcast(message)

		case <-h.cleanupTicker.C:
			h.cleanupStaleConnections()

		case <-h.metricsTicker.C:
			if h.metricsEnabled {
				h.logMetrics()
			}

		case <-h.shutdownCh:
			h.log.Info("Hub shutdown signal received")
			h.shutdown()
			return

		case <-h.ctx.Done():
			h.log.Info("Hub context cancelled")
			h.shutdown()
			return
		}
	}
}

// Register registers a new client
func (h *Hub) Register(client *Client) {
	if !h.running.Load() {
		h.log.Error("Cannot register client: hub not running",
			logger.String("client_id", client.ID),
		)
		return
	}

	// Check connection limit per user
	if h.config.MaxConnectionsPerUser > 0 {
		h.clientsMu.RLock()
		userClients := h.clients[client.UserID]
		count := len(userClients)
		h.clientsMu.RUnlock()

		if count >= h.config.MaxConnectionsPerUser {
			h.log.Warn("Max connections per user exceeded",
				logger.String("user_id", client.UserID.String()),
				logger.Int("current", count),
				logger.Int("max", h.config.MaxConnectionsPerUser),
			)
			client.SendMessage(map[string]interface{}{
				"type":  "error",
				"error": "max_connections_exceeded",
			})
			client.Close()
			return
		}
	}

	select {
	case h.register <- client:
	case <-time.After(5 * time.Second):
		h.log.Error("Failed to register client: timeout",
			logger.String("client_id", client.ID),
		)
		client.Close()
	}
}

// Unregister unregisters a client
func (h *Hub) Unregister(client *Client) {
	if !h.running.Load() {
		return
	}

	select {
	case h.unregister <- client:
	case <-time.After(5 * time.Second):
		h.log.Error("Failed to unregister client: timeout",
			logger.String("client_id", client.ID),
		)
	}
}

// handleRegister handles client registration
func (h *Hub) handleRegister(client *Client) {
	h.clientsMu.Lock()
	defer h.clientsMu.Unlock()

	// Add to user's clients map
	if h.clients[client.UserID] == nil {
		h.clients[client.UserID] = make(map[*Client]bool)
	}
	h.clients[client.UserID][client] = true

	// Add to ID lookup map
	h.clientsByIDMu.Lock()
	h.clientsByID[client.ID] = client
	h.clientsByIDMu.Unlock()

	h.totalConnections.Add(1)

	h.log.Info("Client registered",
		logger.String("client_id", client.ID),
		logger.String("user_id", client.UserID.String()),
		logger.String("device_id", client.DeviceID),
		logger.Int("user_connections", len(h.clients[client.UserID])),
		logger.Int("total_connections", h.GetConnectionCount()),
	)

	// Call connect handler
	if h.onConnect != nil {
		h.onConnect(client)
	}
}

// handleUnregister handles client unregistration
func (h *Hub) handleUnregister(client *Client) {
	h.clientsMu.Lock()
	defer h.clientsMu.Unlock()

	// Remove from user's clients map
	if userClients, ok := h.clients[client.UserID]; ok {
		if _, exists := userClients[client]; exists {
			delete(userClients, client)
			h.totalDisconnections.Add(1)

			// If no more clients for this user, remove the user entry
			if len(userClients) == 0 {
				delete(h.clients, client.UserID)
			}
		}
	}

	// Remove from ID lookup map
	h.clientsByIDMu.Lock()
	delete(h.clientsByID, client.ID)
	h.clientsByIDMu.Unlock()

	h.log.Info("Client unregistered",
		logger.String("client_id", client.ID),
		logger.String("user_id", client.UserID.String()),
		logger.String("device_id", client.DeviceID),
		logger.Int("remaining_user_connections", len(h.clients[client.UserID])),
		logger.Int("total_connections", h.GetConnectionCount()),
	)

	// Call disconnect handler
	if h.onDisconnect != nil {
		h.onDisconnect(client)
	}
}

// handleBroadcast handles message broadcasting
func (h *Hub) handleBroadcast(msg *BroadcastMessage) {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()

	var clients []*Client
	excludeMap := make(map[string]bool)
	for _, id := range msg.Exclude {
		excludeMap[id] = true
	}

	switch msg.Target {
	case BroadcastAll:
		// Broadcast to all clients
		for _, userClients := range h.clients {
			for client := range userClients {
				if !excludeMap[client.ID] {
					clients = append(clients, client)
				}
			}
		}

	case BroadcastUser:
		// Broadcast to specific users
		for _, userID := range msg.UserIDs {
			if userClients, ok := h.clients[userID]; ok {
				for client := range userClients {
					if !excludeMap[client.ID] {
						clients = append(clients, client)
					}
				}
			}
		}

	case BroadcastExcept:
		// Broadcast to all except excluded
		for _, userClients := range h.clients {
			for client := range userClients {
				if !excludeMap[client.ID] {
					clients = append(clients, client)
				}
			}
		}
	}

	// Send to all selected clients
	sent := 0
	for _, client := range clients {
		if err := client.SendMessage(msg.Data); err != nil {
			h.log.Debug("Failed to send broadcast message",
				logger.String("client_id", client.ID),
				logger.Error(err),
			)
		} else {
			sent++
		}
	}

	h.messagesSent.Add(int64(sent))

	h.log.Debug("Broadcast complete",
		logger.Int("target_clients", len(clients)),
		logger.Int("sent", sent),
	)
}

// Broadcast sends a message to clients based on target
func (h *Hub) Broadcast(target BroadcastTarget, data interface{}, userIDs []uuid.UUID, exclude ...string) {
	if !h.running.Load() {
		h.log.Warn("Cannot broadcast: hub not running")
		return
	}

	msg := &BroadcastMessage{
		Data:      data,
		Target:    target,
		UserIDs:   userIDs,
		Exclude:   exclude,
		Timestamp: time.Now(),
	}

	select {
	case h.broadcast <- msg:
	case <-time.After(5 * time.Second):
		h.log.Error("Failed to broadcast: timeout")
	}
}

// BroadcastToAll sends a message to all connected clients
func (h *Hub) BroadcastToAll(data interface{}, exclude ...string) {
	h.Broadcast(BroadcastAll, data, nil, exclude...)
}

// BroadcastToUser sends a message to all devices of specific users
func (h *Hub) BroadcastToUser(userIDs []uuid.UUID, data interface{}, exclude ...string) {
	h.Broadcast(BroadcastUser, data, userIDs, exclude...)
}

// SendToUser sends a message to all devices of a user
func (h *Hub) SendToUser(userID uuid.UUID, message interface{}) int {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()

	clients, ok := h.clients[userID]
	if !ok {
		return 0
	}

	sent := 0
	for client := range clients {
		if err := client.SendMessage(message); err != nil {
			h.log.Debug("Failed to send message to user",
				logger.String("client_id", client.ID),
				logger.String("user_id", userID.String()),
				logger.Error(err),
			)
		} else {
			sent++
		}
	}

	return sent
}

// SendToClient sends a message to a specific client by ID
func (h *Hub) SendToClient(clientID string, message interface{}) error {
	h.clientsByIDMu.RLock()
	client, ok := h.clientsByID[clientID]
	h.clientsByIDMu.RUnlock()

	if !ok {
		return ErrClientNotFound
	}

	return client.SendMessage(message)
}

// GetClient returns a client by ID
func (h *Hub) GetClient(clientID string) (*Client, bool) {
	h.clientsByIDMu.RLock()
	defer h.clientsByIDMu.RUnlock()

	client, ok := h.clientsByID[clientID]
	return client, ok
}

// GetUserClients returns all clients for a user
func (h *Hub) GetUserClients(userID uuid.UUID) []*Client {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()

	userClients, ok := h.clients[userID]
	if !ok {
		return nil
	}

	clients := make([]*Client, 0, len(userClients))
	for client := range userClients {
		clients = append(clients, client)
	}

	return clients
}

// IsUserOnline returns true if the user has any active connections
func (h *Hub) IsUserOnline(userID uuid.UUID) bool {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()

	userClients, ok := h.clients[userID]
	return ok && len(userClients) > 0
}

// GetUserDeviceCount returns the number of devices connected for a user
func (h *Hub) GetUserDeviceCount(userID uuid.UUID) int {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()

	if userClients, ok := h.clients[userID]; ok {
		return len(userClients)
	}
	return 0
}

// GetOnlineUsers returns list of online user IDs
func (h *Hub) GetOnlineUsers() []uuid.UUID {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()

	users := make([]uuid.UUID, 0, len(h.clients))
	for userID := range h.clients {
		users = append(users, userID)
	}

	return users
}

// GetConnectionCount returns the total number of active connections
func (h *Hub) GetConnectionCount() int {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()

	count := 0
	for _, userClients := range h.clients {
		count += len(userClients)
	}
	return count
}

// GetUserCount returns the number of unique users connected
func (h *Hub) GetUserCount() int {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()

	return len(h.clients)
}

// GetStats returns hub statistics
func (h *Hub) GetStats() *HubStats {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()

	userDeviceCount := make(map[uuid.UUID]int)
	totalConnections := 0

	for userID, userClients := range h.clients {
		count := len(userClients)
		userDeviceCount[userID] = count
		totalConnections += count
	}

	return &HubStats{
		TotalClients:     totalConnections,
		TotalConnections: int(h.totalConnections.Load()),
		OnlineUsers:      len(h.clients),
		MessagesSent:     h.messagesSent.Load(),
		MessagesReceived: h.messagesReceived.Load(),
		BytesSent:        h.bytesSent.Load(),
		BytesReceived:    h.bytesReceived.Load(),
		Uptime:           time.Since(h.startTime),
		UserDeviceCount:  userDeviceCount,
	}
}

// cleanupStaleConnections removes stale connections
func (h *Hub) cleanupStaleConnections() {
	h.clientsMu.RLock()
	staleClients := make([]*Client, 0)

	for _, userClients := range h.clients {
		for client := range userClients {
			if client.IsStale() {
				staleClients = append(staleClients, client)
			}
		}
	}
	h.clientsMu.RUnlock()

	if len(staleClients) > 0 {
		h.log.Info("Cleaning up stale connections",
			logger.Int("count", len(staleClients)),
		)

		for _, client := range staleClients {
			client.Close()
		}
	}
}

// logMetrics logs hub metrics
func (h *Hub) logMetrics() {
	stats := h.GetStats()

	h.log.Info("Hub metrics",
		logger.Int("total_clients", stats.TotalClients),
		logger.Int("online_users", stats.OnlineUsers),
		logger.Int64("messages_sent", stats.MessagesSent),
		logger.Int64("messages_received", stats.MessagesReceived),
		logger.Int64("bytes_sent", stats.BytesSent),
		logger.Int64("bytes_received", stats.BytesReceived),
		logger.Duration("uptime", stats.Uptime),
	)
}

// Shutdown gracefully shuts down the hub
func (h *Hub) Shutdown() {
	if !h.running.Load() {
		return
	}

	h.log.Info("Shutting down WebSocket Hub")

	close(h.shutdownCh)
}

// shutdown performs the actual shutdown
func (h *Hub) shutdown() {
	if !h.running.CompareAndSwap(true, false) {
		return
	}

	// Stop tickers
	if h.cleanupTicker != nil {
		h.cleanupTicker.Stop()
	}
	if h.metricsTicker != nil {
		h.metricsTicker.Stop()
	}

	// Close all client connections
	h.clientsMu.Lock()
	allClients := make([]*Client, 0)
	for _, userClients := range h.clients {
		for client := range userClients {
			allClients = append(allClients, client)
		}
	}
	h.clientsMu.Unlock()

	h.log.Info("Closing all client connections",
		logger.Int("count", len(allClients)),
	)

	// Close clients concurrently
	var wg sync.WaitGroup
	for _, client := range allClients {
		wg.Add(1)
		go func(c *Client) {
			defer wg.Done()
			c.Close()
		}(client)
	}

	// Wait for all clients to close (with timeout)
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		h.log.Info("All clients closed")
	case <-time.After(10 * time.Second):
		h.log.Warn("Timeout waiting for clients to close")
	}

	// Close channels
	close(h.register)
	close(h.unregister)
	close(h.broadcast)

	// Cancel context
	h.cancel()

	h.log.Info("WebSocket Hub shut down",
		logger.Int64("total_connections", h.totalConnections.Load()),
		logger.Int64("total_disconnections", h.totalDisconnections.Load()),
		logger.Duration("uptime", time.Since(h.startTime)),
	)
}

// IsRunning returns true if the hub is running
func (h *Hub) IsRunning() bool {
	return h.running.Load()
}

// Context returns the hub context
func (h *Hub) Context() context.Context {
	return h.ctx
}
