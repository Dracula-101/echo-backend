package main

import (
	"auth-service/api/v1/handler"
	"auth-service/api/v1/middleware"
	ratelimiter "auth-service/api/v1/rate_limiter"
	"auth-service/internal/config"
	"auth-service/internal/health"
	"auth-service/internal/health/checkers"
	repository "auth-service/internal/repo"
	"auth-service/internal/repo/email_verification"
	"auth-service/internal/repo/login_history"
	"auth-service/internal/repo/password_reset"
	"auth-service/internal/repo/session"
	"auth-service/internal/service"
	authCache "auth-service/internal/service/cache"
	"auth-service/internal/service/location"
	sessionSvc "auth-service/internal/service/session"

	"context"
	"fmt"

	"shared/pkg/cache"
	"shared/pkg/cache/memory"
	"shared/pkg/cache/redis"
	"shared/pkg/database"
	"shared/pkg/database/postgres"
	"shared/pkg/logger"
	adapter "shared/pkg/logger/adapter"
	"shared/pkg/windowstore"
	redisWindowStore "shared/pkg/windowstore/redis"
	"shared/server/common/hashing"
	"shared/server/common/token"
	"shared/server/headers"

	email "shared/pkg/email"
	mailgun "shared/pkg/email/mailgun"
	prommetrics "shared/pkg/monitoring/metrics/prometheus"
	env "shared/server/env"
	coreMiddleware "shared/server/middleware"
	req "shared/server/request"
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

func createMemCacheClient(log logger.Logger) cache.Cache {
	log.Info("Initializing in-memory cache")
	memCache := memory.NewMemCache()
	return memCache
}

func createWindowStore(cfg config.Config, log logger.Logger) windowstore.WindowStore {
	log.Info("Initializing in-memory window store for rate limiting")
	windowStore := redisWindowStore.NewRedis(windowstore.Config{
		Addr:     fmt.Sprintf("%s:%d", cfg.Cache.RedisConfig.RedisHost, cfg.Cache.RedisConfig.RedisPort),
		DB:       cfg.Cache.RedisConfig.RedisDB,
		Password: cfg.Cache.RedisConfig.RedisPassword,
	}, log)
	return windowStore
}

func createRateLimiter(cacheClient cache.Cache, windowStore windowstore.WindowStore, log logger.Logger) *ratelimiter.RateLimiter {
	log.Debug("Setting up Rate Limiter middleware with configuration")
	rateLimiter := ratelimiter.NewRateLimiter(cacheClient, windowStore, log)
	return &rateLimiter
}

func setupEmailService(cfg config.Config, log logger.Logger) *email.EmailService {
	log.Debug("Setting up Email service with provider", logger.String("provider", cfg.Email.Provider))
	var emailService email.EmailService
	switch cfg.Email.Provider {
	case "mailgun":
		emailService = mailgun.New(mailgun.MailgunConfig{
			Domain:  cfg.Email.Mailgun.Domain,
			APIKey:  cfg.Email.Mailgun.APIKey,
			BaseURL: cfg.Email.Mailgun.BaseURL,
		})
		log.Info("Mailgun Email service initialized successfully")
	default:
		log.Warn("No valid email provider configured, email functionality will be disabled")
	}
	return &emailService
}

func setupHealthChecks(dbClient database.Database, cacheClient cache.Cache, cfg *config.Config) *health.Manager {
	healthMgr := health.NewManager(cfg.Service.Name, cfg.Service.Version)
	healthMgr.RegisterChecker(checkers.NewDatabaseChecker(dbClient))
	if cfg.Cache.Enabled {
		healthMgr.RegisterChecker(checkers.NewCacheChecker(cacheClient))
	}
	return healthMgr
}

func setupRoutes(builder *router.Builder, h *handler.AuthHandler, rateLimiter ratelimiter.RateLimiter, log logger.Logger) *router.Builder {
	log.Debug("Registering auth routes")
	builder = builder.WithRoutes(func(r *router.Router) {
		r.Post(
			"/register",
			req.Adapt(h.Register),
			middleware.RateLimit(
				func(ctx context.Context, key string, limit int64) (bool, int64, error) {
					return rateLimiter.AllowRegisterIP(ctx, key, limit)
				},
				func(req req.RequestHandler) string {
					return req.GetClientIP()
				},
				5,
			),
		)
		r.Post(
			"/login",
			req.Adapt(h.Login),
			middleware.RateLimit(
				func(ctx context.Context, key string, limit int64) (bool, int64, error) {
					return rateLimiter.AllowLoginIP(ctx, key, limit)
				},
				func(req req.RequestHandler) string {
					return req.GetClientIP()
				},
				10,
			),
		)
		r.Post(
			"/refresh-token",
			req.Adapt(h.RefreshToken),
			middleware.RateLimit(
				func(ctx context.Context, key string, limit int64) (bool, int64, error) {
					if key == "" {
						return true, 0, nil
					}
					return rateLimiter.AllowRefreshDevice(ctx, key, limit)
				},
				func(req req.RequestHandler) string {
					deviceID, _ := req.GetDeviceIDFromHeader()
					return deviceID
				},
				10,
			),
		)
		r.Post(
			"/logout",
			req.Adapt(h.Logout),
		)
		r.Post(
			"/forgot-password",
			req.Adapt(h.ForgotPassword),
			middleware.RateLimitMulti(
				rateLimiter,
				middleware.RateLimitCheck{
					Allow: func(ctx context.Context, key string, limit int64) (bool, int64, error) {
						return rateLimiter.AllowPwdForgotIP(ctx, key, limit)
					},
					Key: func(req req.RequestHandler) string {
						return req.GetClientIP()
					},
					Limit: 5,
				},
				middleware.RateLimitCheck{
					Allow: func(ctx context.Context, key string, limit int64) (bool, int64, error) {
						if key == "" {
							return true, 0, nil
						}
						return rateLimiter.AllowPwdForgotEmail(ctx, key, limit)
					},
					Key: func(req req.RequestHandler) string {
						ok, email := req.GetBodyValue("email")
						if !ok {
							return ""
						}
						return email
					},
					Limit: 5,
				},
			),
		)
		r.Post(
			"/reset-password",
			req.Adapt(h.ResetPassword),
			middleware.RateLimit(
				func(ctx context.Context, key string, limit int64) (bool, int64, error) {
					return rateLimiter.AllowPwdResetIP(ctx, key, limit)
				},
				func(req req.RequestHandler) string {
					return req.GetClientIP()
				},
				10,
			),
		)
		r.Post("/change-password", req.Adapt(h.ChangePassword))
		r.Post("/verify-email", req.Adapt(h.VerifyEmail))
		r.Post(
			"/resend-verification",
			req.Adapt(h.ResendVerification),
			middleware.RateLimit(
				func(ctx context.Context, key string, limit int64) (bool, int64, error) {
					return rateLimiter.AllowResendVerify(ctx, key, limit)
				},
				func(req req.RequestHandler) string {
					return req.GetClientIP()
				},
				5,
			),
		)
		r.Get("/sessions", req.Adapt(h.GetSessions))
		r.Delete("/sessions/{id}", req.Adapt(h.DeleteSession))

		r.Post("/mfa/enable", req.Adapt(h.MFAEnable))
		r.Post("/mfa/disable", req.Adapt(h.MFADisable))
		r.Post("/mfa/verify", req.Adapt(h.MFAVerify))

		r.Delete("/account", req.Adapt(h.DeleteAccount))
	})
	log.Debug("Auth routes registered successfully")
	return builder
}

func createRouter(h *handler.AuthHandler, memCache cache.Cache, locationService location.LocationService, sessionService sessionSvc.SessionService, sessionCache sessionSvc.SessionCache, rateLimiter ratelimiter.RateLimiter, healthHandler *health.Handler, log logger.Logger) (*router.Router, error) {
	skipAuthPaths := []string{"/login", "/register", "/verify-email", "/refresh-token", "/forgot-password", "/reset-password", "/resend-verification"}
	builder := router.NewBuilder().
		WithHealthEndpoint("/health", func(rh req.RequestHandler) {
			healthHandler.Health(rh.Writer(), rh.Request())
		}).
		WithMetricsEndpoint("/metrics", func(rh req.RequestHandler) {
			prommetrics.Handler().ServeHTTP(rh.Writer(), rh.Request())
		}).
		WithNotFoundHandler(func(rh req.RequestHandler) {
			response.NotFoundError(rh.Context(), rh.Request(), rh.Writer(), fmt.Sprintf("Endpoint %s not found", rh.Request().URL.Path))
		}).
		WithMethodNotAllowedHandler(func(rh req.RequestHandler) {
			response.MethodNotAllowedError(rh.Context(), rh.Request(), rh.Writer())
		}).
		WithEarlyMiddleware(
			router.Middleware(middleware.SessionAuth(sessionService, sessionCache, skipAuthPaths...)),
			router.Middleware(middleware.LocationFromIP(locationService, memCache)),
			router.Middleware(coreMiddleware.RequestReceivedLogger(log)),
			router.Middleware(coreMiddleware.InterceptCorrelationID(headers.XCorrelationID)),
			router.Middleware(coreMiddleware.InterceptRequestID(headers.XRequestID)),
			router.Middleware(coreMiddleware.InterceptUserId(skipAuthPaths...)),
			router.Middleware(coreMiddleware.InterceptDeviceId(skipAuthPaths...)),
			router.Middleware(coreMiddleware.InterceptSessionId(skipAuthPaths...)),
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

	builder = setupRoutes(builder, h, rateLimiter, log)
	r := builder.Build()
	return r, nil
}

func setupShutdownManager(srv *server.Server, log logger.Logger, cfg config.Config) *shutdown.Manager {
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

func createTokenManager(cfg config.Config, log logger.Logger) *token.JWTTokenService {
	log.Debug("Creating Token service")
	key, err := token.NewStaticKeySet([]byte(cfg.Auth.JWT.SecretKey))
	if err != nil {
		log.Fatal("Failed to create Token KeySet", logger.Error(err))
	}
	tokenService, err := token.NewJWTTokenService(token.Config{
		KeySet:          key,
		Issuer:          cfg.Auth.JWT.Issuer,
		Audience:        []string{cfg.Auth.JWT.Audience},
		AccessTokenTTL:  cfg.Auth.JWT.AccessTokenTTL,
		RefreshTokenTTL: cfg.Auth.JWT.RefreshTokenTTL,
		Leeway:          cfg.Auth.JWT.Leeway,
	})
	if err != nil {
		log.Fatal("Failed to create Token service", logger.Error(err))
	}
	log.Info("Token Service created successfully")
	return tokenService
}

func createHashingService(cfg config.Config, log logger.Logger) *hashing.HashingService {
	log.Debug("Creating Hashing service")
	hashingService, err := hashing.NewService(hashing.Config{
		Default: hashing.Algorithm(cfg.Auth.Hash.Default),
		Argon2: hashing.Argon2Config{
			SaltLength: uint32(cfg.Auth.Hash.SaltLength),
			Time:       uint32(cfg.Auth.Hash.Iterations),
			Memory:     uint32(64 * 1024), // 64 MB
			Threads:    uint8(4),
			KeyLength:  uint32(cfg.Auth.Hash.KeyLength),
		},
		Bcrypt: hashing.BcryptConfig{
			Cost: cfg.Auth.Hash.Cost,
		},
		Scrypt: hashing.ScryptConfig{
			SaltLength: cfg.Auth.Hash.SaltLength,
			N:          1 << uint8(cfg.Auth.Hash.Iterations),
			R:          8,
			P:          1,
			KeyLength:  cfg.Auth.Hash.KeyLength,
		},
	})
	if err != nil {
		log.Fatal("Failed to create Hashing service", logger.Error(err))
	}
	log.Info("Hashing Service created successfully")
	return hashingService
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

	var memCache cache.Cache
	if cfg.Cache.Enabled {
		log.Info("Initializing in-memory cache since external cache is disabled")
		memCache = createMemCacheClient(log)
		defer func() {
			if memCache != nil {
				log.Info("Closing in-memory cache")
				if err := memCache.Close(); err != nil {
					log.Error("Failed to close in-memory cache", logger.Error(err))
				}
			}
		}()
	}

	var windowStore windowstore.WindowStore
	if cfg.Cache.Enabled {
		windowStore = createWindowStore(*cfg, log)
		defer func() {
			if windowStore != nil {
				log.Info("Closing window store")
				err := windowStore.Close()
				if err != nil {
					log.Error("Failed to close window store", logger.Error(err))
				}
			}
		}()
	}

	authCache := authCache.NewAuthCache(cacheClient, log)
	sessionCache := sessionSvc.NewSessionCache(cacheClient, log)
	rateLimiter := createRateLimiter(cacheClient, windowStore, log)

	emailService := setupEmailService(*cfg, log)
	tokenService := createTokenManager(*cfg, log)
	hashingService := createHashingService(*cfg, log)
	locationService := location.NewLocationService(cfg.LocationService.Endpoint, log)

	loginHistoryRepo := login_history.NewLoginHistoryRepo(dbClient, log)
	sessionRepo := session.NewSessionRepo(dbClient, log)
	sessionService := sessionSvc.NewSessionService(sessionRepo, sessionCache, *tokenService, log, cfg.Cache)
	emailVerificationRepo := email_verification.NewEmailVerificationTokenRepo(dbClient, log)
	resetTokenRepo := password_reset.NewPasswordResetTokenRepo(dbClient, log)

	authRepo := repository.NewAuthRepository(dbClient, log)
	authService := service.NewAuthServiceBuilder().
		WithRepo(authRepo).
		WithLoginHistoryRepo(loginHistoryRepo).
		WithEmailVerificationRepo(emailVerificationRepo).
		WithPasswordResetRepo(resetTokenRepo).
		WithSessionRepo(sessionRepo).
		WithAuthCache(authCache).
		WithSessionCache(sessionCache).
		WithRateLimiter(*rateLimiter).
		WithTokenService(*tokenService).
		WithHashingService(*hashingService).
		WithEmailService(*emailService).
		WithConfig(&cfg.Auth).
		WithLogger(log).
		Build()

	authHandler := handler.NewAuthHandler(authService, sessionService, *locationService, authCache, *rateLimiter, log)

	healthMgr := setupHealthChecks(dbClient, cacheClient, cfg)
	healthHandler := health.NewHandler(healthMgr)

	routerInstance, err := createRouter(authHandler, memCache, *locationService, sessionService, sessionCache, *rateLimiter, healthHandler, log)
	if err != nil {
		log.Fatal("Failed to create router", logger.Error(err))
	}

	serverCfg := server.Config{
		Host:            cfg.Server.Host,
		Port:            cfg.Server.Port,
		ReadTimeout:     cfg.Server.ReadTimeout,
		WriteTimeout:    cfg.Server.WriteTimeout,
		IdleTimeout:     cfg.Server.IdleTimeout,
		MaxHeaderBytes:  cfg.Server.MaxHeaderBytes,
		Handler:         routerInstance.Mux(),
		ShutdownTimeout: cfg.Server.ShutdownTimeout,
	}

	srv, err := server.New(&serverCfg, log)
	if err != nil {
		log.Fatal("Failed to create server", logger.Error(err))
	}

	shutdownMgr := setupShutdownManager(srv, log, *cfg)

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
		if err != nil && !server.IsServerDownErr(err) {
			log.Fatal("Server error", logger.Error(err))
		}
		log.Info("Server stopped")

	case <-waitForShutdown(shutdownMgr):
		log.Info("Auth Service stopped gracefully")
	}
}
