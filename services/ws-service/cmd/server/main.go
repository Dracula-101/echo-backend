package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"ws-service/internal/config"
	"ws-service/internal/health"
	healthCheckers "ws-service/internal/health/checkers"
	"ws-service/internal/service"
	wsManager "ws-service/internal/websocket"

	"shared/pkg/cache"
	"shared/pkg/cache/redis"
	"shared/pkg/database"
	"shared/pkg/database/postgres"
	"shared/pkg/logger"
	adapter "shared/pkg/logger/adapter"
	env "shared/server/env"
	"shared/server/middleware"
	"shared/server/response"
	"shared/server/router"
	"shared/server/server"
	"shared/server/shutdown"
	"shared/server/websocket/handler"

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

	configLogger.Debug("Loading configuration",
		logger.String("path", configPath),
		logger.String("environment", appEnv),
	)

	cfg, err := config.Load(configPath, appEnv)
	if err != nil {
		return nil, err
	}

	if err := config.ValidateAndSetDefaults(cfg); err != nil {
		return nil, err
	}

	configLogger.Debug("Configuration loaded successfully")
	return cfg, nil
}

func createDBClient(cfg config.PostgresConfig, log logger.Logger) (database.Database, error) {
	log.Debug("Creating database client",
		logger.String("host", cfg.Host),
		logger.Int("port", cfg.Port),
		logger.String("database", cfg.DBName),
	)

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
	log.Debug("Creating cache client",
		logger.String("host", cfg.Host),
		logger.Int("port", cfg.Port),
	)

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

func setupHealthChecks(dbClient database.Database, cacheClient cache.Cache, cfg *config.Config) *health.Manager {
	healthMgr := health.NewManager(cfg.Service.Name, cfg.Service.Version)

	if dbClient != nil {
		healthMgr.RegisterChecker(healthCheckers.NewDatabaseChecker(dbClient))
	}

	if cacheClient != nil && cfg.Cache.Enabled {
		healthMgr.RegisterChecker(healthCheckers.NewCacheChecker(cacheClient))
	}

	return healthMgr
}

func createWebSocketHandler(
	manager *wsManager.Manager,
	wsService service.WSService,
	cfg *config.Config,
	log logger.Logger,
) *handler.Handler {
	handlerCfg := &handler.Config{
		// Connection settings from config
		SendBufferSize:    cfg.WebSocket.ClientBufferSize,
		MaxMessageSize:    int64(cfg.WebSocket.MaxMessageSize),
		PingInterval:      cfg.WebSocket.PingPeriod,
		WriteTimeout:      cfg.WebSocket.WriteWait,
		ReadTimeout:       cfg.WebSocket.PongWait,
		StaleTimeout:      cfg.WebSocket.StaleConnectionTimeout,
		CheckOrigin:       func(r *http.Request) bool { return true },
		ReadBufferSize:    cfg.WebSocket.ReadBufferSize,
		WriteBufferSize:   cfg.WebSocket.WriteBufferSize,
		EnableCompression: false,

		ValidateUser: func(ctx context.Context, userID uuid.UUID) (bool, error) {
			return wsService.ValidateUserExists(ctx, userID)
		},
		ExtractUserID: handler.DefaultUserIDExtractor,
		HandleMessage: func(ctx context.Context, conn *handler.Connection, message []byte) error {
			return manager.HandleMessage(ctx, conn, message)
		},
		ExtractMetadata: handler.DefaultMetadataExtractor,
		OnConnected: func(conn *handler.Connection) {
			userID, _ := conn.GetMetadata("user_id")
			deviceID, _ := conn.GetMetadata("device_id")
			if uid, ok := userID.(uuid.UUID); ok {
				if did, ok := deviceID.(string); ok {
					wsService.HandleClientConnect(context.Background(), uid, did)
				}
			}
		},
		OnDisconnected: func(conn *handler.Connection) {
			userID, _ := conn.GetMetadata("user_id")
			deviceID, _ := conn.GetMetadata("device_id")
			if uid, ok := userID.(uuid.UUID); ok {
				if did, ok := deviceID.(string); ok {
					wsService.HandleClientDisconnect(context.Background(), uid, did)
				}
			}
		},
		SendErrorsToClient: true,
	}

	return handler.New(manager.GetEngine(), handlerCfg, log)
}

func setupAPIRoutes(
	builder *router.Builder,
	wsHandler *handler.Handler,
	log logger.Logger,
) *router.Builder {
	log.Debug("Registering API routes")

	builder = builder.WithRoutes(func(r *router.Router) {
		r.Get("/", wsHandler.HandleUpgrade)
	})

	log.Debug("API routes registered successfully")
	return builder
}

func createRouter(
	wsHandler *handler.Handler,
	healthHandler *health.Handler,
	log logger.Logger,
) (*router.Router, error) {
	builder := router.NewBuilder().
		WithHealthEndpoint("/health", healthHandler.Health).
		WithNotFoundHandler(func(w http.ResponseWriter, r *http.Request) {
			response.RouteNotFoundError(r.Context(), r, w, log)
		}).
		WithMethodNotAllowedHandler(func(w http.ResponseWriter, r *http.Request) {
			response.MethodNotAllowedError(r.Context(), r, w)
		}).
		WithEarlyMiddleware(
			router.Middleware(middleware.RequestReceivedLogger(log)),
		).
		WithLateMiddleware(
			router.Middleware(middleware.Recovery(log)),
			router.Middleware(middleware.RequestCompletedLogger(log)),
		)

	// Health check endpoints
	builder = builder.WithRoutes(func(r *router.Router) {
		r.Get("/live", healthHandler.Liveness)
		r.Get("/ready", healthHandler.Readiness)
		r.Get("/health/liveness", healthHandler.Liveness)
		r.Get("/health/readiness", healthHandler.Readiness)
	})

	builder = setupAPIRoutes(builder, wsHandler, log)

	r := builder.Build()
	return r, nil
}

func setupShutdownManager(
	srv *server.Server,
	manager *wsManager.Manager,
	dbClient database.Database,
	cacheClient cache.Cache,
	log logger.Logger,
	cfg *config.Config,
) *shutdown.Manager {
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

	// Then shutdown WebSocket manager
	shutdownMgr.RegisterWithPriority(
		"websocket-manager",
		shutdown.Hook(func(ctx context.Context) error {
			log.Info("Shutting down WebSocket manager")
			return manager.Stop()
		}),
		shutdown.PriorityHigh,
	)

	// Close database
	if dbClient != nil {
		shutdownMgr.RegisterWithPriority(
			"database",
			shutdown.Hook(func(ctx context.Context) error {
				log.Info("Closing database connection")
				return dbClient.Close()
			}),
			shutdown.PriorityNormal,
		)
	}

	// Close cache
	if cacheClient != nil {
		shutdownMgr.RegisterWithPriority(
			"cache",
			shutdown.Hook(func(ctx context.Context) error {
				log.Info("Closing cache connection")
				return cacheClient.Close()
			}),
			shutdown.PriorityNormal,
		)
	}

	// Sync logger
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

	log.Info("Initializing application",
		logger.String("service", cfg.Service.Name),
		logger.String("version", cfg.Service.Version),
		logger.String("environment", cfg.Service.Environment),
	)

	// Create database client
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

	// Create cache client (optional)
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

	// Initialize WebSocket manager
	manager := wsManager.NewManager(log)
	log.Info("WebSocket manager initialized")

	// Start WebSocket engine
	if err := manager.Start(); err != nil {
		log.Fatal("Failed to start WebSocket manager", logger.Error(err))
	}
	log.Info("WebSocket engine started")

	// Initialize service with hub
	wsService := service.NewWSService(dbClient, cacheClient, manager.GetHub(), log)

	// Setup health checks
	healthMgr := setupHealthChecks(dbClient, cacheClient, cfg)
	healthHandler := health.NewHandler(healthMgr)
	log.Info("Health checks registered")

	// Initialize WebSocket handler with config
	wsHandler := createWebSocketHandler(manager, wsService, cfg, log)

	// Create HTTP server
	routerInstance, err := createRouter(wsHandler, healthHandler, log)
	if err != nil {
		log.Fatal("Failed to create router", logger.Error(err))
	}

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
	shutdownMgr := setupShutdownManager(srv, manager, dbClient, cacheClient, log, cfg)

	// Start server
	serverErrors := make(chan error, 1)
	go func() {
		log.Info("Starting WebSocket Service",
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
