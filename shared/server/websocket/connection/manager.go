package connection

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"shared/pkg/logger"
)

// Manager manages all active connections
type Manager struct {
	connections   map[string]*Connection
	mu            sync.RWMutex
	maxConnections int32
	currentCount  atomic.Int32

	// Lifecycle hooks
	onConnect    func(*Connection)
	onDisconnect func(*Connection)

	// Cleanup
	cleanupInterval time.Duration
	stopCleanup     chan struct{}

	log logger.Logger
}

// NewManager creates a new connection manager
func NewManager(maxConnections int, cleanupInterval time.Duration, log logger.Logger) *Manager {
	return &Manager{
		connections:     make(map[string]*Connection),
		maxConnections:  int32(maxConnections),
		cleanupInterval: cleanupInterval,
		stopCleanup:     make(chan struct{}),
		log:             log,
	}
}

// Add adds a connection to the manager
func (m *Manager) Add(conn *Connection) error {
	if m.maxConnections > 0 && m.currentCount.Load() >= m.maxConnections {
		return ErrMaxConnectionsReached
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.connections[conn.ID()] = conn
	m.currentCount.Add(1)

	if m.onConnect != nil {
		m.onConnect(conn)
	}

	m.log.Info("Connection added", logger.String("conn_id", conn.ID()))
	return nil
}

// Remove removes a connection from the manager
func (m *Manager) Remove(connID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if conn, exists := m.connections[connID]; exists {
		delete(m.connections, connID)
		m.currentCount.Add(-1)

		if m.onDisconnect != nil {
			m.onDisconnect(conn)
		}

		m.log.Info("Connection removed", logger.String("conn_id", connID))
	}
}

// Get retrieves a connection by ID
func (m *Manager) Get(connID string) (*Connection, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conn, exists := m.connections[connID]
	return conn, exists
}

// GetAll returns all connections
func (m *Manager) GetAll() []*Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conns := make([]*Connection, 0, len(m.connections))
	for _, conn := range m.connections {
		conns = append(conns, conn)
	}
	return conns
}

// Count returns the current number of connections
func (m *Manager) Count() int {
	return int(m.currentCount.Load())
}

// StartCleanup starts the cleanup goroutine
func (m *Manager) StartCleanup(ctx context.Context) {
	ticker := time.NewTicker(m.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCleanup:
			return
		case <-ticker.C:
			m.cleanup()
		}
	}
}

// StopCleanup stops the cleanup goroutine
func (m *Manager) StopCleanup() {
	close(m.stopCleanup)
}

// cleanup removes stale connections
func (m *Manager) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	stale := make([]string, 0)
	for id, conn := range m.connections {
		if conn.IsStale() {
			stale = append(stale, id)
		}
	}

	for _, id := range stale {
		if conn, exists := m.connections[id]; exists {
			conn.Close()
			delete(m.connections, id)
			m.currentCount.Add(-1)
		}
	}

	if len(stale) > 0 {
		m.log.Info("Cleaned up stale connections", logger.Int("count", len(stale)))
	}
}

// SetOnConnect sets the connection callback
func (m *Manager) SetOnConnect(fn func(*Connection)) {
	m.onConnect = fn
}

// SetOnDisconnect sets the disconnection callback
func (m *Manager) SetOnDisconnect(fn func(*Connection)) {
	m.onDisconnect = fn
}

// CloseAll closes all connections
func (m *Manager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, conn := range m.connections {
		conn.Close()
	}

	m.connections = make(map[string]*Connection)
	m.currentCount.Store(0)
}
