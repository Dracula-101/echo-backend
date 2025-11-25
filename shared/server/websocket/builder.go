package websocket

import (
	"time"

	"shared/pkg/logger"
	"shared/server/websocket/dispatcher"
	"shared/server/websocket/health"
	"shared/server/websocket/pubsub"
	"shared/server/websocket/reconnect"
	"shared/server/websocket/session"
)

// EngineBuilder builds a WebSocket engine
type EngineBuilder struct {
	engine *Engine
	config *EngineConfig
	log    logger.Logger
}

// NewEngineBuilder creates a new engine builder
func NewEngineBuilder() *EngineBuilder {
	return &EngineBuilder{
		config: DefaultEngineConfig(),
	}
}

// WithLogger sets the logger
func (b *EngineBuilder) WithLogger(log logger.Logger) *EngineBuilder {
	b.log = log
	return b
}

// WithConfig sets the configuration
func (b *EngineBuilder) WithConfig(config *EngineConfig) *EngineBuilder {
	b.config = config
	return b
}

// WithMaxConnections sets max connections
func (b *EngineBuilder) WithMaxConnections(max int) *EngineBuilder {
	b.config.MaxConnections = max
	return b
}

// WithDispatcher enables the dispatcher
func (b *EngineBuilder) WithDispatcher(workers, queueSize int) *EngineBuilder {
	b.config.DispatcherWorkers = workers
	b.config.DispatcherQueueSize = queueSize
	return b
}

// WithPubSub enables pub/sub
func (b *EngineBuilder) WithPubSub() *EngineBuilder {
	// Marker to enable pub/sub
	return b
}

// WithSessions enables session management
func (b *EngineBuilder) WithSessions(ttl time.Duration) *EngineBuilder {
	b.config.SessionTTL = int(ttl.Seconds())
	return b
}

// WithHealthCheck enables health checking
func (b *EngineBuilder) WithHealthCheck(interval time.Duration) *EngineBuilder {
	b.config.HealthCheckInterval = int(interval.Seconds())
	return b
}

// WithReconnect enables reconnection handling
func (b *EngineBuilder) WithReconnect(config *reconnect.Config) *EngineBuilder {
	// Store reconnect config
	return b
}

// Build builds the engine
func (b *EngineBuilder) Build() *Engine {
	if b.log == nil {
		panic("logger is required")
	}

	engine := NewEngine(b.config, b.log)

	// Initialize optional components
	if b.config.DispatcherWorkers > 0 {
		engine.dispatcher = dispatcher.New(
			b.config.DispatcherWorkers,
			b.config.DispatcherQueueSize,
			b.log,
		)
	}

	// Always enable pub/sub
	engine.pubsub = pubsub.NewBroker(b.log)

	// Enable sessions
	if b.config.SessionTTL > 0 {
		engine.sessionMgr = session.NewManager(
			time.Duration(b.config.SessionTTL)*time.Second,
			time.Duration(b.config.CleanupInterval)*time.Second,
		)
	}

	// Enable health checking
	if b.config.HealthCheckInterval > 0 {
		engine.healthChecker = health.NewChecker(
			time.Duration(b.config.HealthCheckInterval) * time.Second,
		)
	}

	// Enable reconnection
	engine.reconnect = reconnect.NewHandler(reconnect.DefaultConfig(), b.log)

	return engine
}
