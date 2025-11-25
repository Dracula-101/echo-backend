package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
	"ws-service/internal/config"
	"ws-service/internal/health"
	healthCheckers "ws-service/internal/health/checkers"
	"ws-service/internal/model"
	"ws-service/internal/service"
	"ws-service/internal/ws/broadcast"
	"ws-service/internal/ws/handlers"
	"ws-service/internal/ws/presence"
	"ws-service/internal/ws/router"
	"ws-service/internal/ws/subscription"

	"shared/pkg/cache"
	"shared/pkg/cache/redis"
	"shared/pkg/database"
	"shared/pkg/database/postgres"
	"shared/pkg/logger"
	adapter "shared/pkg/logger/adapter"
	env "shared/server/env"
	"shared/server/middleware"
	"shared/server/response"
	serverRouter "shared/server/router"
	"shared/server/server"
	"shared/server/shutdown"
	"shared/server/websocket"

	"github.com/google/uuid"
)

func createLogger(name string) logger.Logger {
	log, err := adapter.NewZap(logger.Config{
		Level:   logger.GetLoggerLevel(),
		Format:  logger.GetLoggerFormat(),
		Service: name,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create logger: %v", err))
	}
	return log
}

func loadConfig() (*config.Config, error) {
	configLogger := createLogger("config-loader")
	defer configLogger.Sync()

	appEnv := env.GetEnv("APP_ENV", "development")
	configPath := env.GetEnv("CONFIG_PATH", "configs/config.yaml")
	configLogger.Debug("Loading config from environment variables",
		logger.String("configPath", configPath),
		logger.String("environment", appEnv))

	cfg, err := config.Load(configPath, appEnv)
	if err != nil {
		configLogger.Error("Failed to load config", logger.Error(err))
		return nil, err
	}

	if err := config.ValidateAndSetDefaults(cfg); err != nil {
		configLogger.Error("Invalid configuration", logger.Error(err))
		return nil, err
	}

	configLogger.Debug("Config loaded successfully")
	return cfg, nil
}

func createDBClient(cfg config.PostgresConfig, log logger.Logger) (database.Database, error) {
	log.Debug("Creating database client")
	dbClient, err := postgres.New(database.Config{
		Host:            cfg.Host,
		Port:            cfg.Port,
		User:            cfg.User,
		Password:        cfg.Password,
		Database:        cfg.DBName,
		SSLMode:         cfg.SSLMode,
		MaxOpenConns:    cfg.MaxOpenConns,
		MaxIdleConns:    cfg.MaxIdleConns,
		ConnMaxLifetime: cfg.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.ConnMaxIdleTime,
	})
	if err != nil {
		return nil, err
	}
	log.Info("Database client created successfully")
	return dbClient, nil
}

func createCacheClient(cfg config.RedisConfig, log logger.Logger) (cache.Cache, error) {
	log.Debug("Creating cache client")
	cacheClient, err := redis.New(cache.Config{
		Host:         cfg.Host,
		Port:         cfg.Port,
		Password:     cfg.Password,
		DB:           cfg.DB,
		MaxRetries:   cfg.MaxRetries,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})
	if err != nil {
		return nil, err
	}
	log.Info("Cache client created successfully")
	return cacheClient, nil
}

type hubAdapter struct {
	*websocket.Hub
}

func (ha *hubAdapter) BroadcastEvent(event *model.RealtimeEvent) int {
	ha.Hub.BroadcastToAll(event)
	return 1 // Return count of broadcasts
}

func (ha *hubAdapter) GetStats() *model.StatsResponse {
	stats := ha.Hub.GetStats()
	return &model.StatsResponse{
		TotalUsers:       stats.OnlineUsers,
		TotalDevices:     stats.TotalClients,
		TotalConnections: int64(stats.TotalConnections),
	}
}

func newHubAdapter(hub *websocket.Hub) service.Hub {
	return &hubAdapter{Hub: hub}
}

func convertWebSocketConfig(cfg *config.WebSocketConfig) *websocket.Config {
	return &websocket.Config{
		WriteWait:              cfg.WriteWait,
		PongWait:               cfg.PongWait,
		PingPeriod:             cfg.PingPeriod,
		CloseGracePeriod:       5 * time.Second,
		HandshakeTimeout:       10 * time.Second,
		ReadBufferSize:         cfg.ReadBufferSize,
		WriteBufferSize:        cfg.WriteBufferSize,
		MaxMessageSize:         int64(cfg.MaxMessageSize),
		ClientBufferSize:       cfg.ClientBufferSize,
		CleanupInterval:        cfg.CleanupInterval,
		StaleConnectionTimeout: cfg.StaleConnectionTimeout,
		MaxConnectionsPerUser:  10,
		MaxReconnectAttempts:   3,
		ReconnectBackoff:       time.Second,
		RegisterBuffer:         cfg.RegisterBuffer,
		UnregisterBuffer:       cfg.UnregisterBuffer,
		BroadcastBuffer:        cfg.BroadcastBuffer,
		CheckOrigin:            false,
		AllowedOrigins:         []string{},
		EnableCompression:      false,
		CompressionLevel:       -1,
		MaxMessagesPerSecond:   100,
		BurstSize:              10,
		EnableMetrics:          true,
		MetricsInterval:        5 * time.Minute,
	}
}

func createHTTPRouter(
	healthHandler *health.Handler,
	wsUpgrader *websocket.Upgrader,
	createMessageHandler func(context.Context, *websocket.Client) websocket.MessageHandler,
	log logger.Logger,
) (*serverRouter.Router, error) {

	builder := serverRouter.NewBuilder().
		WithHealthEndpoint("/health", healthHandler.Health).
		WithNotFoundHandler(func(w http.ResponseWriter, r *http.Request) {
			response.RouteNotFoundError(r.Context(), r, w, log)
		}).
		WithMethodNotAllowedHandler(func(w http.ResponseWriter, r *http.Request) {
			response.MethodNotAllowedError(r.Context(), r, w)
		}).
		WithEarlyMiddleware(
			serverRouter.Middleware(middleware.RequestReceivedLogger(log)),
		).
		WithLateMiddleware(
			serverRouter.Middleware(middleware.Recovery(log)),
			serverRouter.Middleware(middleware.RequestCompletedLogger(log)),
		)

	// Health check endpoints
	builder = builder.WithRoutes(func(r *serverRouter.Router) {
		r.Get("/live", healthHandler.Liveness)
		r.Get("/ready", healthHandler.Readiness)
		r.Get("/health/liveness", healthHandler.Liveness)
		r.Get("/health/readiness", healthHandler.Readiness)
	})

	// WebSocket upgrade endpoints
	builder = builder.WithRoutes(func(r *serverRouter.Router) {
		wsHandler := func(w http.ResponseWriter, req *http.Request) {
			log.Info("Upgrading HTTP connection to WebSocket",
				logger.String("remote_addr", req.RemoteAddr),
				logger.String("request_uri", req.RequestURI),
			)
			messageHandler := createMessageHandler(req.Context(), nil)
			wsUpgrader.HandleUpgrade(w, req, messageHandler)
		}
		r.Get("/", wsHandler)
	})

	routerInstance := builder.Build()
	return routerInstance, nil
}

func setupShutdownManager(srv *server.Server, hub *websocket.Hub, subManager *subscription.Manager, log logger.Logger, cfg *config.Config) *shutdown.Manager {
	shutdownMgr := shutdown.New(
		shutdown.WithTimeout(cfg.Server.ShutdownTimeout),
		shutdown.WithLogger(log),
	)

	// Shutdown HTTP server first
	shutdownMgr.RegisterWithPriority(
		"http-server",
		shutdown.ServerShutdownHook(srv),
		shutdown.PriorityHigh,
	)

	// Then shutdown WebSocket hub
	shutdownMgr.RegisterWithPriority(
		"websocket-hub",
		shutdown.Hook(func(ctx context.Context) error {
			log.Info("Shutting down WebSocket hub")
			hub.Shutdown()
			return nil
		}),
		shutdown.PriorityHigh,
	)

	// Finally sync logger
	shutdownMgr.RegisterWithPriority(
		"logger-sync",
		shutdown.Hook(func(ctx context.Context) error {
			log.Info("Syncing logger before shutdown")
			return log.Sync()
		}),
		shutdown.PriorityLow,
	)

	return shutdownMgr
}

func waitForShutdown(shutdownMgr *shutdown.Manager) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := shutdownMgr.Wait(); err != nil {
			// Error already logged by shutdown manager
		}
	}()
	return done
}

