package main

import (
	"context"
	"fmt"
	"user-service/api/v1/handler"
	"user-service/api/v1/middleware"
	ratelimiter "user-service/api/v1/rate_limiter"
	"user-service/internal/config"
	"user-service/internal/consumer"
	"user-service/internal/health"
	"user-service/internal/health/checkers"
	repository "user-service/internal/repo"
	"user-service/internal/repo/blocked_users"
	"user-service/internal/repo/contacts"
	"user-service/internal/repo/privacy_overrides"
	"user-service/internal/repo/settings"
	"user-service/internal/service"
	userCache "user-service/internal/service/cache"
	"user-service/internal/service/contact"
	"user-service/internal/service/location"
	"user-service/internal/service/search"

	"shared/pkg/cache"
	"shared/pkg/cache/memory"
	"shared/pkg/cache/redis"
	"shared/pkg/database"
	"shared/pkg/database/postgres"
	"shared/pkg/logger"
	adapter "shared/pkg/logger/adapter"
	"shared/pkg/messaging"
	"shared/pkg/messaging/kafka"
	windowStore "shared/pkg/windowstore/memory"

	prommetrics "shared/pkg/monitoring/metrics/prometheus"
	searchClient "shared/pkg/search"
	"shared/pkg/search/meilisearch"
	"shared/server/common/token"
	env "shared/server/env"
	"shared/server/headers"
	coreMiddleware "shared/server/middleware"
	"shared/server/request"
	"shared/server/response"
	"shared/server/router"
	"shared/server/server"
	"shared/server/shutdown"
)

