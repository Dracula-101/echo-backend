package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"ws-service/internal/config"
	"ws-service/internal/consumer"
	"ws-service/internal/dedup"
	"ws-service/internal/dlq"
	codes "ws-service/internal/domain"
	"ws-service/internal/health"
	healthCheckers "ws-service/internal/health/checkers"
	wsmiddleware "ws-service/internal/middleware"
	internalRedis "ws-service/internal/redis"
	"ws-service/internal/repo"
	"ws-service/internal/service"
	wsManager "ws-service/websocket"

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
	"shared/server/middleware"
	"shared/server/request"
	"shared/server/response"
	"shared/server/router"
	"shared/server/server"
	"shared/server/shutdown"
	"shared/server/websocket/handler"

	goRedis "github.com/redis/go-redis/v9"

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

// createRawRedisClient builds a *redis.Client used by ws-service's own Redis
// helpers (pubsub, dedup, pending, etc.). The cache.Cache abstraction doesn't
// expose pub/sub or sorted-set ops; this is parallel access to the same Redis.
func createRawRedisClient(cfg config.RedisConfig, log logger.Logger) (*goRedis.Client, error) {
	rdb := goRedis.NewClient(&goRedis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		MaxRetries:   cfg.MaxRetries,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	log.Info("Raw Redis client connected for pub/sub + ws-service internals")
	return rdb, nil
}

// deriveInstanceID returns a stable-ish identifier for this pod. Hostname under
// k8s is the pod name (StatefulSet) or pod hash (Deployment); add a short uuid
// suffix so two pods that somehow share a hostname can't collide.
func deriveInstanceID() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "ws-service"
	}
	return host + "-" + uuid.NewString()[:8]
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

func createKafkaConsumer(cfg config.KafkaConsumerConfig, log logger.Logger) (messaging.Consumer, error) {
	log.Debug("Creating Kafka consumer",
		logger.Strings("brokers", cfg.Brokers),
		logger.String("group_id", cfg.GroupID),
		logger.String("topic", cfg.Topic),
	)

	kafkaConsumer, err := kafka.NewConsumer(messaging.ConsumerConfig{
		Brokers:           cfg.Brokers,
		ClientID:          cfg.ClientID,
		GroupID:           cfg.GroupID,
		SessionTimeout:    cfg.SessionTimeout,
		HeartbeatInterval: cfg.HeartbeatInterval,
		ReturnOnError:     cfg.ReturnOnError,
		DisableAutoCommit: cfg.DisableAutoCommit,
	})
	if err != nil {
		log.Error("Failed to create Kafka consumer", logger.Error(err))
		return nil, err
	}

	log.Info("Kafka consumer created successfully")
	return kafkaConsumer, nil
}