func main() {
	env.LoadEnv()

	cfg, err := loadConfig()
	if err != nil {
		panic(fmt.Sprintf("Failed to load configuration: %v", err))
	}

	log := createLogger(cfg.Service.Name)
	defer log.Sync()

	log.Info("Starting WebSocket Service",
		logger.String("service", cfg.Service.Name),
		logger.String("version", cfg.Service.Version),
		logger.String("environment", cfg.Service.Environment),
	)

	dbClient, err := createDBClient(cfg.Database.Postgres, log)
	if err != nil {
		log.Fatal("Failed to create database client", logger.Error(err))
	}
	defer func() {
		if dbClient != nil {
			log.Info("Closing database connection")
			if err := dbClient.Close(); err != nil {
				log.Error("Failed to close database connection", logger.Error(err))
			}
		}
	}()

	var cacheClient cache.Cache
	if cfg.Cache.Enabled {
		cacheClient, err = createCacheClient(cfg.Cache.Redis, log)
		if err != nil {
			log.Fatal("Failed to create cache client", logger.Error(err))
		}
		defer func() {
			if cacheClient != nil {
				log.Info("Closing cache connection")
				if err := cacheClient.Close(); err != nil {
					log.Error("Failed to close cache connection", logger.Error(err))
				}
			}
		}()
	} else {
		log.Info("Cache is disabled in configuration")
	}

	// Initialize health manager
	healthMgr := health.NewManager(cfg.Service.Name, cfg.Service.Version)
	healthMgr.RegisterChecker(healthCheckers.NewDatabaseChecker(dbClient))
	if cfg.Cache.Enabled && cacheClient != nil {
		healthMgr.RegisterChecker(healthCheckers.NewCacheChecker(cacheClient))
	}
	log.Info("Health checks registered")

	// Initialize WebSocket Hub with converted config
	wsConfig := convertWebSocketConfig(&cfg.WebSocket)
	hub := websocket.NewHub(wsConfig, log)
	go hub.Run()
	log.Info("WebSocket hub initialized and started")

	// Initialize subscription manager
	subManager := subscription.NewManager(log)
	log.Info("Subscription manager initialized")

	// Initialize broadcaster
	broadcaster := broadcast.NewBroadcaster(hub, subManager, log)
	log.Info("Broadcaster initialized")

	// Initialize presence tracker
	presenceTracker := presence.NewTracker(hub, log)
	log.Info("Presence tracker initialized")

	// Initialize message router
	msgRouter := router.NewMessageRouter(log)
	log.Info("Message router initialized")

	// Register message handlers
	msgRouter.RegisterHandler(handlers.NewPingHandler(log))
	msgRouter.RegisterHandler(handlers.NewAuthenticateHandler(log))
	msgRouter.RegisterHandler(handlers.NewSubscribeHandler(subManager, log))
	msgRouter.RegisterHandler(handlers.NewUnsubscribeHandler(subManager, log))
	msgRouter.RegisterHandler(handlers.NewPresenceUpdateHandler(presenceTracker, log))
	msgRouter.RegisterHandler(handlers.NewPresenceQueryHandler(presenceTracker, log))
	msgRouter.RegisterHandler(handlers.NewTypingStartHandler(broadcaster, hub, log))
	msgRouter.RegisterHandler(handlers.NewTypingStopHandler(broadcaster, hub, log))
	msgRouter.RegisterHandler(handlers.NewMarkAsReadHandler(broadcaster, log))
	msgRouter.RegisterHandler(handlers.NewMarkAsDeliveredHandler(broadcaster, log))
	msgRouter.RegisterHandler(handlers.NewCallOfferHandler(broadcaster, hub, log))
	msgRouter.RegisterHandler(handlers.NewCallAnswerHandler(broadcaster, log))
	msgRouter.RegisterHandler(handlers.NewCallICEHandler(broadcaster, log))
	msgRouter.RegisterHandler(handlers.NewCallHangupHandler(broadcaster, log))
	log.Info("All message handlers registered")

	// Initialize service (minimal service for user validation)
	// Wrap hub with adapter to match service.Hub interface
	hubAdapted := newHubAdapter(hub)
	wsService := service.NewWSService(dbClient, cacheClient, hubAdapted, log)

	// Set disconnect callback
	hub.SetOnDisconnect(func(client *websocket.Client) {
		// Unsubscribe from all topics
		subManager.UnsubscribeAll(client)

		// Update presence
		presenceTracker.OnUserDisconnected(client.UserID)

		// Handle disconnect in service
		if err := wsService.HandleClientDisconnect(context.Background(), client.UserID, client.DeviceID); err != nil {
			log.Error("Failed to handle client disconnect",
				logger.String("client_id", client.ID),
				logger.String("user_id", client.UserID.String()),
				logger.Error(err),
			)
		}
	})

	// Create message handler function that uses the router
	createMessageHandler := func(ctx context.Context, client *websocket.Client) websocket.MessageHandler {
		return func(c *websocket.Client, messageData []byte) {
			if err := msgRouter.Route(ctx, c, messageData); err != nil {
				log.Error("Failed to route message",
					logger.String("client_id", c.ID),
					logger.Error(err),
				)
			}
		}
	}

	// Initialize WebSocket upgrader
	wsUpgrader := websocket.NewUpgrader(hub, wsConfig, log)

	// Set user validator
	wsUpgrader.SetUserValidator(func(ctx context.Context, userID uuid.UUID) (bool, error) {
		return wsService.ValidateUserExists(ctx, userID)
	})

	// Set after upgrade hook to initialize message handler
	wsUpgrader.SetAfterUpgrade(func(ctx context.Context, client *websocket.Client) error {
		// Set message handler for this client
		handler := createMessageHandler(ctx, client)
		client.SetMessageHandler(handler)

		// Notify presence
		presenceTracker.OnUserConnected(client.UserID)

		// Handle connect in service
		if err := wsService.HandleClientConnect(ctx, client.UserID, client.DeviceID); err != nil {
			log.Error("Failed to handle client connect",
				logger.String("client_id", client.ID),
				logger.Error(err),
			)
		}

		return nil
	})

	log.Info("WebSocket upgrader initialized")

	// Initialize health handler
	healthHandler := health.NewHandler(healthMgr)

	// Create HTTP router with WebSocket support
	routerInstance, err := createHTTPRouter(healthHandler, wsUpgrader, createMessageHandler, log)
	if err != nil {
		log.Fatal("Failed to create router", logger.Error(err))
	}

	// Create HTTP server
	serverCfg := &server.Config{
		Port:            cfg.Server.Port,
		Host:            cfg.Server.Host,
		ReadTimeout:     cfg.Server.ReadTimeout,
		WriteTimeout:    cfg.Server.WriteTimeout,
		IdleTimeout:     cfg.Server.IdleTimeout,
		ShutdownTimeout: cfg.Server.ShutdownTimeout,
		MaxHeaderBytes:  cfg.Server.MaxHeaderBytes,
		Handler:         routerInstance.Mux(),
	}

	srv, err := server.New(serverCfg, log)
	if err != nil {
		log.Fatal("Failed to create server", logger.Error(err))
	}

	// Setup graceful shutdown
	shutdownMgr := setupShutdownManager(srv, hub, subManager, log, cfg)

	// Start server
	serverErrors := make(chan error, 1)
	go func() {
		log.Info("WebSocket Service is running",
			logger.String("address", srv.Address()),
			logger.String("websocket_endpoint", "ws://"+srv.Address()),
		)
		serverErrors <- srv.Start()
	}()

	// Wait for shutdown signal or error
	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("Server error", logger.Error(err))
		}
		log.Info("Server stopped")

	case <-waitForShutdown(shutdownMgr):
		log.Info("WebSocket Service stopped gracefully")
	}
}
