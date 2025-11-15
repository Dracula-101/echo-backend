package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"user-service/api/v1/handler"
	"user-service/internal/config"
	"user-service/internal/health"
	"user-service/internal/health/checkers"
	repository "user-service/internal/repo"
	"user-service/internal/service"

	"shared/pkg/cache"
	"shared/pkg/cache/redis"
	"shared/pkg/database"
	"shared/pkg/database/postgres"
	"shared/pkg/logger"
	adapter "shared/pkg/logger/adapter"

	"shared/server/common/token"
	env "shared/server/env"
	coreMiddleware "shared/server/middleware"
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

func setupRoutes(builder *router.Builder, h *handler.UserHandler, log logger.Logger) *router.Builder {
	log.Debug("Registering user routes")
	builder = builder.WithRoutes(func(r *router.Router) {
		r.Post("/profile", h.CreateProfile)
		r.Get("/profile/{user_id}", h.GetProfile)
	})
	log.Debug("User routes registered successfully")
	return builder
}

func createRouter(h *handler.UserHandler, healthHandler *health.Handler, log logger.Logger) (*router.Router, error) {
	builder := router.NewBuilder().
		WithHealthEndpoint("/health", healthHandler.Health).
		WithNotFoundHandler(func(w http.ResponseWriter, r *http.Request) {
			response.RouteNotFoundError(r.Context(), r, w, log)
		}).
		WithMethodNotAllowedHandler(func(w http.ResponseWriter, r *http.Request) {
			response.MethodNotAllowedError(r.Context(), r, w)
		}).
		WithEarlyMiddleware(
			router.Middleware(coreMiddleware.RequestReceivedLogger(log)),
			router.Middleware(coreMiddleware.InterceptUserId()),
			router.Middleware(coreMiddleware.InterceptSessionId()),
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

	builder = setupRoutes(builder, h, log)
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

	tokenService := createTokenService(cfg, log)

	userRepo := repository.NewUserRepository(dbClient, log)
	userService := service.NewUserServiceBuilder().
		WithRepo(userRepo).
		WithCache(cacheClient).
		WithLogger(log).
		Build()
	locationService := service.NewLocationService(cfg.Server.LocationServiceEndpoint, log)
	userHandler := handler.NewUserHandler(userService, locationService, tokenService, log)

	healthMgr := setupHealthChecks(dbClient, cacheClient, cfg)
	healthHandler := health.NewHandler(healthMgr)

	routerInstance, err := createRouter(userHandler, healthHandler, log)
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
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("Server error", logger.Error(err))
		}
		log.Info("Server stopped")

	case <-waitForShutdown(shutdownMgr):
		log.Info("User Service stopped gracefully")
	}
}
