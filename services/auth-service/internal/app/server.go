package app

import (
	"context"
	"fmt"

	"shared/pkg/cache"
	"shared/pkg/database"
	"shared/pkg/logger"
	prommetrics "shared/pkg/monitoring/metrics/prometheus"
	"shared/server/headers"
	coreMiddleware "shared/server/middleware"
	req "shared/server/request"
	"shared/server/response"
	"shared/server/router"
	"shared/server/server"

	"auth-service/api/v1/handler"
	"auth-service/api/v1/middleware"
	ratelimiter "auth-service/api/v1/rate_limiter"
	"auth-service/internal/config"
	"auth-service/internal/health"
	"auth-service/internal/health/checkers"
	"auth-service/internal/service/location"
	sessionSvc "auth-service/internal/service/session"
)

func buildHTTPServer(
	cfg *config.Config,
	h *handler.AuthHandler,
	memCache cache.Cache,
	locationService location.LocationService,
	sessionService sessionSvc.SessionService,
	sessionCache sessionSvc.SessionCache,
	rl ratelimiter.RateLimiter,
	db database.Database,
	cacheClient cache.Cache,
	log logger.Logger,
) (*server.Server, error) {
	healthMgr := buildHealthManager(db, cacheClient, cfg)
	healthHandler := health.NewHandler(healthMgr)

	r := buildRouter(h, memCache, locationService, sessionService, sessionCache, rl, healthHandler, log)

	return server.New(&server.Config{
		Host:            cfg.Server.Host,
		Port:            cfg.Server.Port,
		ReadTimeout:     cfg.Server.ReadTimeout,
		WriteTimeout:    cfg.Server.WriteTimeout,
		IdleTimeout:     cfg.Server.IdleTimeout,
		MaxHeaderBytes:  cfg.Server.MaxHeaderBytes,
		Handler:         r.Mux(),
		ShutdownTimeout: cfg.Server.ShutdownTimeout,
	}, log)
}

func buildHealthManager(db database.Database, cacheClient cache.Cache, cfg *config.Config) *health.Manager {
	mgr := health.NewManager(cfg.Service.Name, cfg.Service.Version)
	if db != nil {
		mgr.RegisterChecker(checkers.NewDatabaseChecker(db))
	}
	if cacheClient != nil && cfg.Cache.Enabled {
		mgr.RegisterChecker(checkers.NewCacheChecker(cacheClient))
	}
	return mgr
}

func buildRouter(
	h *handler.AuthHandler,
	memCache cache.Cache,
	locationService location.LocationService,
	sessionService sessionSvc.SessionService,
	sessionCache sessionSvc.SessionCache,
	rl ratelimiter.RateLimiter,
	healthHandler *health.Handler,
	log logger.Logger,
) *router.Router {
	skipAuthPaths := []string{
		"/login", "/register", "/verify-email", "/refresh-token",
		"/forgot-password", "/reset-password", "/resend-verification",
		"/health", "/live", "/ready", "/health/liveness", "/health/readiness", "/metrics",
	}

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
			router.Middleware(middleware.SessionAuth(sessionService, sessionCache, log, skipAuthPaths...)),
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
		).
		WithRoutes(func(r *router.Router) {
			r.Get("/live", healthHandler.Liveness)
			r.Get("/ready", healthHandler.Readiness)
			r.Get("/health/liveness", healthHandler.Liveness)
			r.Get("/health/readiness", healthHandler.Readiness)
		})

	return registerAuthRoutes(builder, h, rl).Build()
}

// registerAuthRoutes attaches every public auth endpoint with its
// per-route rate-limit middleware.
func registerAuthRoutes(builder *router.Builder, h *handler.AuthHandler, rl ratelimiter.RateLimiter) *router.Builder {
	clientIP := func(r req.RequestHandler) string { return r.GetClientIP() }
	deviceID := func(r req.RequestHandler) string {
		id, _ := r.GetDeviceIDFromHeader()
		return id
	}

	return builder.WithRoutes(func(r *router.Router) {
		r.Post("/register", req.Adapt(h.Register), middleware.RateLimit(
			func(ctx context.Context, key string, limit int64) (bool, int64, error) {
				return rl.AllowRegisterIP(ctx, key, limit)
			}, clientIP, 5,
		))
		r.Post("/login", req.Adapt(h.Login), middleware.RateLimit(
			func(ctx context.Context, key string, limit int64) (bool, int64, error) {
				return rl.AllowLoginIP(ctx, key, limit)
			}, clientIP, 10,
		))
		r.Post("/refresh-token", req.Adapt(h.RefreshToken), middleware.RateLimit(
			func(ctx context.Context, key string, limit int64) (bool, int64, error) {
				if key == "" {
					return true, 0, nil
				}
				return rl.AllowRefreshDevice(ctx, key, limit)
			}, deviceID, 10,
		))
		r.Post("/logout", req.Adapt(h.Logout))
		r.Post("/forgot-password", req.Adapt(h.ForgotPassword), middleware.RateLimitMulti(
			rl,
			middleware.RateLimitCheck{
				Allow: func(ctx context.Context, key string, limit int64) (bool, int64, error) {
					return rl.AllowPwdForgotIP(ctx, key, limit)
				},
				Key:   clientIP,
				Limit: 5,
			},
			middleware.RateLimitCheck{
				Allow: func(ctx context.Context, key string, limit int64) (bool, int64, error) {
					if key == "" {
						return true, 0, nil
					}
					return rl.AllowPwdForgotEmail(ctx, key, limit)
				},
				Key: func(r req.RequestHandler) string {
					ok, email := r.GetBodyValue("email")
					if !ok {
						return ""
					}
					return email
				},
				Limit: 5,
			},
		))
		r.Post("/reset-password", req.Adapt(h.ResetPassword), middleware.RateLimit(
			func(ctx context.Context, key string, limit int64) (bool, int64, error) {
				return rl.AllowPwdResetIP(ctx, key, limit)
			}, clientIP, 10,
		))
		r.Post("/change-password", req.Adapt(h.ChangePassword))
		r.Post("/verify-email", req.Adapt(h.VerifyEmail))
		r.Post("/resend-verification", req.Adapt(h.ResendVerification), middleware.RateLimit(
			func(ctx context.Context, key string, limit int64) (bool, int64, error) {
				return rl.AllowResendVerify(ctx, key, limit)
			}, clientIP, 5,
		))
		r.Post("/send-otp", req.Adapt(h.SendOTP), middleware.RateLimit(
			func(ctx context.Context, key string, limit int64) (bool, int64, error) {
				return rl.AllowOtpSend(ctx, key, limit)
			}, clientIP, 10,
		))
		r.Post("/verify-phone", req.Adapt(h.VerifyPhone), middleware.RateLimit(
			func(ctx context.Context, key string, limit int64) (bool, int64, error) {
				return rl.AllowOtpSend(ctx, key, limit)
			}, clientIP, 10,
		))

		r.Get("/sessions", req.Adapt(h.GetSessions))
		r.Delete("/sessions/{id}", req.Adapt(h.DeleteSession))

		r.Post("/mfa/enable", req.Adapt(h.MFAEnable))
		r.Post("/mfa/disable", req.Adapt(h.MFADisable))
		r.Post("/mfa/verify", req.Adapt(h.MFAVerify))

		r.Delete("/account", req.Adapt(h.DeleteAccount))
	})
}
