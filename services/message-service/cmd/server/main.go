package main

import (
	"context"
	"fmt"
	"time"

	conversationHandler "echo-backend/services/message-service/api/v1/handler/conversation"
	messageHandler "echo-backend/services/message-service/api/v1/handler/message"
	"echo-backend/services/message-service/api/v1/middleware"
	ratelimiter "echo-backend/services/message-service/api/v1/rate_limiter"
	"echo-backend/services/message-service/internal/config"
	"echo-backend/services/message-service/internal/consumer"
	"echo-backend/services/message-service/internal/health"
	healthCheckers "echo-backend/services/message-service/internal/health/checkers"
	msgPublisher "echo-backend/services/message-service/internal/messaging"
	"echo-backend/services/message-service/internal/repo"
	"echo-backend/services/message-service/internal/service"
	cacheSvc "echo-backend/services/message-service/internal/service/cache"

	"shared/pkg/cache"
	"shared/pkg/cache/redis"
	"shared/pkg/database"
	"shared/pkg/database/postgres"
	"shared/pkg/logger"
	adapter "shared/pkg/logger/adapter"
	"shared/pkg/messaging"
	"shared/pkg/messaging/kafka"
	prommetrics "shared/pkg/monitoring/metrics/prometheus"
	env "shared/server/env"
	coreMiddleware "shared/server/middleware"
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

func createKafkaProducer(cfg config.KafkaProducerConfig, log logger.Logger) (messaging.Producer, error) {
	log.Debug("Creating Kafka producer",
		logger.String("brokers", fmt.Sprintf("%v", cfg.Brokers)),
		logger.String("topic", cfg.Topic),
	)
	producer, err := kafka.NewProducer(messaging.ProducerConfig{
		Brokers:           cfg.Brokers,
		ClientID:          cfg.ClientID,
		Compression:       cfg.Compression,
		BatchSize:         cfg.BatchSize,
		LingerMs:          cfg.LingerMs,
		Acks:              cfg.Acks,
		EnableIdempotence: cfg.EnableIdempotence,
		MaxInFlight:       cfg.MaxInFlight,
		MaxRetries:        cfg.MaxRetries,
		RetryBackoff:      cfg.RetryBackoff,
		DialTimeout:       cfg.DialTimeout,
	})
	if err != nil {
		log.Error("Failed to create Kafka producer", logger.Error(err))
		return nil, err
	}
	log.Info("Kafka producer created successfully",
		logger.String("brokers", fmt.Sprintf("%v", cfg.Brokers)),
		logger.String("topic", cfg.Topic),
	)
	return producer, nil
}

func createKafkaConsumer(cfg config.KafkaConsumerConfig, log logger.Logger) (messaging.Consumer, error) {
	log.Debug("Creating Kafka consumer",
		logger.String("brokers", fmt.Sprintf("%v", cfg.Brokers)),
		logger.String("group_id", cfg.GroupID),
	)
	kafkaConsumer, err := kafka.NewConsumer(messaging.ConsumerConfig{
		Brokers:           cfg.Brokers,
		ClientID:          cfg.ClientID,
		GroupID:           cfg.GroupID,
		SessionTimeout:    cfg.SessionTimeout,
		HeartbeatInterval: cfg.HeartbeatInterval,
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
		if ctx.Err() == nil {
			log.Error("Chat message consumer error", logger.Error(err))
		}
	}
}

func startDeliveryEventConsumer(
	ctx context.Context,
	kafkaConsumer messaging.Consumer,
	deliveryEventConsumer *consumer.DeliveryEventConsumer,
	topic string,
	log logger.Logger,
) {
	log.Info("Starting Kafka consumer for delivery events",
		logger.String("topic", topic),
	)
	if err := kafkaConsumer.Consume(ctx, []string{topic}, deliveryEventConsumer); err != nil {
		if ctx.Err() == nil {
			log.Error("Delivery event consumer error", logger.Error(err))
		}
	}
}

func createDeliveryKafkaConsumer(cfg config.KafkaConsumerConfig, log logger.Logger) (messaging.Consumer, error) {
	log.Debug("Creating Kafka consumer for delivery events",
		logger.String("brokers", fmt.Sprintf("%v", cfg.Brokers)),
		logger.String("topic", cfg.Topic),
		logger.String("group_id", cfg.GroupID),
	)
	kafkaConsumer, err := kafka.NewConsumer(messaging.ConsumerConfig{
		Brokers:           cfg.Brokers,
		ClientID:          cfg.ClientID,
		GroupID:           cfg.GroupID,
		SessionTimeout:    cfg.SessionTimeout,
		HeartbeatInterval: cfg.HeartbeatInterval,
	})
	if err != nil {
		log.Error("Failed to create delivery Kafka consumer", logger.Error(err))
		return nil, err
	}
	log.Info("Delivery Kafka consumer created successfully",
		logger.String("brokers", fmt.Sprintf("%v", cfg.Brokers)),
		logger.String("group_id", cfg.GroupID),
	)
	return kafkaConsumer, nil
}

func setupAPIRoutes(
	builder *router.Builder,
	messageHandler *messageHandler.MessageHandler,
	conversationHandler *conversationHandler.ConversationHandler,
	rateLimiter ratelimiter.RateLimiter,
	log logger.Logger,
) *router.Builder {
	log.Debug("Registering API routes")

	// Message endpoints (root level - API Gateway routes /api/v1/messages to this service)
	builder = builder.WithRoutes(func(r *router.Router) {
		r.Post(
			"conversations/{conversation_id}/messages",
			request.Adapt(messageHandler.SendMessage),
			middleware.RateLimit(
				func(ctx context.Context, key string, limit int64) (bool, int64, error) {
					return rateLimiter.CheckSendRateLimit(ctx, key)
				},
				func(req request.RequestHandler) string {
					userId, _ := request.GetUserIDFromContext(req.Context())
					return userId
				},
				60,
			),
		)
	})

	log.Debug("API routes registered successfully")
	return builder
}

func createRouter(
	messageHandler *messageHandler.MessageHandler,
	conversationHandler *conversationHandler.ConversationHandler,
	healthHandler *health.Handler,
	rateLimiter ratelimiter.RateLimiter,
	cfg *config.Config,
	log logger.Logger,
) (*router.Router, error) {

	builder := router.NewBuilder().
		WithHealthEndpoint("/health", func(rh request.RequestHandler) {
			healthHandler.Health(rh.Writer(), rh.Request())
		}).
		WithMetricsEndpoint("/metrics", func(rh request.RequestHandler) {
			prommetrics.Handler().ServeHTTP(rh.Writer(), rh.Request())
		}).
		WithNotFoundHandler(func(rh request.RequestHandler) {
			response.NotFoundError(rh.Context(), rh.Request(), rh.Writer(), fmt.Sprintf("Endpoint %s not found", rh.Request().URL.Path))
		}).
		WithMethodNotAllowedHandler(func(rh request.RequestHandler) {
			response.MethodNotAllowedError(rh.Context(), rh.Request(), rh.Writer())
		}).
		WithEarlyMiddleware(
			router.Middleware(coreMiddleware.Timeout(30*time.Second)),
			router.Middleware(coreMiddleware.BodyLimit(10*1024*1024)),
			router.Middleware(coreMiddleware.RequestReceivedLogger(log)),
			router.Middleware(coreMiddleware.RateLimit(coreMiddleware.RateLimitConfig{
				RequestsPerWindow: 100,
				Window:            time.Minute,
			})),
			router.Middleware(coreMiddleware.InterceptUserId()),
			router.Middleware(coreMiddleware.InterceptSessionId()),
			router.Middleware(coreMiddleware.InterceptSessionToken()),
			router.Middleware(coreMiddleware.InterceptIdempotencyKey()),
		).
		WithLateMiddleware(
			router.Middleware(coreMiddleware.Recovery(log)),
			router.Middleware(coreMiddleware.RequestCompletedLogger(log)),
		)

	builder = setupAPIRoutes(builder, messageHandler, conversationHandler, rateLimiter, log)

	r := builder.Build()
	return r, nil
}

func setupShutdownManager(srv *server.Server, kafkaConsumer messaging.Consumer, deliveryKafkaConsumer messaging.Consumer, consumerCancel context.CancelFunc, cfg *config.Config, log logger.Logger) *shutdown.Manager {
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

	if deliveryKafkaConsumer != nil {
		shutdownMgr.RegisterWithPriority(
			"delivery-kafka-consumer",
			shutdown.Hook(func(ctx context.Context) error {
				log.Info("Closing delivery Kafka consumer")
				return deliveryKafkaConsumer.Close()
			}),
			shutdown.PriorityHigh,
		)
	}

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

	kafkaProducer, err := createKafkaProducer(cfg.Kafka.Producer, log)
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

	// Create Kafka consumer for chat messages
	kafkaConsumer, err := createKafkaConsumer(cfg.Kafka.Consumer, log)
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

	// Create Kafka consumer for delivery events (separate consumer group)
	deliveryKafkaConsumer, err := createDeliveryKafkaConsumer(cfg.Kafka.DeliveryConsumer, log)
	if err != nil {
		log.Fatal("Failed to create delivery Kafka consumer", logger.Error(err))
	}
	defer func() {
		if deliveryKafkaConsumer != nil {
			log.Info("Closing delivery Kafka consumer")
			if err := deliveryKafkaConsumer.Close(); err != nil {
				log.Error("Failed to close delivery Kafka consumer", logger.Error(err))
			}
		}
	}()

	healthMgr := health.NewManager(cfg.Service.Name, cfg.Service.Version)
	healthMgr.RegisterChecker(healthCheckers.NewDatabaseChecker(dbClient))
	if cfg.Cache.Enabled && cacheClient != nil {
		healthMgr.RegisterChecker(healthCheckers.NewCacheChecker(cacheClient))
	}
	log.Info("Health checks registered")

	messageCache := cacheSvc.NewMessageCache(cacheClient, log)

	// Initialize repositories
	messageRepo := repo.NewMessageRepository(dbClient, cacheClient, log)
	deliveryRepo := repo.NewDeliveryRepository(dbClient, cacheClient, log)
	conversationRepo := repo.NewConversationRepository(dbClient, cacheClient, log)
	userRepo := repo.NewUserRepository(dbClient)
	reactionRepo := repo.NewReactionRepository(dbClient, cacheClient, log)
	pollRepo := repo.NewPollRepository(dbClient, cacheClient, log)
	typingRepo := repo.NewTypingRepository(dbClient, log)
	draftRepo := repo.NewDraftRepository(dbClient, log)
	bookmarkRepo := repo.NewBookmarkRepository(dbClient, log)
	settingsRepo := repo.NewConversationSettingsRepository(dbClient, log)
	inviteRepo := repo.NewInviteRepository(dbClient, log)
	reportRepo := repo.NewReportRepository(dbClient, log)

	// Initialize event publisher
	publisher := msgPublisher.NewKafkaEventPublisher(kafkaProducer, log)

	// Initialize services
	messageService := service.NewMessageService(messageRepo, deliveryRepo, conversationRepo, userRepo, publisher, messageCache, log)
	conversationService := service.NewConversationService(conversationRepo, userRepo, publisher, messageCache, log)
	deliveryService := service.NewDeliveryService(deliveryRepo, log)
	reactionService := service.NewReactionService(reactionRepo, messageRepo, conversationRepo, publisher, log)
	pollService := service.NewPollService(pollRepo, messageRepo, conversationRepo, publisher, log)
	typingService := service.NewTypingService(typingRepo, publisher, log)
	draftService := service.NewDraftService(draftRepo, log)
	bookmarkService := service.NewBookmarkService(bookmarkRepo, log)
	settingsService := service.NewConversationSettingsService(settingsRepo, log)
	inviteService := service.NewInviteService(inviteRepo, conversationRepo, log)
	reportService := service.NewReportService(reportRepo, log)

	// Initialize handlers
	messageHandler := messageHandler.NewMessageHandler(messageService, reactionService, pollService, bookmarkService, reportService, messageCache, log)
	conversationHandler := conversationHandler.NewConversationHandler(conversationService, messageService, typingService, draftService, settingsService, inviteService, log)
	healthHandler := health.NewHandler(healthMgr)
	chatMessageConsumer := consumer.NewChatMessageConsumer(messageService, log)
	deliveryEventConsumer := consumer.NewDeliveryEventConsumer(deliveryService, log)

	rateLimiter := ratelimiter.NewRateLimiter(cacheClient, log)
	routerInstance, err := createRouter(messageHandler, conversationHandler, healthHandler, rateLimiter, cfg, log)
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
	go startChatMessageConsumer(consumerCtx, kafkaConsumer, chatMessageConsumer, cfg.Kafka.Consumer.Topic, log)
	go startDeliveryEventConsumer(consumerCtx, deliveryKafkaConsumer, deliveryEventConsumer, cfg.Kafka.DeliveryConsumer.Topic, log)

	shutdownMgr := setupShutdownManager(srv, kafkaConsumer, deliveryKafkaConsumer, consumerCancel, cfg, log)

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
