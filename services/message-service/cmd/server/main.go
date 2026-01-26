package main

import (
	"context"
	"fmt"
	"time"

	conversationHandler "echo-backend/services/message-service/api/v1/handler/conversation"
	messageHandler "echo-backend/services/message-service/api/v1/handler/message"
	"echo-backend/services/message-service/internal/config"
	"echo-backend/services/message-service/internal/consumer"
	"echo-backend/services/message-service/internal/health"
	healthCheckers "echo-backend/services/message-service/internal/health/checkers"
	"echo-backend/services/message-service/internal/repo"
	"echo-backend/services/message-service/internal/service"

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
	"shared/server/request"
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
		Brokers:           cfg.Brokers,
		ClientID:          cfg.ClientID,
		GroupID:           cfg.GroupID,
		RetryBackoff:      100,
		SessionTimeout:    30000,
		HeartbeatInterval: 10000,
		MaxRetries:        5,
	})
	if err != nil {
		log.Error("Failed to create Kafka producer", logger.Error(err))
		return nil, err
	}
	log.Info("Kafka producer created successfully",
		logger.String("brokers", fmt.Sprintf("%v", cfg.Brokers)),
	)
	return producer, nil
}

func createKafkaConsumer(cfg config.KafkaConfig, log logger.Logger) (messaging.Consumer, error) {
	log.Debug("Creating Kafka consumer",
		logger.String("brokers", fmt.Sprintf("%v", cfg.Brokers)),
		logger.String("group_id", cfg.GroupID),
	)
	kafkaConsumer, err := kafka.NewConsumer(messaging.Config{
		Brokers:           cfg.Brokers,
		ClientID:          cfg.ClientID,
		GroupID:           cfg.GroupID,
		RetryBackoff:      100,
		SessionTimeout:    30000,
		HeartbeatInterval: 10000,
		MaxRetries:        5,
	})
	if err != nil {
		log.Error("Failed to create Kafka consumer", logger.Error(err))
		return nil, err
	}
	log.Info("Kafka consumer created successfully",
		logger.String("brokers", fmt.Sprintf("%v", cfg.Brokers)),
		logger.String("group_id", cfg.GroupID),
	)
	return kafkaConsumer, nil
}

func startChatMessageConsumer(
	ctx context.Context,
	kafkaConsumer messaging.Consumer,
	chatMessageConsumer *consumer.ChatMessageConsumer,
	topic string,
	log logger.Logger,
) {
	log.Info("Starting Kafka consumer for chat messages",
		logger.String("topic", topic),
	)
	consumerTopics := []string{topic}
	if err := kafkaConsumer.Consume(ctx, consumerTopics, chatMessageConsumer); err != nil {
		log.Fatal("Failed to start Kafka consumer", logger.Error(err))
	}
}

func setupAPIRoutes(
	builder *router.Builder,
	messageHandler *messageHandler.MessageHandler,
	conversationHandler *conversationHandler.ConversationHandler,
	log logger.Logger,
) *router.Builder {
	log.Debug("Registering API routes")

	// Message endpoints (root level - API Gateway routes /api/v1/messages to this service)
	builder = builder.WithRoutes(func(r *router.Router) {
		r.Post("/", request.Adapt(messageHandler.SendMessage))
		r.Get("/", request.Adapt(messageHandler.GetMessages))
		r.Get("/{id}", request.Adapt(messageHandler.GetMessageByID))
	})

	// Conversation endpoints
	builder = builder.WithRoutes(func(r *router.Router) {
		r.Post("/conversations", request.Adapt(conversationHandler.CreateConversation))
		r.Get("/conversations/{id}", request.Adapt(conversationHandler.GetConversationByID))
	})
	log.Debug("API routes registered successfully")
	return builder
}

