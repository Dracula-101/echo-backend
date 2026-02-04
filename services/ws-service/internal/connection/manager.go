package connection

import (
	"sync"

	"shared/pkg/logger"

	"github.com/google/uuid"
)

type LocalManager struct {
	connections     map[string]Connection
	userConnections map[uuid.UUID][]string
	mu              sync.RWMutex

	onConnectHooks    []Hook
	onDisconnectHooks []Hook
	hooksMu           sync.RWMutex

	presence      PresenceNotifier
	subscriptions SubscriptionCleaner
	log           logger.Logger
}

func NewLocalManager(
	presence PresenceNotifier,
	subscriptions SubscriptionCleaner,
	log logger.Logger,
) *LocalManager {
	return &LocalManager{
		connections:     make(map[string]Connection),
		userConnections: make(map[uuid.UUID][]string),
		presence:        presence,
		subscriptions:   subscriptions,
		log:             log,
	}
}

func (m *LocalManager) OnConnect(conn Connection) error {
	m.mu.Lock()
	m.connections[conn.ID()] = conn
	if conn.IsAuthenticated() {
		userID := conn.UserID()
		m.userConnections[userID] = append(m.userConnections[userID], conn.ID())
	}
	m.mu.Unlock()

	if conn.IsAuthenticated() && m.presence != nil {
		if err := m.presence.SetOnline(conn.UserID(), conn.DeviceID(), conn.Platform()); err != nil {
			m.log.Error("failed to set user online",
				logger.String("user_id", conn.UserID().String()),
				logger.Error(err),
			)
		}
	}

	m.hooksMu.RLock()
	hooks := m.onConnectHooks
	m.hooksMu.RUnlock()

	for _, hook := range hooks {
		hook(conn)
	}

	m.log.Info("connection established",
		logger.String("conn_id", conn.ID()),
		logger.String("user_id", conn.UserID().String()),
		logger.String("device_id", conn.DeviceID()),
	)

	return nil
}

func (m *LocalManager) OnDisconnect(conn Connection) error {
	m.mu.Lock()
	delete(m.connections, conn.ID())
	if conn.IsAuthenticated() {
		userID := conn.UserID()
		connIDs := m.userConnections[userID]
		for i, id := range connIDs {
			if id == conn.ID() {
				m.userConnections[userID] = append(connIDs[:i], connIDs[i+1:]...)
				break
			}
		}
		if len(m.userConnections[userID]) == 0 {
			delete(m.userConnections, userID)
		}
	}
	m.mu.Unlock()

	if m.subscriptions != nil {
		if _, err := m.subscriptions.UnsubscribeAll(conn.ID()); err != nil {
			m.log.Error("failed to cleanup subscriptions",
				logger.String("conn_id", conn.ID()),
				logger.Error(err),
			)
		}
	}

	if conn.IsAuthenticated() && m.presence != nil {
		if err := m.presence.SetOffline(conn.UserID(), conn.DeviceID()); err != nil {
			m.log.Error("failed to set user offline",
				logger.String("user_id", conn.UserID().String()),
				logger.Error(err),
			)
		}
	}

	m.hooksMu.RLock()
	hooks := m.onDisconnectHooks
	m.hooksMu.RUnlock()

	for _, hook := range hooks {
		hook(conn)
	}

	m.log.Info("connection closed",
		logger.String("conn_id", conn.ID()),
		logger.String("user_id", conn.UserID().String()),
	)

	return nil
}

func (m *LocalManager) GetConnection(connID string) (Connection, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	conn, ok := m.connections[connID]
	return conn, ok
}

func (m *LocalManager) GetUserConnections(userID uuid.UUID) []Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()

	connIDs := m.userConnections[userID]
	conns := make([]Connection, 0, len(connIDs))
	for _, connID := range connIDs {
		if conn, ok := m.connections[connID]; ok {
			conns = append(conns, conn)
		}
	}
	return conns
}

func (m *LocalManager) SendToUser(userID uuid.UUID, data []byte) error {
	conns := m.GetUserConnections(userID)
	for _, conn := range conns {
		conn.Send(data)
	}
	return nil
}

func (m *LocalManager) AddOnConnect(fn Hook) {
	m.hooksMu.Lock()
	defer m.hooksMu.Unlock()
	m.onConnectHooks = append(m.onConnectHooks, fn)
}

func (m *LocalManager) AddOnDisconnect(fn Hook) {
	m.hooksMu.Lock()
	defer m.hooksMu.Unlock()
	m.onDisconnectHooks = append(m.onDisconnectHooks, fn)
}
