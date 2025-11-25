package websocket

import (
	"context"
	"encoding/json"
	"time"

	"shared/pkg/logger"
	"shared/server/websocket"
	"shared/server/websocket/connection"
	"shared/server/websocket/hub"
	"shared/server/websocket/pubsub"
	"shared/server/websocket/router"
	"ws-service/internal/protocol"

	"github.com/google/uuid"
)

// Manager manages WebSocket connections for the messaging app
type Manager struct {
	engine *websocket.Engine
	hub    *hub.Hub
	log    logger.Logger

	// Application-specific components (keep these in ws-service)
	subscriptions *SubscriptionManager
	presence      *PresenceTracker
	typing        *TypingManager

	// Message router for application messages
	messageRouter *router.Router
}

// NewManager creates a new WebSocket manager
func NewManager(log logger.Logger) *Manager {
	// Build the engine with required components
	engine := websocket.NewEngineBuilder().
		WithLogger(log).
		WithMaxConnections(10000).
		WithDispatcher(10, 1000).
		WithPubSub().
		WithSessions(24 * time.Hour).
		WithHealthCheck(30 * time.Second).
		Build()

	// Create hub for multi-device management
	hubInstance := hub.New(engine.EventEmitter(), log)

	mgr := &Manager{
		engine:        engine,
		hub:           hubInstance,
		log:           log,
		subscriptions: NewSubscriptionManager(log),
		presence:      NewPresenceTracker(log),
		typing:        NewTypingManager(log),
		messageRouter: router.New(),
	}

	// Register application-specific message handlers
	mgr.registerHandlers()

	// Setup connection lifecycle hooks
	mgr.setupLifecycleHooks()

	return mgr
}

// registerHandlers registers all message type handlers
func (m *Manager) registerHandlers() {
	// Subscription handlers
	m.messageRouter.Register("subscribe", m.handleSubscribe)
	m.messageRouter.Register("unsubscribe", m.handleUnsubscribe)

	// Presence handlers
	m.messageRouter.Register("presence.update", m.handlePresenceUpdate)
	m.messageRouter.Register("presence.query", m.handlePresenceQuery)

	// Typing handlers
	m.messageRouter.Register("typing.start", m.handleTypingStart)
	m.messageRouter.Register("typing.stop", m.handleTypingStop)

	// Read receipt handlers
	m.messageRouter.Register("mark.read", m.handleMarkRead)
	m.messageRouter.Register("mark.delivered", m.handleMarkDelivered)

	// Call signaling handlers
	m.messageRouter.Register("call.offer", m.handleCallOffer)
	m.messageRouter.Register("call.answer", m.handleCallAnswer)
	m.messageRouter.Register("call.ice", m.handleCallICE)
	m.messageRouter.Register("call.hangup", m.handleCallHangup)

	// Ping handler
	m.messageRouter.Register("ping", m.handlePing)
}

// setupLifecycleHooks sets up connection lifecycle hooks
func (m *Manager) setupLifecycleHooks() {
	// On connect callback
	m.engine.ConnectionManager().SetOnConnect(func(conn *connection.Connection) {
		// Get user ID and device ID from metadata
		userIDVal, ok := conn.GetMetadata("user_id")
		if !ok {
			m.log.Error("Connection missing user_id metadata")
			return
		}

		userID, ok := userIDVal.(uuid.UUID)
		if !ok {
			m.log.Error("Invalid user_id type in metadata")
			return
		}

		deviceIDVal, ok := conn.GetMetadata("device_id")
		if !ok {
			m.log.Error("Connection missing device_id metadata")
			return
		}

		deviceID, ok := deviceIDVal.(string)
		if !ok {
			m.log.Error("Invalid device_id type in metadata")
			return
		}

		// Register with hub
		if err := m.hub.Register(userID, deviceID, conn); err != nil {
			m.log.Error("Failed to register connection with hub",
				logger.String("user_id", userID.String()),
				logger.String("device_id", deviceID),
				logger.Error(err),
			)
			return
		}

		// Update presence
		m.presence.OnUserConnected(userID)

		m.log.Info("User connected via WebSocket",
			logger.String("user_id", userID.String()),
			logger.String("device_id", deviceID),
			logger.String("conn_id", conn.ID()),
		)
	})

	// On disconnect callback
	m.engine.ConnectionManager().SetOnDisconnect(func(conn *connection.Connection) {
		// Get user ID and device ID from metadata
		userIDVal, ok := conn.GetMetadata("user_id")
		if !ok {
			return
		}

		userID, ok := userIDVal.(uuid.UUID)
		if !ok {
			return
		}

		deviceIDVal, ok := conn.GetMetadata("device_id")
		if !ok {
			return
		}

		deviceID, ok := deviceIDVal.(string)
		if !ok {
			return
		}

		// Unregister from hub
		m.hub.Unregister(userID, deviceID)

		// Unsubscribe from all topics
		m.subscriptions.UnsubscribeAll(conn.ID())

		// Update presence if user has no more connections
		if !m.hub.IsOnline(userID) {
			m.presence.OnUserDisconnected(userID)
		}

		m.log.Info("User disconnected from WebSocket",
			logger.String("user_id", userID.String()),
			logger.String("device_id", deviceID),
			logger.String("conn_id", conn.ID()),
		)
	})
}

