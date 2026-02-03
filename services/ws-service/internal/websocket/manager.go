package websocket

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	"shared/pkg/database"
	"shared/pkg/logger"
	"shared/server/websocket"
	"shared/server/websocket/connection"
	"shared/server/websocket/hub"
	"shared/server/websocket/router"
	"ws-service/internal/protocol"
	"ws-service/internal/repo"
	"ws-service/internal/service"
	"ws-service/internal/websocket/middleware"
	"ws-service/internal/websocket/tracker"

	codes "ws-service/internal/domain"

	"github.com/google/uuid"
)

type Manager struct {
	engine *websocket.Engine
	db     *database.Database
	hub    *hub.Hub
	log    logger.Logger
	config *Config

	subscriptions *tracker.SubscriptionManager
	presence      *tracker.PresenceTracker
	typing        *tracker.TypingManager

	messageRouter *router.Router
	rateLimiter   *middleware.RateLimiter
	messageBuffer *middleware.MessageBuffer
	metrics       *Metrics

	// Repositories
	conversationRepo repo.ConversationRepository
	userRepo         repo.UserRepository

	// Services
	authzService      service.AuthorizationService
	deliveryPublisher service.DeliveryEventPublisher

	state          ManagerState
	startTime      time.Time
	messageCounter atomic.Uint64
	errorCounter   atomic.Uint64

	shutdownOnce sync.Once
	stopCh       chan struct{}
	wg           sync.WaitGroup
}

type ManagerState int32

const (
	StateInitialized ManagerState = iota
	StateStarting
	StateRunning
	StateStopping
	StateStopped
)

func NewManager(log logger.Logger, db *database.Database, opts ...Option) *Manager {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	engine := websocket.NewEngineBuilder().
		WithLogger(log).
		WithMaxConnections(cfg.MaxConnections).
		WithDispatcher(cfg.DispatcherWorkers, cfg.DispatcherQueueSize).
		WithPubSub().
		WithSessions(cfg.SessionTTL).
		WithHealthCheck(cfg.HealthCheckInterval).
		Build()

	hubInstance := hub.New(engine.EventEmitter(), log)

	subCfg := &tracker.SubscriptionManagerConfig{
		MaxPerConnection: cfg.MaxSubscriptionsPerConn,
	}

	presenceCfg := &tracker.PresenceTrackerConfig{
		IdleTimeout: 5 * time.Minute,
		AwayTimeout: 15 * time.Minute,
	}

	typingCfg := &tracker.TypingManagerConfig{
		Timeout: cfg.TypingTimeout,
	}

	rateLimiterCfg := &middleware.RateLimiterConfig{
		RequestsPerSecond: cfg.RateLimitPerSecond,
		BurstSize:         cfg.RateLimitBurst,
		CleanupInterval:   cfg.RateLimiterCleanupInterval,
	}

	bufferCfg := &middleware.MessageBufferConfig{
		MaxSize:         cfg.MessageBufferSize,
		MaxRetries:      cfg.MessageRetryAttempts,
		RetryDelay:      cfg.MessageRetryDelay,
		CleanupInterval: 30 * time.Second,
	}

	var metrics *Metrics
	if cfg.MetricsEnabled {
		metrics = NewMetrics(cfg.MetricsPrefix)
	}

	// Initialize repositories
	conversationRepo := repo.NewConversationRepository(*db, log)
	userRepo := repo.NewUserRepository(*db, log)

	// Initialize services
	authzService := service.NewAuthorizationService(conversationRepo, userRepo, log)

	mgr := &Manager{
		engine:           engine,
		db:               db,
		hub:              hubInstance,
		log:              log,
		config:           cfg,
		subscriptions:    tracker.NewSubscriptionManager(log, subCfg),
		presence:         tracker.NewPresenceTracker(log, presenceCfg),
		typing:           tracker.NewTypingManager(log, typingCfg),
		messageRouter:    router.New(),
		rateLimiter:      middleware.NewRateLimiter(rateLimiterCfg),
		messageBuffer:    middleware.NewMessageBuffer(bufferCfg),
		metrics:          metrics,
		conversationRepo: conversationRepo,
		userRepo:         userRepo,
		authzService:     authzService,
		state:            StateInitialized,
		stopCh:           make(chan struct{}),
	}

	mgr.registerHandlers()
	mgr.setupLifecycleHooks()
	mgr.setupTrackerCallbacks()

	return mgr
}