func createLogger(name string) logger.Logger {
	log, err := adapter.NewZap(logger.Config{
		Level:      logger.GetLoggerLevel(),
		Format:     logger.GetLoggerFormat(),
		Output:     logger.GetLoggerOutput(),
		TimeFormat: logger.GetLoggerTimeFormat(),
		Service:    name,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create logger: %v", err))
	}
	return log
}

func loadConfig() (*config.Config, error) {
	log := createLogger("config-loader")
	defer log.Sync()

	configPath := env.GetEnv("CONFIG_PATH")
	env := env.GetEnv("APP_ENV")

	var cfg *config.Config
	var err error
	log.Debug("Loading config from file",
		logger.String("configPath", configPath),
		logger.String("environment", env),
	)
	cfg, err = config.Load(configPath, env)
	if err != nil {
		log.Error("Failed to load config", logger.Error(err))
		return nil, err
	}
	log.Debug("Config loaded successfully")
	return cfg, nil
}

func createDBClient(dbConfig config.DatabaseConfig, log logger.Logger) (database.Database, error) {
	log.Debug("Creating Postgres client - configuration",
		logger.String("host", dbConfig.Postgres.Host),
		logger.Int("port", dbConfig.Postgres.Port),
		logger.String("user", dbConfig.Postgres.User),
		logger.String("password", dbConfig.Postgres.Password),
		logger.String("database", dbConfig.Postgres.DBName),
	)
	dbClient, err := postgres.New(database.Config{
		Host:            dbConfig.Postgres.Host,
		Port:            dbConfig.Postgres.Port,
		User:            dbConfig.Postgres.User,
		Password:        dbConfig.Postgres.Password,
		Database:        dbConfig.Postgres.DBName,
		SSLMode:         dbConfig.Postgres.SSLMode,
		MaxOpenConns:    dbConfig.Postgres.MaxOpenConns,
		MaxIdleConns:    dbConfig.Postgres.MaxIdleConns,
		ConnMaxLifetime: dbConfig.Postgres.ConnMaxLifetime,
		ConnMaxIdleTime: dbConfig.Postgres.ConnMaxIdleTime,
	})
	if err != nil {
		log.Error("Failed to create Postgres client", logger.Error(err))
		return nil, err
	}
	log.Info("Postgres client created successfully")
	return dbClient, nil
}

func createCacheClient(cacheConfig config.CacheConfig, log logger.Logger) (cache.Cache, error) {
	log.Debug("Creating Redis cache client - configuration",
		logger.String("host", cacheConfig.RedisConfig.RedisHost),
		logger.Int("port", cacheConfig.RedisConfig.RedisPort),
		logger.String("password", cacheConfig.RedisConfig.RedisPassword),
		logger.Int("db", cacheConfig.RedisConfig.RedisDB),
	)
	cacheClient, err := redis.New(cache.Config{
		Host:         cacheConfig.RedisConfig.RedisHost,
		Port:         cacheConfig.RedisConfig.RedisPort,
		Password:     cacheConfig.RedisConfig.RedisPassword,
		DB:           cacheConfig.RedisConfig.RedisDB,
		DialTimeout:  cacheConfig.RedisConfig.RedisDialTimeout,
		PoolSize:     cacheConfig.RedisConfig.RedisPoolSize,
		MinIdleConns: cacheConfig.RedisConfig.RedisMinIdleConns,
	})
	if err != nil {
		log.Error("Failed to create Redis client", logger.Error(err))
		return nil, err
	}
	log.Info("Redis client created successfully")
	return cacheClient, nil
}

func createSearchClient(cfg *config.Config, log logger.Logger) (searchClient.Search, error) {
	log.Debug("Creating Meilisearch client - configuration",
		logger.String("host", cfg.Search.Meilisearch.Host),
	)
	searchClient, err := meilisearch.New(meilisearch.Config{
		Host:    cfg.Search.Meilisearch.Host,
		APIKey:  cfg.Search.Meilisearch.APIKey,
		Timeout: cfg.Search.Meilisearch.Timeout,
	}, log)
	if err != nil {
		log.Error("Failed to create Meilisearch client", logger.Error(err))
		return nil, err
	}
	log.Info("Meilisearch client created successfully")
	return searchClient, nil
}

func createConsumer(cfg *config.Config, userRepo repository.UserRepository, log logger.Logger) (*consumer.MediaEventsConsumer, error) {
	c := cfg.Kafka.Consumer
	log.Debug("Creating messaging consumer - configuration",
		logger.Any("brokers", c.Brokers),
		logger.String("topic", c.Topic),
		logger.String("groupID", c.GroupID),
	)
	kafkaConsumer, err := kafka.NewConsumer(messaging.ConsumerConfig{
		Brokers:           c.Brokers,
		ClientID:          c.ClientID,
		GroupID:           c.GroupID,
		SessionTimeout:    c.SessionTimeout,
		HeartbeatInterval: c.HeartbeatInterval,
	})
	if err != nil {
		log.Error("Failed to create messaging consumer", logger.Error(err))
		return nil, err
	}
	mediaConsumer := consumer.NewMediaEventConsumer(kafkaConsumer, c, userRepo, log)
	log.Info("Messaging consumer created successfully")

	// start the consumer
	go func() {
		if err := mediaConsumer.Start(context.Background()); err != nil {
			log.Error("Failed to start media event consumer", logger.Error(err))
		} else {
			log.Info("Media event consumer started successfully")
		}
	}()
	return &mediaConsumer, nil
}

func createTokenService(cfg *config.Config, log logger.Logger) *token.JWTTokenService {
	keyset, err := token.NewStaticKeySet([]byte(cfg.JWT.SecretKey))
	if err != nil {
		log.Fatal("Failed to create JWT keyset", logger.Error(err))
		return nil
	}

	tokenService, err := token.NewJWTTokenService(token.Config{
		KeySet:          keyset,
		Issuer:          cfg.JWT.Issuer,
		Audience:        []string{cfg.JWT.Audience},
		AccessTokenTTL:  cfg.JWT.AccessTokenTTL,
		RefreshTokenTTL: cfg.JWT.RefreshTokenTTL,
	})
	if err != nil {
		log.Fatal("Failed to create JWT token service", logger.Error(err))
		return nil
	}
	return tokenService
}

func setupHealthChecks(dbClient database.Database, cacheClient cache.Cache, cfg *config.Config) *health.Manager {
	healthMgr := health.NewManager(cfg.Service.Name, cfg.Service.Version)

	// Register database health checker
	if dbClient != nil {
		healthMgr.RegisterChecker(checkers.NewDatabaseChecker(dbClient))
	}

	// Register cache health checker
	if cacheClient != nil && cfg.Cache.Enabled {
		healthMgr.RegisterChecker(checkers.NewCacheChecker(cacheClient))
		healthMgr.RegisterChecker(checkers.NewCachePerformanceChecker(cacheClient))
	}
	return healthMgr
}

func setupRoutes(builder *router.Builder, h *handler.UserHandler, ratelimiter ratelimiter.RateLimiter, log logger.Logger) *router.Builder {
	log.Debug("Registering user routes")

	// Profile routes
	builder = builder.WithRoutes(func(r *router.Router) {
		r.Get(
			"/me",
			request.Adapt(h.GetMyProfile),
		)
		r.Post(
			"/me",
			request.Adapt(h.CreateMyProfile),
		)
		r.Put(
			"/me",
			request.Adapt(h.UpdateMyProfile),
			middleware.RateLimit(
				func(ctx context.Context, key string, limit int64) (bool, int64, error) {
					return ratelimiter.AllowProfileUpdate(ctx, key, limit)
				},
				func(req request.RequestHandler) string {
					userId, _ := request.GetUserIDFromContext(req.Context())
					return userId
				},
				10,
			),
		)
		r.Get(
			"/@{username}",
			request.Adapt(h.GetProfile),
		)
		// r.Delete("/profile", request.Adapt(h.DeleteProfile))
		r.Get(
			"/search",
			request.Adapt(h.SearchProfiles),
			middleware.RateLimit(
				func(ctx context.Context, key string, limit int64) (bool, int64, error) {
					return ratelimiter.AllowSearch(ctx, key, limit)
				},
				func(req request.RequestHandler) string {
					userId, _ := request.GetUserIDFromContext(req.Context())
					return userId
				},
				30,
			),
		)
	})

	// Contact routes
	builder = builder.WithRoutesGroup("/me", func(rg *router.RouteGroup) {
		rg.Get(
			"/contacts",
			request.Adapt(h.ListContacts),
		)
		rg.Post(
			"/contacts",
			request.Adapt(h.AddContact),
			middleware.RateLimit(
				func(ctx context.Context, key string, limit int64) (bool, int64, error) {
					return ratelimiter.AllowContactAdd(ctx, key, limit)
				},
				func(req request.RequestHandler) string {
					user, _ := request.GetUserIDFromContext(req.Context())
					return user
				},
				20,
			),
		)
		rg.Put(
			"/contacts/{contact_id}",
			request.Adapt(h.UpdateContact),
		)
		rg.Delete(
			"/contacts/{contact_id}",
			request.Adapt(h.RemoveContact),
		)
	})

	builder = builder.WithRoutesGroup("/me/contacts", func(rg *router.RouteGroup) {
		rg.Post(
			"/{contact_id}/accept",
			request.Adapt(h.AcceptContact),
		)
	})

	// Contact group routes
	builder = builder.WithRoutes(func(r *router.Router) {
		r.Get("/contacts/groups", request.Adapt(h.ListContactGroups))
		r.Post("/contacts/groups", request.Adapt(h.CreateContactGroup))
		r.Put("/contacts/groups/{group_id}", request.Adapt(h.UpdateContactGroup))
		r.Delete("/contacts/groups/{group_id}", request.Adapt(h.DeleteContactGroup))
	})

	// Settings routes
	builder = builder.WithRoutes(func(r *router.Router) {
		r.Get("/settings", request.Adapt(h.GetSettings))
		r.Put("/settings", request.Adapt(h.UpdateSettings))
	})

	// Blocked users routes
	builder = builder.WithRoutes(func(r *router.Router) {
		r.Get("/blocked", request.Adapt(h.ListBlocked))
		r.Post("/blocked", request.Adapt(h.BlockUser))
		r.Delete("/blocked/{user_id}", request.Adapt(h.UnblockUser))
	})

	// Status routes
	builder = builder.WithRoutes(func(r *router.Router) {
		r.Post("/status", request.Adapt(h.SetStatus))
		r.Get("/status/{user_id}", request.Adapt(h.GetStatus))
		r.Delete("/status/{status_id}", request.Adapt(h.DeleteStatus))
		r.Get("/status/{status_id}/views", request.Adapt(h.GetStatusViews))
	})

	// Device routes
	builder = builder.WithRoutes(func(r *router.Router) {
		r.Get("/devices", request.Adapt(h.ListDevices))
		r.Put("/devices/{device_id}", request.Adapt(h.UpdateDevice))
		r.Delete("/devices/{device_id}", request.Adapt(h.RemoveDevice))
	})

	// Preferences routes
	builder = builder.WithRoutes(func(r *router.Router) {
		r.Get("/preferences", request.Adapt(h.GetPreferences))
		r.Put("/preferences", request.Adapt(h.UpdatePreferences))
	})

	// Achievements routes
	builder = builder.WithRoutes(func(r *router.Router) {
		r.Get("/achievements", request.Adapt(h.ListAchievements))
	})

	// Reports routes
	builder = builder.WithRoutes(func(r *router.Router) {
		r.Post("/reports", request.Adapt(h.SubmitReport))
	})

	log.Debug("User routes registered successfully")
	return builder
}

func createRouter(h *handler.UserHandler, locationService location.LocationService, memCache cache.Cache, ratelimiter ratelimiter.RateLimiter, healthHandler *health.Handler, log logger.Logger) (*router.Router, error) {
	skipPaths := []string{"/health", "/metrics", "/live", "/ready", "/health/liveness", "/health/readiness"}
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
			router.Middleware(middleware.LocationFromIP(locationService, memCache, log)),
			router.Middleware(coreMiddleware.RequestReceivedLogger(log)),
			router.Middleware(coreMiddleware.InterceptRequestID(headers.XRequestID)),
			router.Middleware(coreMiddleware.InterceptCorrelationID(headers.XCorrelationID)),
			router.Middleware(coreMiddleware.InterceptUserId(skipPaths...)),
			router.Middleware(coreMiddleware.InterceptSessionId(skipPaths...)),
			router.Middleware(coreMiddleware.InterceptSessionToken()),
		).
		WithLateMiddleware(
			router.Middleware(coreMiddleware.Recovery(log)),
			router.Middleware(coreMiddleware.RequestCompletedLogger(log)),
		)

	builder = builder.WithRoutes(func(r *router.Router) {
		r.Get("/live", healthHandler.Liveness)
		r.Get("/ready", healthHandler.Readiness)
		r.Get("/health/liveness", healthHandler.Liveness)
		r.Get("/health/readiness", healthHandler.Readiness)
	})

	builder = setupRoutes(builder, h, ratelimiter, log)
	r := builder.Build()
	return r, nil
}