func createKafkaProducer(cfg config.KafkaProducerConfig, log logger.Logger) (messaging.Producer, error) {
	log.Debug("Creating Kafka producer",
		logger.Strings("brokers", cfg.Brokers),
	)

	producer, err := kafka.NewProducer(messaging.ProducerConfig{
		Brokers:           cfg.Brokers,
		ClientID:          cfg.ClientID,
		Compression:       cfg.Compression,
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

	log.Info("Kafka producer created successfully")
	return producer, nil
}

func startChatMessageConsumer(
	ctx context.Context,
	kafkaConsumer messaging.Consumer,
	chatMessageConsumer *consumer.ChatMessageConsumer,
	topic string,
	log logger.Logger,
) {
	err := kafkaConsumer.Consume(ctx, []string{topic}, chatMessageConsumer)
	if err == nil {
		return
	}

	if ctx.Err() != nil {
		log.Info("Kafka consumer stopped due to context cancellation")
		return
	}
}

// startAuthEventsConsumer subscribes ws-service to the auth-events Kafka topic
// so it can react to session revocations in real time. Returns the consumer
// (so it can be closed on shutdown) and the cancel func for the consumer ctx.
// Returns nil consumer + nil error when topic is empty (feature disabled).
func startAuthEventsConsumer(cfg config.KafkaConsumerConfig, manager *wsManager.Manager, log logger.Logger) (messaging.Consumer, context.CancelFunc, error) {
	if cfg.Topic == "" {
		return nil, nil, nil
	}

	c, err := kafka.NewConsumer(messaging.ConsumerConfig{
		Brokers:           cfg.Brokers,
		ClientID:          cfg.ClientID,
		GroupID:           cfg.GroupID,
		SessionTimeout:    cfg.SessionTimeout,
		HeartbeatInterval: cfg.HeartbeatInterval,
		ReturnOnError:     cfg.ReturnOnError,
		DisableAutoCommit: cfg.DisableAutoCommit,
	})
	if err != nil {
		return nil, nil, err
	}

	handler := consumer.NewSessionEventConsumer(manager, log)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if err := c.Consume(ctx, []string{cfg.Topic}, handler); err != nil {
			if ctx.Err() != nil {
				log.Info("auth-events consumer stopped")
				return
			}
		}
	}()
	log.Info("auth-events consumer started", logger.String("topic", cfg.Topic))
	return c, cancel, nil
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

func createOriginChecker(allowedOrigins []string, log logger.Logger) func(r *http.Request) bool {
	if len(allowedOrigins) == 0 {
		log.Warn("No allowed origins configured - accepting all origins (not recommended for production)")
		return func(r *http.Request) bool { return true }
	}

	if slices.Contains(allowedOrigins, "*") {
		log.Warn("Wildcard origin configured - accepting all origins (not recommended for production)")
		return func(r *http.Request) bool { return true }
	}

	normalized := make([]string, 0, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			normalized = append(normalized, strings.ToLower(origin))
		}
	}

	if len(normalized) == 0 {
		log.Warn("No valid allowed origins after normalization - accepting all origins")
		return func(r *http.Request) bool { return true }
	}

	log.Info("WebSocket origin validation enabled",
		logger.Strings("allowed_origins", normalized),
	)

	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}

		originLower := strings.ToLower(origin)
		allowed := slices.Contains(normalized, originLower)

		if !allowed {
			log.Warn("Rejected WebSocket connection from disallowed origin",
				logger.String("origin", origin),
			)
		}

		return allowed
	}
}

func createWebSocketHandler(
	manager *wsManager.Manager,
	wsService service.WSService,
	cfg *config.Config,
	log logger.Logger,
) *handler.Handler {
	handlerCfg := &handler.Config{
		SendBufferSize:    cfg.WebSocket.ClientBufferSize,
		MaxMessageSize:    int64(cfg.WebSocket.MaxMessageSize),
		PingInterval:      cfg.WebSocket.PingPeriod,
		WriteTimeout:      cfg.WebSocket.WriteWait,
		ReadTimeout:       cfg.WebSocket.PongWait,
		StaleTimeout:      cfg.WebSocket.StaleConnectionTimeout,
		CheckOrigin:       createOriginChecker(cfg.WebSocket.AllowedOrigins, log),
		ReadBufferSize:    cfg.WebSocket.ReadBufferSize,
		WriteBufferSize:   cfg.WebSocket.WriteBufferSize,
		EnableCompression: false,

		ValidateUser: func(ctx context.Context, userID uuid.UUID) (bool, error) {
			if manager.IsDraining() {
				return false, nil
			}
			return wsService.ValidateUserExists(ctx, userID)
		},
		ExtractUserID: handler.DefaultUserIDExtractor,
		HandleMessage: func(ctx context.Context, conn *handler.Connection, message []byte) error {
			return manager.HandleMessage(ctx, conn, message)
		},
		ExtractMetadata: handler.DefaultMetadataExtractor,
		OnConnected: func(conn *handler.Connection) {
			userID, _ := conn.GetMetadata(codes.UserIdMetaKey)
			deviceID, _ := conn.GetMetadata(codes.DeviceIdMetaKey)
			if uid, ok := userID.(uuid.UUID); ok {
				wsService.HandleClientConnect(context.Background(), uid, deviceID.(string))
			}
		},
		OnDisconnected: func(conn *handler.Connection) {
			// Disconnect is handled by manager.SetOnDisconnect which calls wsService.HandleClientDisconnect
			// after hub.Unregister() to ensure correct online status
		},
		SendErrorsToClient: true,
	}

	return handler.New(manager.GetEngine(), handlerCfg, log)
}

