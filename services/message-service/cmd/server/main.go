package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"echo-backend/services/message-service/internal/config"
	"echo-backend/services/message-service/internal/handler"
	"echo-backend/services/message-service/internal/repo"
	"echo-backend/services/message-service/internal/service"
	"echo-backend/services/message-service/internal/websocket"

	"shared/pkg/cache"
	"shared/pkg/cache/redis"
	"shared/pkg/database"
	"shared/pkg/database/postgres"
	"shared/pkg/logger"
	"shared/pkg/logger/adapter"
	"shared/pkg/messaging"
	"shared/pkg/messaging/kafka"
	"shared/server/env"
	"shared/server/middleware"
	"shared/server/response"
	"shared/server/router"
	"shared/server/server"
	"shared/server/shutdown"
)

func main() {
	loadenv()
	cfg, err := loadConfig()
	if err != nil {
		panic(fmt.Sprintf("Failed to load configuration: %v", err))
	}

	log := createLogger(cfg.Service.Name)
	defer log.Sync()

	dbClient, err := createDBClient(cfg.Database, log)
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
		cacheClient, err = createCacheClient(cfg.Cache, log)
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

	kafkaProducer, err := createKafkaProducer(cfg.Kafka, log)
	if err != nil {
		log.Fatal("Failed to create Kafka producer", logger.Error(err))
	}
	defer func() {
		if kafkaProducer != nil {
			log.Info("Closing Kafka producer")
			if err := kafkaProducer.Close(); err != nil {
				log.Error("Failed to close Kafka producer", logger.Error(err))
			}
		}
	}()

	hub := websocket.NewHub(log)
	go hub.Run()
	log.Info("WebSocket hub started")

	messageRepo := repo.NewMessageRepository(dbClient)

	messageService := service.NewMessageService(messageRepo, hub, kafkaProducer, log)

	httpHandler := handler.NewHTTPHandler(messageService, log)
	wsHandler := handler.NewWebSocketHandler(hub, messageService, log)

	routerInstance, err := createRouter(httpHandler, wsHandler, log)
	if err != nil {
		log.Fatal("Failed to create router", logger.Error(err))
	}

	serverCfg := server.Config{
		Host:            cfg.Server.Host,
		Port:            cfg.Server.Port,
		ReadTimeout:     cfg.Server.ReadTimeout,
		WriteTimeout:    cfg.Server.WriteTimeout,
		IdleTimeout:     cfg.Server.IdleTimeout,
		ShutdownTimeout: cfg.Server.ShutdownTimeout,
		MaxHeaderBytes:  cfg.Server.MaxHeaderBytes,
		Handler:         routerInstance.Mux(),
	}

	srv, err := server.New(&serverCfg, log)
	if err != nil {
		log.Fatal("Failed to create server", logger.Error(err))
	}

	shutdownMgr := setupShutdownManager(srv, hub, log, cfg)

	serverErrors := make(chan error, 1)
	go func() {
		log.Info("Starting Message Service server",
			logger.String("host", cfg.Server.Host),
			logger.Int("port", cfg.Server.Port),
		)
		serverErrors <- srv.Start()
	}()

	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("Server error", logger.Error(err))
		}
		log.Info("Server stopped")

	case <-waitForShutdown(shutdownMgr):
		log.Info("Message Service stopped gracefully")
	}
}

func loadenv() {
	if err := env.LoadEnv(); err != nil {
		panic(fmt.Sprintf("Failed to load environment variables: %v", err))
	}
}

func loadConfig() (*config.Config, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return cfg, nil
}

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