func createRouter(
	messageHandler *messageHandler.MessageHandler,
	conversationHandler *conversationHandler.ConversationHandler,
	healthHandler *health.Handler,
	cfg *config.Config,
	log logger.Logger,
) (*router.Router, error) {

	builder := router.NewBuilder().
		WithHealthEndpoint("/health", func(rh request.RequestHandler) {
			healthHandler.Health(rh.Writer(), rh.Request())
		}).
		WithNotFoundHandler(func(rh request.RequestHandler) {
			response.NotFoundError(rh.Context(), rh.Request(), rh.Writer(), fmt.Sprintf("Endpoint %s not found", rh.Request().URL.Path))
		}).
		WithMethodNotAllowedHandler(func(rh request.RequestHandler) {
			response.MethodNotAllowedError(rh.Context(), rh.Request(), rh.Writer())
		}).
		WithEarlyMiddleware(
			router.Middleware(middleware.Timeout(30*time.Second)),
			router.Middleware(middleware.BodyLimit(10*1024*1024)),
			router.Middleware(middleware.RequestReceivedLogger(log)),
			router.Middleware(middleware.RateLimit(middleware.RateLimitConfig{
				RequestsPerWindow: 100,
				Window:            time.Minute,
			})),
			router.Middleware(middleware.InterceptUserId()),
			router.Middleware(middleware.InterceptSessionId()),
			router.Middleware(middleware.InterceptSessionToken()),
		).
		WithLateMiddleware(
			router.Middleware(middleware.Recovery(log)),
			router.Middleware(middleware.RequestCompletedLogger(log)),
		)

	builder = setupAPIRoutes(builder, messageHandler, conversationHandler, log)

	r := builder.Build()
	return r, nil
}

func setupShutdownManager(srv *server.Server, kafkaConsumer messaging.Consumer, consumerCancel context.CancelFunc, cfg *config.Config, log logger.Logger) *shutdown.Manager {
	shutdownMgr := shutdown.New(
		shutdown.WithTimeout(cfg.Server.ShutdownTimeout),
		shutdown.WithLogger(log),
	)

	shutdownMgr.RegisterWithPriority(
		"kafka-consumer",
		shutdown.Hook(func(ctx context.Context) error {
			consumerCancel()
			return kafkaConsumer.Close()
		}),
		shutdown.PriorityHigh,
	)

	shutdownMgr.RegisterWithPriority(
		"http-server",
		shutdown.ServerShutdownHook(srv),
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

	// Create Kafka consumer
	kafkaConsumer, err := createKafkaConsumer(cfg.Kafka, log)
	if err != nil {
		log.Fatal("Failed to create Kafka consumer", logger.Error(err))
	}
	defer func() {
		if kafkaConsumer != nil {
			log.Info("Closing Kafka consumer")
			if err := kafkaConsumer.Close(); err != nil {
				log.Error("Failed to close Kafka consumer", logger.Error(err))
			}
		}
	}()

	healthMgr := health.NewManager(cfg.Service.Name, cfg.Service.Version)
	healthMgr.RegisterChecker(healthCheckers.NewDatabaseChecker(dbClient))
	if cfg.Cache.Enabled && cacheClient != nil {
		healthMgr.RegisterChecker(healthCheckers.NewCacheChecker(cacheClient))
	}
	log.Info("Health checks registered")

	// Initialize repositories
	messageRepo := repo.NewMessageRepository(dbClient, cacheClient, log)
	conversationRepo := repo.NewConversationRepository(dbClient, cacheClient, log)
	userRepo := repo.NewUserRepository(dbClient)

	// Initialize event publisher
	eventPublisher := service.NewKafkaEventPublisher(kafkaProducer, log)

	// Initialize services
	messageService := service.NewMessageService(messageRepo, conversationRepo, userRepo, eventPublisher, cacheClient, log)
	conversationService := service.NewConversationService(conversationRepo, log)

	// Initialize handlers
	messageHandler := messageHandler.NewMessageHandler(messageService, log)
	conversationHandler := conversationHandler.NewConversationHandler(conversationService, log)
	healthHandler := health.NewHandler(healthMgr)
	chatMessageConsumer := consumer.NewChatMessageConsumer(messageService, log)

	routerInstance, err := createRouter(messageHandler, conversationHandler, healthHandler, cfg, log)
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

	consumerCtx, consumerCancel := context.WithCancel(context.Background())
	go startChatMessageConsumer(consumerCtx, kafkaConsumer, chatMessageConsumer, cfg.Kafka.Topic, log)

	shutdownMgr := setupShutdownManager(srv, kafkaConsumer, consumerCancel, cfg, log)

	serverErrors := make(chan error, 1)
	go func() {
		log.Info("Message Service is running",
			logger.String("address", srv.Address()),
		)
		serverErrors <- srv.Start()
	}()

	select {
	case err := <-serverErrors:
		if err != nil && !server.IsServerDownErr(err) {
			log.Fatal("Server error", logger.Error(err))
		}
		log.Info("Server stopped")

	case <-waitForShutdown(shutdownMgr):
		log.Info("Message Service stopped gracefully")
	}
}