func (m *Manager) Start() error {
	m.state = StateStarting
	m.startTime = time.Now()

	m.wg.Add(2)
	go m.runTypingCleanup()
	go m.runPresenceCleanup()

	if err := m.engine.Start(); err != nil {
		m.state = StateStopped
		return err
	}

	m.state = StateRunning
	m.log.Info("WebSocket manager started",
		logger.Int("max_connections", m.config.MaxConnections),
		logger.Int("dispatcher_workers", m.config.DispatcherWorkers),
	)

	return nil
}

func (m *Manager) Stop() error {
	var err error

	m.shutdownOnce.Do(func() {
		m.state = StateStopping
		m.log.Info("WebSocket manager stopping...")

		close(m.stopCh)

		done := make(chan struct{})
		go func() {
			m.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(m.config.GracefulShutdownTimeout):
			m.log.Warn("Graceful shutdown timeout exceeded")
		}

		m.rateLimiter.Stop()
		m.messageBuffer.Stop()

		err = m.engine.Stop()
		m.state = StateStopped

		m.log.Info("WebSocket manager stopped",
			logger.Int64("total_messages", int64(m.messageCounter.Load())),
			logger.Int64("total_errors", int64(m.errorCounter.Load())),
		)
	})

	return err
}

func (m *Manager) GetEngine() *websocket.Engine {
	return m.engine
}

func (m *Manager) GetHub() *hub.Hub {
	return m.hub
}

func (m *Manager) GetMetrics() *Metrics {
	return m.metrics
}

func (m *Manager) GetConfig() *Config {
	return m.config
}

func (m *Manager) GetState() ManagerState {
	return m.state
}

func (m *Manager) GetUptime() time.Duration {
	if m.startTime.IsZero() {
		return 0
	}
	return time.Since(m.startTime)
}

func (m *Manager) GetPresenceTracker() *tracker.PresenceTracker {
	return m.presence
}

func (m *Manager) GetSubscriptions() *tracker.SubscriptionManager {
	return m.subscriptions
}

func (m *Manager) setupLifecycleHooks() {
	m.engine.ConnectionManager().SetOnConnect(func(conn *connection.Connection) {
		userID, ok := m.getUserID(conn)
		if !ok {
			m.log.Error("Connection missing user_id metadata")
			return
		}

		deviceID, ok := m.getDeviceID(conn)
		if !ok {
			m.log.Error("Connection missing device_id metadata")
			return
		}

		platform := m.getPlatform(conn)

		if err := m.hub.Register(userID, deviceID, conn); err != nil {
			m.log.Error("Failed to register connection with hub",
				logger.String("user_id", userID.String()),
				logger.String("device_id", deviceID),
				logger.Error(err),
			)
			return
		}

		m.presence.OnUserConnected(userID, deviceID, platform)

		if m.metrics != nil {
			m.metrics.OnConnect(platform)
		}

		m.log.Info("User connected via WebSocket",
			logger.String("user_id", userID.String()),
			logger.String("device_id", deviceID),
			logger.String("conn_id", conn.ID()),
			logger.String("platform", platform),
		)

		m.sendConnectionStatus(conn, "connected", "")
	})

	m.engine.ConnectionManager().SetOnDisconnect(func(conn *connection.Connection) {
		m.rateLimiter.Remove(conn.ID())
		m.messageBuffer.ClearConnection(conn.ID())

		userID, ok := m.getUserID(conn)
		if !ok {
			return
		}

		deviceID, _ := m.getDeviceID(conn)
		platform := m.getPlatform(conn)

		m.hub.Unregister(userID, deviceID)
		m.subscriptions.UnsubscribeAll(conn.ID())
		m.typing.StopTypingForUser(userID)
		m.presence.OnUserDisconnected(userID, deviceID)

		if m.metrics != nil {
			m.metrics.OnDisconnect(platform)
		}

		m.log.Info("User disconnected from WebSocket",
			logger.String("user_id", userID.String()),
			logger.String("device_id", deviceID),
			logger.String("conn_id", conn.ID()),
		)
	})
}

