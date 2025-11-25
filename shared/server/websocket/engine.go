package websocket

import (
	"context"
	"sync/atomic"
	"time"

	"shared/pkg/logger"
	"shared/server/websocket/connection"
	"shared/server/websocket/dispatcher"
	"shared/server/websocket/event"
	"shared/server/websocket/health"
	"shared/server/websocket/lifecycle"
	"shared/server/websocket/metrics"
	"shared/server/websocket/pubsub"
	"shared/server/websocket/reconnect"
	"shared/server/websocket/registry"
	"shared/server/websocket/router"
	"shared/server/websocket/session"
)

// Engine is the main WebSocket engine that coordinates all components
type Engine struct {
	// Core components
	connManager *connection.Manager
	router      *router.Router
	dispatcher  *dispatcher.Dispatcher
	pubsub      *pubsub.Broker
	sessionMgr  *session.Manager

	// Supporting components
	eventEmitter  *event.Emitter
	lifecycle     *lifecycle.Manager
	healthChecker *health.Checker
	metrics       *metrics.Collector
	reconnect     *reconnect.Handler
	registry      *registry.Registry

	// State
	running atomic.Bool

	// Configuration
	config *EngineConfig

	// Logger
	log logger.Logger

	// Context
	ctx    context.Context
	cancel context.CancelFunc
}

// EngineConfig holds engine configuration
type EngineConfig struct {
	MaxConnections      int
	CleanupInterval     int
	DispatcherWorkers   int
	DispatcherQueueSize int
	SessionTTL          int
	HealthCheckInterval int
	MaxLatencySamples   int
}

// DefaultEngineConfig returns default engine configuration
func DefaultEngineConfig() *EngineConfig {
	return &EngineConfig{
		MaxConnections:      10000,
		CleanupInterval:     30,
		DispatcherWorkers:   10,
		DispatcherQueueSize: 1000,
		SessionTTL:          3600,
		HealthCheckInterval: 30,
		MaxLatencySamples:   1000,
	}
}

// NewEngine creates a new WebSocket engine
func NewEngine(config *EngineConfig, log logger.Logger) *Engine {
	if config == nil {
		config = DefaultEngineConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	engine := &Engine{
		connManager:  connection.NewManager(config.MaxConnections, time.Duration(config.CleanupInterval)*time.Second, log),
		router:       router.New(),
		eventEmitter: event.NewEmitter(),
		lifecycle:    lifecycle.NewManager(),
		metrics:      metrics.NewCollector(config.MaxLatencySamples),
		registry:     registry.New(),
		config:       config,
		log:          log,
		ctx:          ctx,
		cancel:       cancel,
	}

	return engine
}

// ConnectionManager returns the connection manager
func (e *Engine) ConnectionManager() *connection.Manager {
	return e.connManager
}

// Router returns the message router
func (e *Engine) Router() *router.Router {
	return e.router
}

// Dispatcher returns the dispatcher
func (e *Engine) Dispatcher() *dispatcher.Dispatcher {
	return e.dispatcher
}

// PubSub returns the pub/sub broker
func (e *Engine) PubSub() *pubsub.Broker {
	return e.pubsub
}

// SessionManager returns the session manager
func (e *Engine) SessionManager() *session.Manager {
	return e.sessionMgr
}

// EventEmitter returns the event emitter
func (e *Engine) EventEmitter() *event.Emitter {
	return e.eventEmitter
}

// Lifecycle returns the lifecycle manager
func (e *Engine) Lifecycle() *lifecycle.Manager {
	return e.lifecycle
}

// HealthChecker returns the health checker
func (e *Engine) HealthChecker() *health.Checker {
	return e.healthChecker
}

// Metrics returns the metrics collector
func (e *Engine) Metrics() *metrics.Collector {
	return e.metrics
}

// ReconnectHandler returns the reconnection handler
func (e *Engine) ReconnectHandler() *reconnect.Handler {
	return e.reconnect
}

// Registry returns the registry
func (e *Engine) Registry() *registry.Registry {
	return e.registry
}

// Start starts the engine
func (e *Engine) Start() error {
	if e.running.Load() {
		return ErrEngineAlreadyRunning
	}

	e.log.Info("Starting WebSocket engine")

	// Start dispatcher if configured
	if e.dispatcher != nil {
		e.dispatcher.Start()
	}

	// Start health checker if configured
	if e.healthChecker != nil {
		go e.healthChecker.Start(e.ctx)
	}

	// Start session cleanup if configured
	if e.sessionMgr != nil {
		go e.sessionMgr.StartCleanup(e.ctx)
	}

	// Start connection cleanup
	go e.connManager.StartCleanup(e.ctx)

	e.running.Store(true)
	e.log.Info("WebSocket engine started successfully")

	return nil
}

// Stop stops the engine
func (e *Engine) Stop() error {
	if !e.running.Load() {
		return nil
	}

	e.log.Info("Stopping WebSocket engine")

	// Cancel context
	e.cancel()

	// Stop dispatcher
	if e.dispatcher != nil {
		e.dispatcher.Stop()
	}

	// Stop health checker
	if e.healthChecker != nil {
		e.healthChecker.Stop()
	}

	// Stop session cleanup
	if e.sessionMgr != nil {
		e.sessionMgr.StopCleanup()
	}

	// Close all connections
	e.connManager.CloseAll()

	e.running.Store(false)
	e.log.Info("WebSocket engine stopped")

	return nil
}

// IsRunning returns true if engine is running
func (e *Engine) IsRunning() bool {
	return e.running.Load()
}