func setupShutdownManager(srv *server.Server, log logger.Logger, cfg *config.Config) *shutdown.Manager {
	shutdownMgr := shutdown.New(
		shutdown.WithTimeout(cfg.Server.ShutdownTimeout),
		shutdown.WithLogger(log),
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

	var searchClient searchClient.Search
	searchClient, err = createSearchClient(cfg, log)
	if err != nil {
		log.Fatal("Failed to create search client", logger.Error(err))
	}

	memCache := memory.NewMemCache()

	tokenService := createTokenService(cfg, log)
	userCache := userCache.NewUserCache(cacheClient, log)
	userRepo := repository.NewUserRepository(dbClient, log)
	blockedRepo := blocked_users.NewBlockedRepository(dbClient, log)
	contactsRepo := contacts.NewContactRepository(dbClient, log)
	settingsRepo := settings.NewSettingsRepository(dbClient, log)
	privacyRepo := privacy_overrides.NewPrivacyOverrideRepository(dbClient, log)
	// deviceRepo := devices.NewDeviceRepository(dbClient, log)

	consumer, err := createConsumer(cfg, userRepo, log)
	if err != nil {
		log.Fatal("Failed to create messaging consumer", logger.Error(err))
	}
	if consumer == nil {
		log.Fatal("Messaging consumer is nil after creation")
	}
	windowStore := windowStore.NewMemory(log)
	ratelimiter := ratelimiter.NewRateLimiter(cacheClient, windowStore, log)

	locationService := location.NewLocationService(cfg.Server.LocationServiceEndpoint, log)
	userService := service.NewUserService(userRepo, blockedRepo, contactsRepo, settingsRepo, privacyRepo, userCache, log)
	searchService := search.NewSearchService(userRepo, searchClient, userCache, log)
	go searchService.CreateIndex(context.Background())
	contactService := contact.NewContactService(contactsRepo, userRepo, blockedRepo, log)
	userHandler := handler.NewUserHandler(userService, searchService, contactService, locationService, userCache, tokenService, log)

	healthMgr := setupHealthChecks(dbClient, cacheClient, cfg)
	healthHandler := health.NewHandler(healthMgr)

	routerInstance, err := createRouter(userHandler, locationService, memCache, ratelimiter, healthHandler, log)
	if err != nil {
		log.Fatal("Failed to create router", logger.Error(err))
	}

	serverCfg := server.Config{
		Host:           cfg.Server.Host,
		Port:           cfg.Server.Port,
		ReadTimeout:    cfg.Server.ReadTimeout,
		WriteTimeout:   cfg.Server.WriteTimeout,
		IdleTimeout:    cfg.Server.IdleTimeout,
		MaxHeaderBytes: cfg.Server.MaxHeaderBytes,
		Handler:        routerInstance.Mux(),
	}

	srv, err := server.New(&serverCfg, log)
	if err != nil {
		log.Fatal("Failed to create server", logger.Error(err))
	}

	shutdownMgr := setupShutdownManager(srv, log, cfg)

	serverErrors := make(chan error, 1)
	go func() {
		log.Info("Starting User Service server",
			logger.String("host", cfg.Server.Host),
			logger.Int("port", cfg.Server.Port),
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
		log.Info("User Service stopped gracefully")
	}
}