func (m *Manager) setupTrackerCallbacks() {
	m.presence.OnChange(func(event tracker.PresenceChangeEvent) {
		m.BroadcastPresenceChange(event)
	})

	m.typing.OnTypingEvent(func(event tracker.TypingEvent) {
		m.BroadcastTypingEvent(event)
	})
}

func (m *Manager) registerHandlers() {
	m.messageRouter.Register(string(protocol.MsgTypeSubscribe), m.handleSubscribe)
	m.messageRouter.Register(string(protocol.MsgTypeUnsubscribe), m.handleUnsubscribe)
	m.messageRouter.Register(string(protocol.MsgTypePresenceUpdate), m.handlePresenceUpdate)
	m.messageRouter.Register(string(protocol.MsgTypePresenceQuery), m.handlePresenceQuery)
	m.messageRouter.Register(string(protocol.MsgTypeTypingStart), m.handleTypingStart)
	m.messageRouter.Register(string(protocol.MsgTypeTypingStop), m.handleTypingStop)
	m.messageRouter.Register(string(protocol.MsgTypeMarkRead), m.handleMarkRead)
	m.messageRouter.Register(string(protocol.MsgTypeMarkDelivered), m.handleMarkDelivered)
	m.messageRouter.Register(string(protocol.MsgTypeCallOffer), m.handleCallOffer)
	m.messageRouter.Register(string(protocol.MsgTypeCallAnswer), m.handleCallAnswer)
	m.messageRouter.Register(string(protocol.MsgTypeCallICE), m.handleCallICE)
	m.messageRouter.Register(string(protocol.MsgTypeCallHangup), m.handleCallHangup)
	m.messageRouter.Register(string(protocol.MsgTypePing), m.handlePing)
}

func (m *Manager) HandleMessage(ctx context.Context, conn *connection.Connection, data []byte) error {
	startTime := time.Now()
	m.messageCounter.Add(1)

	if !m.rateLimiter.Allow(conn.ID()) {
		m.log.Warn("Rate limit exceeded", logger.String("conn_id", conn.ID()))
		if m.metrics != nil {
			m.metrics.OnRateLimited()
		}
		return m.sendError(conn, "", protocol.ErrCodeRateLimited, "Too many messages, please slow down")
	}

	var msg protocol.ClientMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		m.errorCounter.Add(1)
		m.log.Error("Failed to parse message", logger.Error(err))
		if m.metrics != nil {
			m.metrics.OnError(string(protocol.ErrCodeInvalidJSON))
		}
		return m.sendError(conn, "", protocol.ErrCodeInvalidJSON, "Invalid JSON message")
	}

	if m.metrics != nil {
		m.metrics.OnMessage(msg.Type, int64(len(data)))
	}

	m.log.Debug("Received message",
		logger.String("conn_id", conn.ID()),
		logger.String("type", msg.Type),
		logger.String("id", msg.ID),
	)

	routerMsg := &router.Message{
		Type:    msg.Type,
		Payload: msg.Payload,
		Metadata: map[string]any{
			"connection": conn,
			"message_id": msg.ID,
			"timestamp":  msg.Timestamp,
		},
	}

	if err := m.messageRouter.Route(ctx, routerMsg); err != nil {
		m.errorCounter.Add(1)
		m.log.Error("Message routing failed",
			logger.String("type", msg.Type),
			logger.Error(err),
		)
		if m.metrics != nil {
			m.metrics.OnError(string(protocol.ErrCodeRoutingFailed))
		}
		return m.sendError(conn, msg.ID, protocol.ErrCodeRoutingFailed, err.Error())
	}

	if m.metrics != nil {
		m.metrics.RecordLatency(time.Since(startTime).Nanoseconds())
	}

	return nil
}

func (m *Manager) runTypingCleanup() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.TypingCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.typing.Cleanup()
		case <-m.stopCh:
			return
		}
	}
}

