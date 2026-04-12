package main

import (
	"context"
	"fmt"
	"time"

	"echo-backend/services/api-gateway/internal/config"
	healthHandler "echo-backend/services/api-gateway/internal/health"
	"echo-backend/services/api-gateway/internal/proxy"
	"shared/pkg/logger"
	adapter "shared/pkg/logger/adapter"

	prommetrics "shared/pkg/monitoring/metrics/prometheus"
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

func loadenv() {
	env.LoadEnv()
}

func createLogger(serviceName string) logger.Logger {
	log, err := adapter.NewZap(logger.Config{
		Level:      logger.GetLoggerLevel(),
		Format:     logger.GetLoggerFormat(),
		Output:     logger.GetLoggerOutput(),
		TimeFormat: logger.GetLoggerTimeFormat(),
		Service:    serviceName,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create logger: %v", err))
	}
	return log
}

func loadConfig() (*config.Config, error) {
	configLogger := createLogger("config-loader")
	defer configLogger.Sync()

	configPath := env.GetEnv("CONFIG_PATH")
	env := env.GetEnv("ENV")
	var cfg *config.Config
	var err error
	configLogger.Info("Loading configuration",
		logger.String("configPath", configPath),
		logger.String("environment", env),
	)
	cfg, err = config.Load(configPath, env)
	if err != nil {
		configLogger.Error("Failed to load configuration", logger.Error(err))
		return nil, err
	}
	if err := config.ValidateAndSetDefaults(cfg); err != nil {
		configLogger.Error("Invalid configuration", logger.Error(err))
		return nil, err
	}
	configLogger.Debug("Config loaded successfully")
	return cfg, nil
}

func createProxyManager(cfg *config.Config, log logger.Logger) (*proxy.Manager, error) {
	proxyManager, err := proxy.NewManager(cfg, log)
	if err != nil {
		log.Error("Failed to create proxy manager", logger.Error(err))
		return nil, err
	}
	return proxyManager, nil
}

func GlobalRateLimitingMiddleware(cfg *config.Config, log logger.Logger) coreMiddleware.Handler {
	var rateLimiter coreMiddleware.Handler
	if cfg.RateLimit.Enabled {
		switch cfg.RateLimit.Global.Strategy {
		case "fixed_window":
			rateLimiter = coreMiddleware.FixedWindowRateLimit(
				cfg.RateLimit.Global.Requests,
				cfg.RateLimit.Global.Window,
			)
		case "sliding_window":
			rateLimiter = coreMiddleware.SlidingWindowRateLimit(
				cfg.RateLimit.Global.Requests,
				cfg.RateLimit.Global.Window,
			)
		case "token_bucket":
			rateLimiter = coreMiddleware.TokenBucketRateLimit(
				cfg.RateLimit.Global.Requests,
				cfg.RateLimit.Global.Window,
			)
		default:
			log.Warn("Unknown global rate limiting strategy, skipping rate limiter setup",
				logger.String("strategy", cfg.RateLimit.Global.Strategy))
		}
	}
	return rateLimiter
}

func setupMiddleware(r *router.Builder, cfg *config.Config, log logger.Logger) *router.Builder {
	commonChain := coreMiddleware.NewChain()
	commonChain.Append(coreMiddleware.Recovery(log))
	commonChain.Append(coreMiddleware.SecurityHeaders(cfg.Security.SecurityHeaders))
	commonChain.Append(coreMiddleware.APIVersion(headers.XAPIVersion, cfg.Service.Version))
	commonChain.Append(coreMiddleware.CORS(cfg.Security.AllowedOrigins, cfg.Security.AllowedMethods, cfg.Security.AllowedHeaders))
	commonChain.Append(coreMiddleware.CacheControl(cfg.Security.MaxAge, true))
	// Body limit removed for API Gateway - backend services handle their own body size validation
	// Applying MaxBytesReader at gateway breaks reverse proxy for large uploads
	if cfg.Server.EnableCompression {
		commonChain.Append(coreMiddleware.Compression(coreMiddleware.CompressionConfig{
			Level:   4,
			MinSize: 1024,
			ContentTypes: []string{
				"application/json",
				"text/plain",
				"text/html",
				"text/css",
				"application/javascript",
				"application/xml",
			},
		}))
	}
	r = r.WithMiddlewareChain(commonChain)
	return r
}

func createTokenService(cfg *config.Config, log logger.Logger) *token.JWTTokenService {
	jwtCfg := cfg.Server.JWTConfig
	key, err := token.NewStaticKeySet([]byte(jwtCfg.SecretKey))
	if err != nil {
		log.Fatal("Failed to create JWT key set", logger.Error(err))
	}
	tokenService, err := token.NewJWTTokenService(token.Config{
		KeySet:          key,
		AccessTokenTTL:  jwtCfg.AccessTokenTTL,
		RefreshTokenTTL: jwtCfg.RefreshTokenTTL,
		Issuer:          jwtCfg.Issuer,
		Audience:        []string{jwtCfg.Audience},
		Leeway:          jwtCfg.Leeway,
	})
	if err != nil {
		log.Fatal("Failed to create JWT token service", logger.Error(err))
	}
	return tokenService
}

func setupRoutes(builder *router.Builder, cfg *config.Config, proxyManager *proxy.Manager, log logger.Logger) *router.Builder {
	for _, routeGroup := range cfg.RouterGroups {
		log.Info("Queueing proxy routes",
			logger.String("prefix", routeGroup.Prefix),
			logger.String("service", routeGroup.Service),
			logger.Any("methods", routeGroup.Methods),
		)

		prefix := cfg.Server.PrefixPath + routeGroup.Prefix
		service := routeGroup.Service
		transform := routeGroup.Transform
		methods := routeGroup.Methods
		if len(methods) == 0 {
			methods = []string{
				request.MethodGet,
				request.MethodPost,
				request.MethodPut,
				request.MethodDelete,
				request.MethodPatch,
				request.MethodOptions,
			}
		}

		builder = builder.WithRoutesGroup(prefix, func(rg *router.RouteGroup) {
			rg.HandleProxy(proxyManager.ProxyHandler(service, transform), methods...)
		})
	}
	return builder
}

func createRouter(cfg *config.Config, proxyManager *proxy.Manager, tokenService *token.JWTTokenService, log logger.Logger) (*router.Router, error) {
	healthHandler := healthHandler.NewHandler(healthHandler.NewManager(cfg.Service.Name, cfg.Service.Version))
	builder := router.NewBuilder().
		WithSystemPrefix(cfg.Server.PrefixPath).
		WithHealthEndpoint("/health", func(rh request.RequestHandler) {
			healthHandler.Health(rh.Writer(), rh.Request())
		}).
		WithMetricsEndpoint("/metrics", func(rh request.RequestHandler) {
			prommetrics.Handler().ServeHTTP(rh.Writer(), rh.Request())
		}).
		WithNotFoundHandlerRequest(func(rh request.RequestHandler) {
			response.NotFoundError(rh.Context(), rh.Request(), rh.Writer(), fmt.Sprintf("The requested URL %s was not found on this server.", rh.Request().URL.Path))
		}).
		WithMethodNotAllowedHandlerRequest(func(rh request.RequestHandler) {
			response.MethodNotAllowedError(rh.Context(), rh.Request(), rh.Writer())
		}).
		WithEarlyMiddleware(
			router.Middleware(coreMiddleware.RequestID(headers.XRequestID)),
			router.Middleware(coreMiddleware.CorrelationID(headers.XCorrelationID)),
			router.Middleware(coreMiddleware.RequestReceivedLogger(log)),
			router.Middleware(GlobalRateLimitingMiddleware(cfg, log)),
			router.Middleware(coreMiddleware.Auth(coreMiddleware.AuthConfig{
				ValidateToken: func(tokenStr string) (userID string, err error) {
					claims, err := tokenService.Validate(context.Background(), tokenStr, token.TokenTypeAccess)
					if err != nil {
						return "", err
					}
					return claims.Subject, nil
				},
				OnAuthFailed: func(handler request.RequestHandler, err error) {
					response.UnauthorizedError(handler.Context(), handler.Request(), handler.Writer(), fmt.Sprintf("Authentication failed: %v", err), err)
				},
				SkipPaths: cfg.Server.JWTConfig.SkipPaths,
			})),
		).
		WithLateMiddleware(
			router.Middleware(coreMiddleware.RequestCompletedLogger(log)),
		)

	builder = setupMiddleware(builder, cfg, log)
	builder = setupRoutes(builder, cfg, proxyManager, log)
	r := builder.Build()
	return r, nil
}

func setupShutdown(srv *server.Server, proxyManager *proxy.Manager, log logger.Logger, cfg *config.Config) *shutdown.Manager {
	shutdownMgr := shutdown.New(
		shutdown.WithTimeout(cfg.Shutdown.Timeout),
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

	shutdownMgr.RegisterWithOptions(
		"proxy-manager",
		func(ctx context.Context) error {
			if closer, ok := interface{}(proxyManager).(interface{ Close() error }); ok {
				return closer.Close()
			}
			log.Debug("Proxy manager cleanup - no Close method")
			return nil
		},
		shutdown.PriorityNormal,
		10*time.Second,
	)

	shutdownMgr.RegisterWithPriority(
		"logger-sync",
		func(ctx context.Context) error {
			return log.Sync()
		},
		shutdown.PriorityLow,
	)

	return shutdownMgr
}

func waitForShutdown(mgr *shutdown.Manager) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := mgr.Wait(); err != nil {
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

	log.Info("Starting API Gateway",
		logger.String("service", cfg.Service.Name),
		logger.String("version", cfg.Service.Version),
		logger.String("environment", cfg.Service.Environment),
	)

	proxyManager, err := createProxyManager(cfg, log)
	if err != nil {
		log.Fatal("Failed to create proxy manager", logger.Error(err))
	}

	tokenService := createTokenService(cfg, log)

	routerInstance, err := createRouter(cfg, proxyManager, tokenService, log)
	if err != nil {
		log.Fatal("Failed to create router", logger.Error(err))
	}

	serverCfg := &server.Config{
		Port:            cfg.Server.Port,
		Host:            cfg.Server.Host,
		ReadTimeout:     cfg.Server.ReadTimeout,
		WriteTimeout:    cfg.Server.WriteTimeout,
		IdleTimeout:     cfg.Server.IdleTimeout,
		ShutdownTimeout: cfg.Shutdown.Timeout,
		MaxHeaderBytes:  cfg.Server.MaxHeaderBytes,
		Handler:         routerInstance.Mux(),
	}

	if cfg.Server.TLSEnabled {
		serverCfg.TLSEnabled = true
		serverCfg.TLSCertFile = cfg.Server.TLSCertFile
		serverCfg.TLSKeyFile = cfg.Server.TLSKeyFile
	}

	srv, err := server.New(serverCfg, log)
	if err != nil {
		log.Fatal("Failed to create server", logger.Error(err))
	}

	shutdownMgr := setupShutdown(srv, proxyManager, log, cfg)

	serverErrors := make(chan error, 1)
	go func() {
		log.Info("API Gateway is running",
			logger.String("address", srv.Address()),
			logger.Bool("tls", cfg.Server.TLSEnabled),
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
		log.Info("API Gateway stopped gracefully")
	}
}