// Start starts the WebSocket manager
func (m *Manager) Start() error {
	return m.engine.Start()
}

// Stop stops the WebSocket manager
func (m *Manager) Stop() error {
	return m.engine.Stop()
}

// HandleMessage handles incoming WebSocket messages
func (m *Manager) HandleMessage(ctx context.Context, conn *connection.Connection, data []byte) error {
	// Parse message
	var msg protocol.ClientMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		m.log.Error("Failed to parse message", logger.Error(err))
		return m.sendError(conn, "", "invalid_json", "Invalid JSON message")
	}

	m.log.Debug("Received message",
		logger.String("conn_id", conn.ID()),
		logger.String("type", msg.Type),
		logger.String("id", msg.ID),
	)

	// Route to handler
	routerMsg := &router.Message{
		Type:     msg.Type,
		Payload:  msg.Payload,
		Metadata: map[string]interface{}{
			"connection": conn,
			"message_id": msg.ID,
		},
	}

	if err := m.messageRouter.Route(ctx, routerMsg); err != nil {
		m.log.Error("Message routing failed",
			logger.String("type", msg.Type),
			logger.Error(err),
		)
		return m.sendError(conn, msg.ID, "routing_failed", err.Error())
	}

	return nil
}

// BroadcastToUser broadcasts a message to all devices of a user
func (m *Manager) BroadcastToUser(userID uuid.UUID, messageType string, payload interface{}) error {
	msg := &pubsub.Message{
		Topic:   "user:" + userID.String(),
		Payload: m.marshalPayload(messageType, payload),
		Metadata: map[string]interface{}{
			"type":    messageType,
			"user_id": userID,
		},
	}

	m.engine.PubSub().Publish(msg)
	return nil
}

// BroadcastToConversation broadcasts to all conversation participants
func (m *Manager) BroadcastToConversation(conversationID uuid.UUID, messageType string, payload interface{}, excludeUserID ...uuid.UUID) error {
	// Get subscribers for this conversation
	subscribers := m.subscriptions.GetSubscribers("conversation:" + conversationID.String())

	for _, connID := range subscribers {
		conn, ok := m.engine.ConnectionManager().Get(connID)
		if !ok {
			continue
		}

		// Skip excluded users
		userIDVal, ok := conn.GetMetadata("user_id")
		if !ok {
			continue
		}

		userID := userIDVal.(uuid.UUID)
		skip := false
		for _, excludeID := range excludeUserID {
			if userID == excludeID {
				skip = true
				break
			}
		}

		if skip {
			continue
		}

		// Send message
		data := m.marshalPayload(messageType, payload)
		conn.Send(data)
	}

	return nil
}

// GetEngine returns the underlying engine for advanced use cases
func (m *Manager) GetEngine() *websocket.Engine {
	return m.engine
}

// GetHub returns the hub for multi-device management
func (m *Manager) GetHub() *hub.Hub {
	return m.hub
}

// sendError sends an error message to a connection
func (m *Manager) sendError(conn *connection.Connection, requestID, code, message string) error {
	errorMsg := protocol.ServerMessage{
		ID:        uuid.New().String(),
		Type:      "error",
		RequestID: requestID,
		Payload: protocol.ErrorPayload{
			Code:    code,
			Message: message,
		},
		Timestamp: time.Now(),
	}

	data, _ := json.Marshal(errorMsg)
	return conn.Send(data)
}

// marshalPayload marshals a payload to JSON
func (m *Manager) marshalPayload(messageType string, payload interface{}) []byte {
	msg := protocol.ServerMessage{
		ID:        uuid.New().String(),
		Type:      messageType,
		Payload:   payload,
		Timestamp: time.Now(),
	}
	data, _ := json.Marshal(msg)
	return data
}
