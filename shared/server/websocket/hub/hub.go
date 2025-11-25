package hub

import (
	"context"
	"sync"

	"shared/pkg/logger"
	"shared/server/websocket/connection"
	"shared/server/websocket/event"

	"github.com/google/uuid"
)

// Client represents a user with multiple device connections
type Client struct {
	UserID      uuid.UUID
	Connections map[string]*connection.Connection // deviceID -> connection
	mu          sync.RWMutex
}

// AddConnection adds a connection to the client
func (c *Client) AddConnection(deviceID string, conn *connection.Connection) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Connections[deviceID] = conn
}

// RemoveConnection removes a connection from the client
func (c *Client) RemoveConnection(deviceID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.Connections, deviceID)
}

// GetConnection retrieves a connection by device ID
func (c *Client) GetConnection(deviceID string) (*connection.Connection, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	conn, exists := c.Connections[deviceID]
	return conn, exists
}

// GetAllConnections returns all connections
func (c *Client) GetAllConnections() []*connection.Connection {
	c.mu.RLock()
	defer c.mu.RUnlock()
	conns := make([]*connection.Connection, 0, len(c.Connections))
	for _, conn := range c.Connections {
		conns = append(conns, conn)
	}
	return conns
}

// ConnectionCount returns the number of active connections
func (c *Client) ConnectionCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.Connections)
}

// Hub manages clients and their multi-device connections
type Hub struct {
	clients       map[uuid.UUID]*Client // userID -> client
	mu            sync.RWMutex
	eventEmitter  *event.Emitter
	log           logger.Logger
	ctx           context.Context
	cancel        context.CancelFunc
}

// New creates a new hub
func New(eventEmitter *event.Emitter, log logger.Logger) *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	return &Hub{
		clients:      make(map[uuid.UUID]*Client),
		eventEmitter: eventEmitter,
		log:          log,
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Register registers a connection for a user
func (h *Hub) Register(userID uuid.UUID, deviceID string, conn *connection.Connection) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	client, exists := h.clients[userID]
	if !exists {
		client = &Client{
			UserID:      userID,
			Connections: make(map[string]*connection.Connection),
		}
		h.clients[userID] = client
	}

	client.AddConnection(deviceID, conn)

	if h.eventEmitter != nil {
		h.eventEmitter.Emit(&event.Event{
			Type: event.EventClientRegistered,
			Data: map[string]interface{}{
				"user_id":   userID,
				"device_id": deviceID,
				"conn_id":   conn.ID(),
			},
		})
	}

	h.log.Info("Client registered",
		logger.String("user_id", userID.String()),
		logger.String("device_id", deviceID),
		logger.String("conn_id", conn.ID()),
	)

	return nil
}

// Unregister removes a connection for a user
func (h *Hub) Unregister(userID uuid.UUID, deviceID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	client, exists := h.clients[userID]
	if !exists {
		return
	}

	client.RemoveConnection(deviceID)

	if client.ConnectionCount() == 0 {
		delete(h.clients, userID)
	}

	if h.eventEmitter != nil {
		h.eventEmitter.Emit(&event.Event{
			Type: event.EventClientUnregistered,
			Data: map[string]interface{}{
				"user_id":   userID,
				"device_id": deviceID,
			},
		})
	}

	h.log.Info("Client unregistered",
		logger.String("user_id", userID.String()),
		logger.String("device_id", deviceID),
	)
}

// GetClient retrieves a client by user ID
func (h *Hub) GetClient(userID uuid.UUID) (*Client, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	client, exists := h.clients[userID]
	return client, exists
}

// GetConnection retrieves a specific connection
func (h *Hub) GetConnection(userID uuid.UUID, deviceID string) (*connection.Connection, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	client, exists := h.clients[userID]
	if !exists {
		return nil, false
	}

	return client.GetConnection(deviceID)
}

// Broadcast sends a message to all connections of a user
func (h *Hub) Broadcast(userID uuid.UUID, data []byte) error {
	client, exists := h.GetClient(userID)
	if !exists {
		return ErrClientNotFound
	}

	conns := client.GetAllConnections()
	for _, conn := range conns {
		if err := conn.Send(data); err != nil {
			h.log.Warn("Failed to send to connection",
				logger.String("user_id", userID.String()),
				logger.String("conn_id", conn.ID()),
				logger.Error(err),
			)
		}
	}

	return nil
}

// BroadcastExcept sends a message to all connections except one
func (h *Hub) BroadcastExcept(userID uuid.UUID, excludeDeviceID string, data []byte) error {
	client, exists := h.GetClient(userID)
	if !exists {
		return ErrClientNotFound
	}

	client.mu.RLock()
	defer client.mu.RUnlock()

	for deviceID, conn := range client.Connections {
		if deviceID == excludeDeviceID {
			continue
		}
		if err := conn.Send(data); err != nil {
			h.log.Warn("Failed to send to connection",
				logger.String("user_id", userID.String()),
				logger.String("conn_id", conn.ID()),
				logger.Error(err),
			)
		}
	}

	return nil
}

// IsOnline checks if a user has any active connections
func (h *Hub) IsOnline(userID uuid.UUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	client, exists := h.clients[userID]
	return exists && client.ConnectionCount() > 0
}

// ClientCount returns the number of clients
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// ConnectionCount returns the total number of connections
func (h *Hub) ConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	count := 0
	for _, client := range h.clients {
		count += client.ConnectionCount()
	}
	return count
}

// GetAllClients returns all clients
func (h *Hub) GetAllClients() []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	clients := make([]*Client, 0, len(h.clients))
	for _, client := range h.clients {
		clients = append(clients, client)
	}
	return clients
}

// Close closes the hub and all connections
func (h *Hub) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.cancel()

	for _, client := range h.clients {
		for _, conn := range client.GetAllConnections() {
			conn.Close()
		}
	}

	h.clients = make(map[uuid.UUID]*Client)
	h.log.Info("Hub closed")

	return nil
}
