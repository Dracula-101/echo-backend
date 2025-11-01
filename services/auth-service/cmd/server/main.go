package main

import (
	"auth-service/internal/config"
	"auth-service/internal/handler"
	repository "auth-service/internal/repo"
	"auth-service/internal/service"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	"shared/pkg/cache"
	"shared/pkg/cache/redis"
	"shared/pkg/database"
	"shared/pkg/database/postgres"
	"shared/pkg/logger"
	adapter "shared/pkg/logger/adapter"

	env "shared/server/env"
	coreMiddleware "shared/server/middleware"
	"shared/server/response"
	"shared/server/router"
	"shared/server/server"
	"shared/server/shutdown"
)

func loadenv() {
	if err := env.LoadEnv(); err != nil {
		panic(fmt.Sprintf("Failed to load environment variables: %v", err))
	}
}

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

	configPath := os.Getenv("CONFIG_PATH")
	env := os.Getenv("ENV")

	var cfg *config.Config
	var err error

	if configPath != "" {
		if env != "" {
			log.Debug("Loading config with environment", logger.String("env", env))
			cfg, err = config.LoadWithEnv(configPath, env)
		} else {
			log.Debug("Loading config from path", logger.String("configPath", configPath))
			cfg, err = config.Load(configPath)
		}
	} else {
		log.Debug("Loading config from environment variables")
		cfg, err = config.LoadFromEnv()
	}
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

func setupRoutes(builder *router.Builder, h *handler.AuthHandler, log logger.Logger) *router.Builder {
	log.Debug("Registering auth routes")
	builder.WithRoutes(func(r *router.Router) {
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
		r.Post("/logout", h.Logout)
		r.Post("/refresh", h.RefreshToken)
		r.Post("/verify-email", h.VerifyEmail)
		r.Post("/resend-verification", h.ResendVerification)
		r.Post("/forgot-password", h.ForgotPassword)
		r.Post("/reset-password", h.ResetPassword)
	})
	log.Debug("Auth routes registered successfully")
	return builder
}

func createRouter(h *handler.AuthHandler, log logger.Logger) (*router.Router, error) {
	builder := router.NewBuilder().
		WithHealthEndpoint("/health", func(w http.ResponseWriter, r *http.Request) {
			response.JSONWithMessage(r.Context(), r, w, http.StatusOK, "Health endpoint", nil)
		}).
		WithNotFoundHandler(func(w http.ResponseWriter, r *http.Request) {
			response.RouteNotFoundError(r.Context(), r, w)
		}).
		WithMethodNotAllowedHandler(func(w http.ResponseWriter, r *http.Request) {
			response.MethodNotAllowedError(r.Context(), r, w)
		}).
		WithEarlyMiddleware(
			router.Middleware(coreMiddleware.RequestReceivedLogger(log)),
		).
		WithLateMiddleware(
			router.Middleware(coreMiddleware.Recovery(log)),
			router.Middleware(coreMiddleware.RequestCompletedLogger(log)),
		)

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

	authRepo := repository.NewAuthRepository(dbClient, log)
	authService := service.NewAuthService(authRepo, cacheClient, cfg, log)
	authHandler := handler.NewAuthHandler(authService, log)

	routerInstance, err := createRouter(authHandler, log)
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
		log.Info("Starting Auth Service server",
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
		log.Info("Auth Service stopped gracefully")
	}
}