func (m *Manager) runPresenceCleanup() {
	defer m.wg.Done()

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.presence.CleanupStale()
		case <-m.stopCh:
			return
		}
	}
}

func (m *Manager) getUserID(conn *connection.Connection) (uuid.UUID, bool) {
	userIDVal, ok := conn.GetMetadata(codes.UserIdMetaKey)
	if !ok {
		return uuid.Nil, false
	}
	userID, ok := userIDVal.(uuid.UUID)
	return userID, ok
}

func (m *Manager) getDeviceID(conn *connection.Connection) (string, bool) {
	deviceIDVal, ok := conn.GetMetadata(codes.DeviceIdMetaKey)
	if !ok {
		return "", false
	}
	deviceID, ok := deviceIDVal.(string)
	return deviceID, ok
}

func (m *Manager) getPlatform(conn *connection.Connection) string {
	platformVal, ok := conn.GetMetadata("platform")
	if !ok {
		return "unknown"
	}
	platform, _ := platformVal.(string)
	return platform
}

func (m *Manager) sendResponse(conn *connection.Connection, requestID string, msgType protocol.MessageType, payload any) error {
	msg := protocol.NewServerMessage(msgType, requestID).WithPayload(payload)
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	if m.metrics != nil {
		m.metrics.OnMessageSent(int64(len(data)))
	}

	return conn.Send(data)
}

func (m *Manager) sendError(conn *connection.Connection, requestID string, code protocol.ErrorCode, message string) error {
	errorMsg := protocol.NewErrorMessage(requestID, code, message)
	data, _ := json.Marshal(errorMsg)
	return conn.Send(data)
}

func (m *Manager) sendErrorWithDetails(conn *connection.Connection, requestID string, code protocol.ErrorCode, message string, details map[string]any) error {
	errorMsg := protocol.NewErrorMessageWithDetails(requestID, code, message, details)
	data, _ := json.Marshal(errorMsg)
	return conn.Send(data)
}

func (m *Manager) sendConnectionStatus(conn *connection.Connection, status, reason string) {
	event := protocol.ConnectionStatusEvent{
		Status:    status,
		Reason:    reason,
		Timestamp: time.Now(),
	}

	m.sendResponse(conn, "", protocol.MsgTypeConnectionStatus, event)
}

func (m *Manager) IsUserOnline(userID uuid.UUID) bool {
	return m.presence.IsOnline(userID)
}

func (m *Manager) GetOnlineUsers() []uuid.UUID {
	return m.presence.GetOnlineUsers()
}

func (m *Manager) GetStats() ManagerStats {
	stats := ManagerStats{
		State:             m.state,
		Uptime:            m.GetUptime(),
		TotalMessages:     m.messageCounter.Load(),
		TotalErrors:       m.errorCounter.Load(),
		SubscriptionStats: m.subscriptions.GetStats(),
		PresenceStats:     m.presence.GetStats(),
		TypingStats:       m.typing.GetStats(),
		RateLimiterStats:  m.rateLimiter.GetStats(),
		BufferStats:       m.messageBuffer.GetStats(),
	}

	if m.metrics != nil {
		stats.Metrics = m.metrics.GetSnapshot()
	}

	return stats
}

type ManagerStats struct {
	State             ManagerState                  `json:"state"`
	Uptime            time.Duration                 `json:"uptime"`
	TotalMessages     uint64                        `json:"total_messages"`
	TotalErrors       uint64                        `json:"total_errors"`
	SubscriptionStats tracker.SubscriptionStats     `json:"subscription_stats"`
	PresenceStats     tracker.PresenceStats         `json:"presence_stats"`
	TypingStats       tracker.TypingStats           `json:"typing_stats"`
	RateLimiterStats  middleware.RateLimiterStats   `json:"rate_limiter_stats"`
	BufferStats       middleware.MessageBufferStats `json:"buffer_stats"`
	Metrics           MetricsSnapshot               `json:"metrics,omitzero"`
}

// SetDeliveryPublisher sets the delivery event publisher for receipt tracking
func (m *Manager) SetDeliveryPublisher(publisher service.DeliveryEventPublisher) {
	m.deliveryPublisher = publisher
}