func createDBClient(cfg config.DatabaseConfig, log logger.Logger) (database.Database, error) {
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

func createCacheClient(cfg config.CacheConfig, log logger.Logger) (cache.Cache, error) {
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

func createKafkaProducer(cfg config.KafkaConfig, log logger.Logger) (messaging.Producer, error) {
	log.Debug("Creating Kafka producer")
	producer, err := kafka.NewProducer(messaging.Config{
		Brokers:    cfg.Brokers,
		ClientID:   "message-service",
		MaxRetries: cfg.RetryMax,
	})
	if err != nil {
		return nil, err
	}
	log.Info("Kafka producer created successfully",
		logger.String("brokers", fmt.Sprintf("%v", cfg.Brokers)),
	)
	return producer, nil
}

func createRouter(
	httpHandler *handler.HTTPHandler,
	wsHandler *handler.WebSocketHandler,
	log logger.Logger,
) (*router.Router, error) {
	builder := router.NewBuilder().
		WithNotFoundHandler(func(w http.ResponseWriter, r *http.Request) {
			response.RouteNotFoundError(r.Context(), r, w, log)
		}).
		WithMethodNotAllowedHandler(func(w http.ResponseWriter, r *http.Request) {
			response.MethodNotAllowedError(r.Context(), r, w)
		}).
		WithHealthEndpoint("/health", func(w http.ResponseWriter, r *http.Request) {
			response.JSON(w, http.StatusOK, map[string]string{
				"status":  "healthy",
				"service": "message-service",
			})
		}).
		WithEarlyMiddleware(
			router.Middleware(middleware.RequestReceivedLogger(log)),
		).
		WithLateMiddleware(
			router.Middleware(middleware.Recovery(log)),
			router.Middleware(middleware.RequestCompletedLogger(log)),
		)

	// Health endpoint
	builder = builder.WithRoutes(func(r *router.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			response.JSON(w, http.StatusOK, map[string]string{
				"status":  "healthy",
				"service": "message-service",
			})
		})
		r.Get("/ws", wsHandler.HandleWebSocket)
	})

	builder = setupAPIRoutes(builder, httpHandler, log)

	r := builder.Build()
	return r, nil
}

func setupAPIRoutes(builder *router.Builder, h *handler.HTTPHandler, log logger.Logger) *router.Builder {
	log.Debug("Registering message API routes")
	builder = builder.WithRoutes(func(r *router.Router) {
		// Message routes - these will be accessed via /api/v1/messages/* from the gateway
		r.Post("/", h.SendMessage)                 // POST /api/v1/messages
		r.Get("/{message_id}", h.GetMessage)       // GET /api/v1/messages/{message_id}
		r.Put("/{message_id}", h.EditMessage)      // PUT /api/v1/messages/{message_id}
		r.Delete("/{message_id}", h.DeleteMessage) // DELETE /api/v1/messages/{message_id}
		r.Post("/{message_id}/read", h.MarkAsRead) // POST /api/v1/messages/{message_id}/read

		// Conversation routes - accessed via /api/v1/messages/conversations/*
		r.Get("/conversations/{conversation_id}/messages", h.GetMessages) // GET /api/v1/messages/conversations/{conversation_id}/messages
		r.Post("/conversations/{conversation_id}/typing", h.SetTyping)    // POST /api/v1/messages/conversations/{conversation_id}/typing
	})
	log.Debug("Message API routes registered successfully")
	return builder
}

func setupShutdownManager(srv *server.Server, hub *websocket.Hub, log logger.Logger, cfg *config.Config) *shutdown.Manager {
	shutdownMgr := shutdown.New(
		shutdown.WithTimeout(cfg.Server.ShutdownTimeout),
		shutdown.WithLogger(log),
	)

	// Register HTTP server shutdown
	shutdownMgr.RegisterWithPriority(
		"http-server",
		shutdown.ServerShutdownHook(srv),
		shutdown.PriorityHigh,
	)

	// Register WebSocket hub shutdown
	shutdownMgr.RegisterWithPriority(
		"websocket-hub",
		shutdown.Hook(func(ctx context.Context) error {
			log.Info("Shutting down WebSocket hub")
			hub.Shutdown()
			return nil
		}),
		shutdown.PriorityHigh,
	)

	// Register connection drain if enabled
	if cfg.Shutdown.WaitForConnections && cfg.Shutdown.DrainTimeout > 0 {
		shutdownMgr.RegisterWithOptions(
			"drain-connections",
			shutdown.DelayHook(cfg.Shutdown.DrainTimeout),
			shutdown.PriorityHigh,
			cfg.Shutdown.DrainTimeout,
		)
	}

	// Register logger sync
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
			// Error is already logged by shutdown manager
		}
	}()
	return done
}