func setupAPIRoutes(
	builder *router.Builder,
	wsHandler *handler.Handler,
	rdb *goRedis.Client,
	log logger.Logger,
) *router.Builder {
	log.Debug("Registering API routes")

	upgradeHandler := http.Handler(http.HandlerFunc(wsHandler.HandleUpgrade))
	if rdb != nil {
		// IP-level connection-attempt rate limit applied only to the upgrade
		// route — no point throttling /health or /metrics.
		upgradeHandler = wsmiddleware.IPRateLimit(rdb, wsmiddleware.DefaultIPRateLimitConfig(), log)(upgradeHandler)
	}

	builder = builder.WithRoutes(func(r *router.Router) {
		r.Get("/", upgradeHandler.ServeHTTP)
	})

	log.Debug("API routes registered successfully")
	return builder
}

func createRouter(
	wsHandler *handler.Handler,
	healthHandler *health.Handler,
	rdb *goRedis.Client,
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
			router.Middleware(middleware.RequestReceivedLogger(log)),
		).
		WithLateMiddleware(
			router.Middleware(middleware.Recovery(log)),
			router.Middleware(middleware.RequestCompletedLogger(log)),
		)

	builder = builder.WithRoutes(func(r *router.Router) {
		r.Get("/live", healthHandler.Liveness)
		r.Get("/ready", healthHandler.Readiness)
		r.Get("/health/liveness", healthHandler.Liveness)
		r.Get("/health/readiness", healthHandler.Readiness)
	})

	builder = setupAPIRoutes(builder, wsHandler, rdb, log)

	r := builder.Build()
	return r, nil
}

