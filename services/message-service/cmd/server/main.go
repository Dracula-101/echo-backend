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
	"echo-backend/services/message-service/internal/service/idempotency"

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

type messageRateLimits struct {
	send   coreMiddleware.Handler
	react  coreMiddleware.Handler
	report coreMiddleware.Handler
	search coreMiddleware.Handler
}

func setupAPIRoutes(
	builder *router.Builder,
	messageHandler *messageHandler.MessageHandler,
	conversationHandler *conversationHandler.ConversationHandler,
	rl ratelimiter.RateLimiter,
	log logger.Logger,
) *router.Builder {
	log.Debug("Registering API routes")

	userKey := func(req request.RequestHandler) string {
		userID, _ := request.GetUserIDUUIDFromContext(req.Context())
		return userID.String()
	}
	userConvKey := func(req request.RequestHandler) string {
		userID, _ := request.GetUserIDUUIDFromContext(req.Context())
		return userID.String() + "|" + req.PathParam("conversation_id")
	}

	createConvRateLimit := middleware.RateLimit(
		func(ctx context.Context, key string, limit int64) (bool, int64, error) {
			return rl.CheckCreateConvRateLimit(ctx, key)
		},
		userKey,
		ratelimiter.RLCreateConvLimit,
	)
	sendRateLimit := middleware.RateLimit(
		func(ctx context.Context, key string, limit int64) (bool, int64, error) {
			return rl.CheckSendRateLimit(ctx, key)
		},
		userKey,
		ratelimiter.RLSendLimit,
	)
	reactRateLimit := middleware.RateLimit(
		func(ctx context.Context, key string, limit int64) (bool, int64, error) {
			return rl.CheckReactRateLimit(ctx, key)
		},
		userKey,
		ratelimiter.RLReactLimit,
	)
	reportRateLimit := middleware.RateLimit(
		func(ctx context.Context, key string, limit int64) (bool, int64, error) {
			return rl.CheckReportRateLimit(ctx, key)
		},
		userKey,
		ratelimiter.RLReportLimit,
	)
	searchRateLimit := middleware.RateLimit(
		func(ctx context.Context, key string, limit int64) (bool, int64, error) {
			userID, convID, ok := splitUserConvKey(key)
			if !ok {
				return true, 0, nil
			}
			return rl.CheckSearchRateLimit(ctx, userID, convID)
		},
		userConvKey,
		ratelimiter.RLSearchLimit,
	)

	rateLimits := messageRateLimits{
		send:   sendRateLimit,
		react:  reactRateLimit,
		report: reportRateLimit,
		search: searchRateLimit,
	}

	builder = builder.WithRoutes(func(r *router.Router) {
		registerConversationRoutes(r, conversationHandler, createConvRateLimit)
		registerMessageRoutes(r, messageHandler, rateLimits)
		registerPollRoutes(r, messageHandler)
		registerInviteRoutes(r, conversationHandler)
	})

	log.Debug("API routes registered successfully")
	return builder
}

// registerConversationRoutes wires endpoints under /conversations.
//
// Note on ordering: literal segments (e.g. "/conversations/me") must be
// registered before parameterized ones (e.g. "/conversations/{id}") so the
// gorilla/mux matcher does not greedily bind them to the {id} route.
func registerConversationRoutes(
	r *router.Router,
	h *conversationHandler.ConversationHandler,
	createConvRateLimit coreMiddleware.Handler,
) {
	r.Post("/conversations", request.Adapt(h.CreateConversation), createConvRateLimit)
	r.Get("/conversations/me", request.Adapt(h.GetUserConversations))

	r.Get("/conversations/{id}", request.Adapt(h.GetConversationByID))
	r.Put("/conversations/{id}", request.Adapt(h.UpdateConversation))
	r.Delete("/conversations/{id}", request.Adapt(h.DeleteConversation))

	r.Post("/conversations/{id}/participants", request.Adapt(h.AddParticipants))
	r.Put("/conversations/{id}/participants/{user_id}", request.Adapt(h.UpdateParticipantRole))
	r.Delete("/conversations/{id}/participants/{user_id}", request.Adapt(h.RemoveParticipant))

	r.Get("/conversations/{id}/pinned", request.Adapt(h.GetPinnedMessages))

	r.Post("/conversations/{id}/mute", request.Adapt(h.MuteConversation))
	r.Delete("/conversations/{id}/mute", request.Adapt(h.UnmuteConversation))

	r.Get("/conversations/{id}/unread", request.Adapt(h.GetUnreadCount))

	r.Post("/conversations/{id}/typing", request.Adapt(h.SetTypingIndicator))
	r.Get("/conversations/{id}/typing", request.Adapt(h.GetTypingUsers))

	r.Put("/conversations/{id}/draft", request.Adapt(h.SaveDraft))
	r.Get("/conversations/{id}/draft", request.Adapt(h.GetDraft))
	r.Delete("/conversations/{id}/draft", request.Adapt(h.DeleteDraft))

	r.Get("/conversations/{id}/settings", request.Adapt(h.GetSettings))
	r.Put("/conversations/{id}/settings", request.Adapt(h.UpdateSettings))

	r.Post("/conversations/{id}/invites", request.Adapt(h.CreateInvite))
	r.Get("/conversations/{id}/invites", request.Adapt(h.GetConversationInvites))
	r.Delete("/conversations/{id}/invites/{invite_id}", request.Adapt(h.RevokeInvite))
}

