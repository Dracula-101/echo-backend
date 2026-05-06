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
	internalRedis "ws-service/internal/redis"
	"ws-service/internal/repo"
	"ws-service/internal/service"
	"ws-service/websocket/middleware"
	"ws-service/websocket/tracker"

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
	wsService         service.WSService

	// Cross-instance fan-out (set via SetPubSub; nil ⇒ single-instance mode)
	pubsub     *internalRedis.PubSub
	instanceID string

	// Reconnect buffer + Postgres catch-up (P1.2)
	pendingStore *internalRedis.PendingStore
	deliveryRepo repo.DeliveryStatusRepository

	// Resume tokens + per-conversation seq (P1.3)
	seqCounter        *internalRedis.SeqCounter
	resumeTokenSecret []byte

	// Drain flag (P1.5) — atomic read in the upgrade path so new connections
	// get rejected with 503 once shutdown begins.
	draining atomic.Bool

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

func (m *Manager) SetWSService(svc service.WSService) {
	m.wsService = svc
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

		// Auto-subscribe to all user's conversations and send sync metadata
		go m.autoSubscribeAndSync(conn, userID)
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
		userStillOnline := m.hub.IsOnline(userID)

		m.subscriptions.UnsubscribeAll(conn.ID())
		m.typing.StopTypingForUser(userID)
		m.presence.OnUserDisconnected(userID, deviceID)

		if m.wsService != nil {
			ctx := context.Background()
			m.wsService.HandleClientDisconnect(ctx, userID, deviceID, userStillOnline)
		}

		if m.metrics != nil {
			m.metrics.OnDisconnect(platform)
		}

		m.log.Info("User disconnected from WebSocket",
			logger.String("user_id", userID.String()),
			logger.String("device_id", deviceID),
			logger.String("conn_id", conn.ID()),
			logger.Bool("user_still_online", userStillOnline),
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
	m.messageRouter.Register(string(protocol.MsgTypeRefreshToken), m.handleRefreshToken)
}

// handleRefreshToken accepts an inbound refresh.token frame carrying a freshly
// minted access token. We don't re-validate against auth-service here — the
// HTTP gateway path will reject any future request with a stale token, so the
// only state we need to update is the connection metadata for any later
// validation paths. Refresh logic itself lives in the REST API.
func (m *Manager) handleRefreshToken(_ context.Context, msg *router.Message) error {
	conn, ok := m.getConnection(msg)
	if !ok {
		return nil
	}

	var payload protocol.RefreshTokenPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}
	if payload.AccessToken == "" {
		return m.sendError(conn, m.getRequestID(msg), protocol.ErrCodeInvalidPayload, "access_token is required")
	}

	conn.SetMetadata("access_token", payload.AccessToken)
	m.log.Info("refreshed connection access token",
		logger.String("conn_id", conn.ID()),
	)
	return nil
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
		m.log.Error("Failed to parse message",
			logger.String("conn_id", conn.ID()),
			logger.String("raw_data", string(data)),
			logger.Error(err),
		)
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
		Status:     status,
		Reason:     reason,
		Timestamp:  time.Now(),
		InstanceID: m.instanceID,
	}

	if status == "connected" && len(m.resumeTokenSecret) > 0 {
		userID, hasUser := m.getUserID(conn)
		deviceID, _ := m.getDeviceID(conn)
		if hasUser {
			tok := protocol.NewResumeToken(userID.String(), deviceID, nil)
			if signed, err := protocol.SignResumeToken(tok, m.resumeTokenSecret); err == nil {
				event.ResumeToken = signed
			}
		}
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

// SetPubSub wires the cross-instance Redis pub/sub. Without this the manager
// only delivers to clients connected to the local pod — fine for dev, broken
// for any multi-instance deployment because Kafka partitions are per-pod.
func (m *Manager) SetPubSub(pubsub *internalRedis.PubSub, instanceID string) {
	m.pubsub = pubsub
	m.instanceID = instanceID
	pubsub.OnMessage(internalRedis.ChannelBroadcast, m.handleRelayBroadcast)
}

// SetPendingStore enables reconnect-buffer behavior: messages targeted at
// users with no live connection are stored briefly so a fast reconnect can
// replay them without touching Postgres.
func (m *Manager) SetPendingStore(store *internalRedis.PendingStore) {
	m.pendingStore = store
}

// SetDeliveryRepo wires the delivery_status repo used for Postgres catch-up
// on reconnect (P1.2) and the safety-net insert in the consumer (P1.1).
func (m *Manager) SetDeliveryRepo(r repo.DeliveryStatusRepository) {
	m.deliveryRepo = r
}

// SetSeqCounter wires per-conversation sequence numbering. Used by the chat
// consumer to stamp each frame with a monotonic seq the client can use to
// detect gaps after a reconnect.
func (m *Manager) SetSeqCounter(s *internalRedis.SeqCounter) {
	m.seqCounter = s
}

func (m *Manager) SeqCounter() *internalRedis.SeqCounter {
	return m.seqCounter
}

// SetResumeTokenSecret installs the HMAC key used to sign resume tokens.
// Without it, no resume token is issued on connect; clients fall back to the
// time-windowed catch-up.
func (m *Manager) SetResumeTokenSecret(secret []byte) {
	m.resumeTokenSecret = secret
}

func (m *Manager) InstanceID() string {
	return m.instanceID
}

// autoSubscribeAndSync subscribes the connection to all user's conversations
// and sends sync metadata for conversations with unread messages.
// Runs in a goroutine to avoid blocking the connect flow.
func (m *Manager) autoSubscribeAndSync(conn *connection.Connection, userID uuid.UUID) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Auto-subscribe to all active conversations
	conversations, err := m.conversationRepo.GetUserConversations(ctx, userID)
	if err != nil {
		m.log.Error("Failed to get user conversations for auto-subscribe",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return
	}

	for _, convID := range conversations {
		topicKey := "conversation:" + convID.String()
		m.subscriptions.Subscribe(conn.ID(), topicKey)
	}

	m.log.Info("Auto-subscribed user to conversations",
		logger.String("user_id", userID.String()),
		logger.String("conn_id", conn.ID()),
		logger.Int("conversation_count", len(conversations)),
	)

	// Drain reconnect buffer first — these are the freshest, in-order frames
	// that arrived while the user was momentarily offline.
	m.drainPendingForUser(ctx, conn, userID)

	// Catch-up from Postgres for messages older than the pending TTL.
	m.sendCatchupFromPostgres(ctx, conn, userID)

	// Send sync metadata for conversations with unread messages
	syncStates, err := m.conversationRepo.GetUserConversationSyncState(ctx, userID)
	if err != nil {
		m.log.Error("Failed to get conversation sync state",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return
	}

	if len(syncStates) == 0 {
		return
	}

	convSyncInfos := make([]protocol.ConversationSyncInfo, 0, len(syncStates))
	for _, state := range syncStates {
		info := protocol.ConversationSyncInfo{
			ConversationID: state.ConversationID.String(),
			UnreadCount:    state.UnreadCount,
		}
		if state.LastMessageID != nil {
			info.LastMessageID = state.LastMessageID.String()
		}
		if state.LastMessageAt != nil {
			info.LastMessageAt = state.LastMessageAt.UTC().Format(time.RFC3339)
		}
		convSyncInfos = append(convSyncInfos, info)
	}

	syncMsg := protocol.NewServerMessage(protocol.MsgTypeSyncRequired, "").
		WithPayload(protocol.SyncRequiredPayload{
			Conversations: convSyncInfos,
		})

	data, err := json.Marshal(syncMsg)
	if err != nil {
		m.log.Error("Failed to marshal sync.required message",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return
	}

	if err := conn.Send(data); err != nil {
		m.log.Error("Failed to send sync.required message",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return
	}

	m.log.Info("Sent sync.required to user",
		logger.String("user_id", userID.String()),
		logger.Int("conversations_with_unread", len(syncStates)),
	)
}

// drainPendingForUser sends out any pre-buffered frames for this user. The
// frames are already JSON-encoded ServerMessages, so we can write them to
// the connection unmodified.
func (m *Manager) drainPendingForUser(ctx context.Context, conn *connection.Connection, userID uuid.UUID) {
	if m.pendingStore == nil {
		return
	}
	frames, err := m.pendingStore.Drain(ctx, userID.String())
	if err != nil {
		m.log.Warn("failed to drain pending buffer",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return
	}
	for _, frame := range frames {
		if err := conn.Send(frame); err != nil {
			m.log.Warn("failed to send pending frame",
				logger.String("user_id", userID.String()),
				logger.Error(err),
			)
			return
		}
	}
	if len(frames) > 0 {
		m.log.Info("drained pending frames on reconnect",
			logger.String("user_id", userID.String()),
			logger.Int("frame_count", len(frames)),
		)
	}
}

// sendCatchupFromPostgres queries delivery_status for messages still marked
// 'sent' and pushes them as a single catchup frame.
func (m *Manager) sendCatchupFromPostgres(ctx context.Context, conn *connection.Connection, userID uuid.UUID) {
	if m.deliveryRepo == nil {
		return
	}

	const catchupLimit = 500
	undelivered, err := m.deliveryRepo.GetUndelivered(ctx, userID, catchupLimit)
	if err != nil {
		m.log.Warn("catch-up query failed",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return
	}
	if len(undelivered) == 0 {
		return
	}

	payload := protocol.CatchupPayload{
		Messages: make([]protocol.CatchupMessage, 0, len(undelivered)),
		HasMore:  len(undelivered) >= catchupLimit,
	}
	for _, m := range undelivered {
		payload.Messages = append(payload.Messages, protocol.CatchupMessage{
			MessageID:      m.MessageID.String(),
			ConversationID: m.ConversationID.String(),
			SenderUserID:   m.SenderUserID.String(),
			Content:        json.RawMessage(m.Content),
			CreatedAt:      m.CreatedAt,
		})
	}

	msg := protocol.NewServerMessage(protocol.MsgTypeCatchup, "").WithPayload(payload)
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	conn.Send(data)
	m.log.Info("sent Postgres catchup on reconnect",
		logger.String("user_id", userID.String()),
		logger.Int("message_count", len(undelivered)),
		logger.Bool("has_more", payload.HasMore),
	)
}

// IsDraining reports whether the manager has been put into draining state.
// The HTTP upgrade path checks this and rejects new connections with 503.
func (m *Manager) IsDraining() bool { return m.draining.Load() }

// SetDraining flips the drain flag. After this returns true reads, new
// connection upgrades should be refused.
func (m *Manager) SetDraining(v bool) { m.draining.Store(v) }

// BroadcastShutdown asks all locally-connected clients to reconnect after a
// short delay. The client picks a different pod (consistent-hash LB plus the
// drain key set in Redis means traffic shifts away). The connections aren't
// force-closed here — that happens at hard-shutdown deadline. This separation
// gives well-behaved clients time to reconnect cleanly.
func (m *Manager) BroadcastShutdown(reconnectAfterMs int, reason string) {
	payload := protocol.ServerShutdownPayload{
		ReconnectAfterMs: reconnectAfterMs,
		Reason:           reason,
	}
	data := m.marshalPayload(string(protocol.MsgTypeServerShutdown), payload)

	for _, client := range m.hub.GetAllClients() {
		for _, conn := range client.GetAllConnections() {
			conn.Send(data)
		}
	}
	m.log.Info("broadcast server.shutdown to all local clients",
		logger.Int("reconnect_after_ms", reconnectAfterMs),
		logger.String("reason", reason),
	)
}

// RevokeUserSessions sends a session.revoked frame to every active connection
// of the given user and closes them. sessionID is informational — until
// auth-service includes device_id in the payload, we can't scope to a single
// connection, so all of the user's devices are kicked.
func (m *Manager) RevokeUserSessions(userID uuid.UUID, sessionID, reason string) {
	client, ok := m.hub.GetClient(userID)
	if !ok {
		// Not connected to this instance. Relay so the owning instance handles it.
		if m.pubsub != nil {
			payload := protocol.SessionRevokedPayload{
				Reason:  reason,
				Message: "Your session was revoked.",
			}
			data := m.marshalPayload(string(protocol.MsgTypeSessionRevoked), payload)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = m.pubsub.PublishBroadcast(ctx, &internalRedis.BroadcastMessage{
				Type:       "user",
				TargetType: "user",
				TargetID:   userID.String(),
				Payload:    data,
			})
		}
		return
	}

	payload := protocol.SessionRevokedPayload{
		Reason:  reason,
		Message: "Your session was revoked.",
	}
	data := m.marshalPayload(string(protocol.MsgTypeSessionRevoked), payload)
	for _, conn := range client.GetAllConnections() {
		conn.Send(data)
		conn.Close()
	}

	m.log.Info("revoked user sessions",
		logger.String("user_id", userID.String()),
		logger.String("session_id", sessionID),
		logger.String("reason", reason),
	)
}

// bufferIfOffline pushes the marshalled frame to the user's pending list
// when there's no live connection anywhere in the cluster. Caller already
// confirmed local connection lookup miss; this also checks the Redis-backed
// user-conns set to know if the user is on another pod.
func (m *Manager) bufferIfOffline(userID uuid.UUID, data []byte) {
	if m.pendingStore == nil {
		return
	}
	if m.hub.IsOnline(userID) {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = m.pendingStore.Push(ctx, userID.String(), data)
}