func setupShutdownManager(
	srv *server.Server,
	manager *wsManager.Manager,
	kafkaConsumer messaging.Consumer,
	kafkaProducer messaging.Producer,
	consumerCancel context.CancelFunc,
	dbClient database.Database,
	cacheClient cache.Cache,
	log logger.Logger,
	cfg *config.Config,
) *shutdown.Manager {
	shutdownMgr := shutdown.New(
		shutdown.WithTimeout(cfg.Server.ShutdownTimeout),
		shutdown.WithLogger(log),
	)

	if kafkaConsumer != nil {
		shutdownMgr.RegisterWithPriority(
			"kafka-consumer",
			shutdown.Hook(func(ctx context.Context) error {
				log.Info("Shutting down Kafka consumer")
				consumerCancel()
				return kafkaConsumer.Close()
			}),
			shutdown.PriorityHigh,
		)
	}

	if kafkaProducer != nil {
		shutdownMgr.RegisterWithPriority(
			"kafka-producer",
			shutdown.Hook(func(ctx context.Context) error {
				log.Info("Closing Kafka producer")
				return kafkaProducer.Close()
			}),
			shutdown.PriorityHigh,
		)
	}

	shutdownMgr.RegisterWithPriority(
		"http-server",
		shutdown.ServerShutdownHook(srv),
		shutdown.PriorityHigh,
	)

	shutdownMgr.RegisterWithPriority(
		"websocket-manager",
		shutdown.Hook(func(ctx context.Context) error {
			log.Info("Shutting down WebSocket manager")
			return manager.Stop()
		}),
		shutdown.PriorityHigh,
	)

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
		_ = shutdownMgr.Wait()
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

	manager := wsManager.NewManager(log, &dbClient)
	log.Info("WebSocket manager initialized")

	if err := manager.Start(); err != nil {
		log.Fatal("Failed to start WebSocket manager", logger.Error(err))
	}
	log.Info("WebSocket engine started")

	// Cross-instance fan-out: a separate raw Redis client + PubSub. Without
	// this, BroadcastToConversation only reaches users connected to this pod.
	var (
		rawRedis     *goRedis.Client
		redisClient  *internalRedis.Client
		redisPubSub  *internalRedis.PubSub
		instanceID   string
		pubsubCancel context.CancelFunc
	)
	if cfg.Cache.Enabled {
		rawRedis, err = createRawRedisClient(cfg.Cache.Redis, log)
		if err != nil {
			log.Fatal("Failed to create raw Redis client", logger.Error(err))
		}
		redisClient = internalRedis.New(rawRedis, log)
		instanceID = deriveInstanceID()
		redisPubSub = internalRedis.NewPubSub(redisClient, instanceID, log)

		var psCtx context.Context
		psCtx, pubsubCancel = context.WithCancel(context.Background())
		if err := redisPubSub.Start(psCtx); err != nil {
			log.Fatal("Failed to start Redis pubsub", logger.Error(err))
		}
		manager.SetPubSub(redisPubSub, instanceID)

		pendingStore := internalRedis.NewPendingStore(redisClient, log)
		manager.SetPendingStore(pendingStore)

		seqCounter := internalRedis.NewSeqCounter(redisClient)
		manager.SetSeqCounter(seqCounter)

		// Resume-token signing key. Uses the same JWT secret as auth-service so
		// no new secret has to be provisioned. Empty value disables resume tokens.
		if secret := env.GetEnv("JWT_SECRET_KEY", ""); secret != "" {
			manager.SetResumeTokenSecret([]byte(secret))
		} else {
			log.Warn("JWT_SECRET_KEY not set — resume tokens disabled")
		}

		log.Info("Cross-instance pubsub + pending buffer + seq counter wired",
			logger.String("instance_id", instanceID),
		)
	} else {
		log.Warn("Cache disabled — running in single-instance mode (cross-instance fan-out + reconnect buffer disabled)")
	}

	// Keyspace listener: safety net for orphaned typing keys after pod crashes.
	// Requires `notify-keyspace-events Kx` set in redis.conf — if Redis isn't
	// configured for it, Subscribe still succeeds and the channel just stays empty.
	var keyspaceListener *internalRedis.KeyspaceListener
	var keyspaceCancel context.CancelFunc
	if redisClient != nil {
		keyspaceListener = internalRedis.NewKeyspaceListener(redisClient, cfg.Cache.Redis.DB, log)
		keyspaceListener.OnPrefixExpired(internalRedis.PrefixTyping, func(key string) {
			log.Debug("typing key expired", logger.String("key", key))
		})

		var ksCtx context.Context
		ksCtx, keyspaceCancel = context.WithCancel(context.Background())
		if err := keyspaceListener.Start(ksCtx); err != nil {
			log.Warn("keyspace listener failed to start (continuing)",
				logger.Error(err),
			)
			keyspaceListener = nil
		}
	}

	kafkaConsumer, err := createKafkaConsumer(cfg.Kafka.Consumer, log)
	if err != nil {
		log.Fatal("Failed to create Kafka consumer", logger.Error(err))
	}

	// Create Kafka producer for delivery events
	kafkaProducer, err := createKafkaProducer(cfg.Kafka.Producer, log)
	if err != nil {
		log.Fatal("Failed to create Kafka producer", logger.Error(err))
	}

	// Create delivery publisher and wire to manager
	deliveryPublisher := service.NewKafkaDeliveryPublisher(kafkaProducer, cfg.Kafka.Producer.DeliveryTopic, log)
	manager.SetDeliveryPublisher(deliveryPublisher)
	log.Info("Delivery publisher wired to WebSocket manager")

	dedupTracker := dedup.NewTracker(time.Hour, 5*time.Minute)
	deliveryStatusRepo := repo.NewDeliveryStatusRepository(dbClient, log)
	manager.SetDeliveryRepo(deliveryStatusRepo)

	dlqTopic := env.GetEnv("KAFKA_DLQ_TOPIC", "chat-messages-dlq")
	dlqPublisher := dlq.New(kafkaProducer, dlqTopic, log)

	chatMessageConsumer := consumer.NewChatMessageConsumer(manager, dedupTracker, deliveryStatusRepo, dlqPublisher, log)

	consumerCtx, consumerCancel := context.WithCancel(context.Background())
	go startChatMessageConsumer(consumerCtx, kafkaConsumer, chatMessageConsumer, cfg.Kafka.Consumer.Topic, log)

	// Auth-events consumer: closes WS connections when auth-service revokes a session.
	authEventsConsumer, authEventsCancel, err := startAuthEventsConsumer(cfg.Kafka.AuthEvents, manager, log)
	if err != nil {
		log.Warn("auth-events consumer not started — session revocation events ignored",
			logger.Error(err),
		)
	}

	userRepo := repo.NewUserRepository(dbClient, log)
	wsService := service.NewWSService(userRepo, cacheClient, manager.GetHub(), log)
	manager.SetWSService(wsService)

	healthMgr := setupHealthChecks(dbClient, cacheClient, cfg)
	healthHandler := health.NewHandler(healthMgr)
	log.Info("Health checks registered")

	wsHandler := createWebSocketHandler(manager, wsService, cfg, log)

	routerInstance, err := createRouter(wsHandler, healthHandler, rawRedis, log)
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

	shutdownMgr := setupShutdownManager(srv, manager, kafkaConsumer, kafkaProducer, consumerCancel, dbClient, cacheClient, log, cfg)

	if redisPubSub != nil {
		shutdownMgr.RegisterWithPriority(
			"redis-pubsub",
			shutdown.Hook(func(ctx context.Context) error {
				log.Info("Stopping Redis pubsub")
				if pubsubCancel != nil {
					pubsubCancel()
				}
				return redisPubSub.Stop()
			}),
			shutdown.PriorityHigh,
		)
	}
	// Drain hook: runs before kafka/ws shutdown so the LB stops routing here
	// and existing clients are nudged to reconnect elsewhere. PriorityCritical
	// would be ideal; using highest available (PriorityHigh) and registering
	// it FIRST so it runs before kafka-consumer.
	if redisClient != nil && instanceID != "" {
		shutdownMgr.RegisterWithPriority(
			"drain-signal",
			shutdown.Hook(func(ctx context.Context) error {
				log.Info("Setting drain signal in Redis",
					logger.String("instance_id", instanceID),
				)
				manager.SetDraining(true)
				if err := redisClient.Set(ctx, internalRedis.DrainKey(instanceID), "1", internalRedis.TTLDrain); err != nil {
					log.Warn("failed to set drain key", logger.Error(err))
				}
				manager.BroadcastShutdown(2000, "rolling_deploy")
				// Give clients ~2s to start reconnecting elsewhere before any
				// further shutdown work begins.
				select {
				case <-ctx.Done():
				case <-time.After(2 * time.Second):
				}
				return nil
			}),
			shutdown.PriorityHigh,
		)
	}
	if rawRedis != nil {
		shutdownMgr.RegisterWithPriority(
			"raw-redis",
			shutdown.Hook(func(ctx context.Context) error {
				log.Info("Closing raw Redis client")
				return rawRedis.Close()
			}),
			shutdown.PriorityNormal,
		)
	}
	shutdownMgr.RegisterWithPriority(
		"dedup-tracker",
		shutdown.Hook(func(ctx context.Context) error {
			dedupTracker.Stop()
			return nil
		}),
		shutdown.PriorityLow,
	)
	if authEventsConsumer != nil {
		shutdownMgr.RegisterWithPriority(
			"auth-events-consumer",
			shutdown.Hook(func(ctx context.Context) error {
				log.Info("Shutting down auth-events consumer")
				if authEventsCancel != nil {
					authEventsCancel()
				}
				return authEventsConsumer.Close()
			}),
			shutdown.PriorityHigh,
		)
	}
	if keyspaceListener != nil {
		shutdownMgr.RegisterWithPriority(
			"keyspace-listener",
			shutdown.Hook(func(ctx context.Context) error {
				log.Info("Stopping keyspace listener")
				if keyspaceCancel != nil {
					keyspaceCancel()
				}
				keyspaceListener.Stop()
				return nil
			}),
			shutdown.PriorityNormal,
		)
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Info("Starting WebSocket Service",
			logger.String("address", srv.Address()),
			logger.String("websocket_endpoint", "ws://"+srv.Address()),
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
		log.Info("WebSocket Service stopped gracefully")
	}
}
