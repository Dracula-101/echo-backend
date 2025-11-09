package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"echo-backend/services/message-service/api/handler"
	"echo-backend/services/message-service/internal/config"
	"echo-backend/services/message-service/internal/health"
	healthCheckers "echo-backend/services/message-service/internal/health/checkers"
	"echo-backend/services/message-service/internal/repo"
	"echo-backend/services/message-service/internal/service"
	"echo-backend/services/message-service/internal/websocket"

	"shared/pkg/cache"
	"shared/pkg/cache/redis"
	"shared/pkg/database"
	"shared/pkg/database/postgres"
	"shared/pkg/logger"
	adapter "shared/pkg/logger/adapter"
	"shared/pkg/messaging"
	"shared/pkg/messaging/kafka"
	env "shared/server/env"
	"shared/server/middleware"
	"shared/server/response"
	"shared/server/router"
	"shared/server/server"
	"shared/server/shutdown"
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
	log.Debug("Creating Kafka producer",
		logger.String("brokers", fmt.Sprintf("%v", cfg.Brokers)),
	)
	producer, err := kafka.NewProducer(messaging.Config{
		Brokers:    cfg.Brokers,
		ClientID:   "message-service",
		MaxRetries: 3,
	})
	if err != nil {
		return nil, err
	}
	log.Info("Kafka producer created successfully",
		logger.String("brokers", fmt.Sprintf("%v", cfg.Brokers)),
	)
	return producer, nil
}

func setupAPIRoutes(builder *router.Builder, h *handler.MessageHandler, log logger.Logger) *router.Builder {
	log.Debug("Registering message API routes")
	builder = builder.WithRoutes(func(r *router.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			response.JSON(w, http.StatusOK, map[string]string{"message": "Message Service is running"})
		})
	})
	log.Debug("Message API routes registered successfully")
	return builder
}

func createRouter(
	httpHandler *handler.MessageHandler,
	healthHandler *health.Handler,
	cfg *config.Config,
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
			router.Middleware(middleware.Timeout(30*time.Second)),
			router.Middleware(middleware.BodyLimit(10*1024*1024)),
			router.Middleware(middleware.RequestReceivedLogger(log)),
			router.Middleware(middleware.RateLimit(middleware.RateLimitConfig{
				RequestsPerWindow: 100,
				Window:            time.Minute,
			})),
		).
		WithLateMiddleware(
			router.Middleware(middleware.Recovery(log)),
			router.Middleware(middleware.RequestCompletedLogger(log)),
		)

	builder = setupAPIRoutes(builder, httpHandler, log)

	r := builder.Build()
	return r, nil
}

func setupShutdownManager(srv *server.Server, hub *websocket.Hub, log logger.Logger, cfg *config.Config) *shutdown.Manager {
	shutdownMgr := shutdown.New(
		shutdown.WithTimeout(cfg.Server.ShutdownTimeout),
		shutdown.WithLogger(log),
	)

	shutdownMgr.RegisterWithPriority(
		"http-server",
		shutdown.ServerShutdownHook(srv),
		shutdown.PriorityHigh,
	)

	shutdownMgr.RegisterWithPriority(
		"websocket-hub",
		shutdown.Hook(func(ctx context.Context) error {
			log.Info("Shutting down WebSocket hub")
			hub.Shutdown()
			return nil
		}),
		shutdown.PriorityHigh,
	)

	if cfg.Shutdown.WaitForConnections && cfg.Shutdown.DrainTimeout > 0 {
		shutdownMgr.RegisterWithOptions(
			"drain-connections",
			shutdown.DelayHook(cfg.Shutdown.DrainTimeout),
			shutdown.PriorityHigh,
			cfg.Shutdown.DrainTimeout,
		)
	}

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

	log.Info("Starting Message Service",
		logger.String("service", cfg.Service.Name),
		logger.String("version", cfg.Service.Version),
		logger.String("environment", cfg.Service.Environment),
	)

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

	healthMgr := health.NewManager(cfg.Service.Name, cfg.Service.Version)
	healthMgr.RegisterChecker(healthCheckers.NewDatabaseChecker(dbClient))
	if cfg.Cache.Enabled && cacheClient != nil {
		healthMgr.RegisterChecker(healthCheckers.NewCacheChecker(cacheClient))
	}
	log.Info("Health checks registered")

	messageRepo := repo.NewMessageRepository(dbClient)
	messageService := service.NewMessageService(messageRepo, hub, kafkaProducer, log)
	httpHandler := handler.NewMessageHandler(messageService, log)
	healthHandler := health.NewHandler(healthMgr)

	routerInstance, err := createRouter(httpHandler, healthHandler, cfg, log)
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

	shutdownMgr := setupShutdownManager(srv, hub, log, cfg)

	serverErrors := make(chan error, 1)
	go func() {
		log.Info("Message Service is running",
			logger.String("address", srv.Address()),
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