// splitUserConvKey reverses the userConvKey closure used by the search
// rate limiter. Returns (userID, conversationID, ok).
func splitUserConvKey(key string) (string, string, bool) {
	for i := 0; i < len(key); i++ {
		if key[i] == '|' {
			return key[:i], key[i+1:], true
		}
	}
	return "", "", false
}

// registerMessageRoutes wires endpoints under /conversations/{conversation_id}/messages.
//
// Literal segments (search, sync, bookmarks) are registered before /{id}
// so they are not consumed by the {id} param.
func registerMessageRoutes(r *router.Router, h *messageHandler.MessageHandler, limits messageRateLimits) {
	const base = "/conversations/{conversation_id}/messages"

	r.Post(base, request.Adapt(h.SendMessage), limits.send)
	r.Get(base, request.Adapt(h.GetMessages))

	r.Post(base+"/search", request.Adapt(h.SearchMessages), limits.search)
	r.Get(base+"/sync", request.Adapt(h.SyncMessages))
	r.Get(base+"/bookmarks", request.Adapt(h.GetBookmarks))

	r.Get(base+"/{id}", request.Adapt(h.GetMessageByID))
	r.Put(base+"/{id}", request.Adapt(h.EditMessage))
	r.Delete(base+"/{id}", request.Adapt(h.DeleteMessage))

	r.Post(base+"/{id}/reactions", request.Adapt(h.AddReaction), limits.react)
	r.Get(base+"/{id}/reactions", request.Adapt(h.GetReactions))
	r.Delete(base+"/{id}/reactions/{reaction_id}", request.Adapt(h.RemoveReaction))

	r.Post(base+"/{id}/read", request.Adapt(h.MarkAsRead))
	r.Get(base+"/{id}/delivery", request.Adapt(h.GetDeliveryStatus))

	r.Post(base+"/{id}/forward", request.Adapt(h.ForwardMessage), limits.send)

	r.Post(base+"/{id}/pin", request.Adapt(h.PinMessage))
	r.Delete(base+"/{id}/pin", request.Adapt(h.UnpinMessage))

	r.Get(base+"/{id}/thread", request.Adapt(h.GetThread))

	r.Post(base+"/{id}/bookmark", request.Adapt(h.BookmarkMessage))
	r.Delete(base+"/{id}/bookmark", request.Adapt(h.RemoveBookmark))

	r.Post(base+"/{id}/report", request.Adapt(h.ReportMessage), limits.report)
}

// registerPollRoutes wires endpoints under /polls.
func registerPollRoutes(r *router.Router, h *messageHandler.MessageHandler) {
	r.Post("/polls", request.Adapt(h.CreatePoll))
	r.Post("/polls/{id}/vote", request.Adapt(h.VotePoll))
	r.Get("/polls/{id}/results", request.Adapt(h.GetPollResults))
}

// registerInviteRoutes wires the invite-accept endpoint.
func registerInviteRoutes(r *router.Router, h *conversationHandler.ConversationHandler) {
	r.Post("/invites/{code}/accept", request.Adapt(h.AcceptInvite))
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
	outboxRepo := repo.NewOutboxRepository(log)
	_ = outboxRepo // wired into send/edit/delete paths in subsequent steps

	idempotencyMgr := idempotency.NewManager(cacheClient, log)

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
	messageHandler := messageHandler.NewMessageHandler(messageService, reactionService, pollService, bookmarkService, reportService, idempotencyMgr, log)
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
