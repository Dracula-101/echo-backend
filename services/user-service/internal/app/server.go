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
	"shared/server/request"
	"shared/server/response"
	"shared/server/router"
	"shared/server/server"

	"user-service/api/v1/handler"
	"user-service/api/v1/middleware"
	ratelimiter "user-service/api/v1/rate_limiter"
	"user-service/internal/config"
	"user-service/internal/health"
	"user-service/internal/health/checkers"
	"user-service/internal/service/location"
)

func buildHTTPServer(
	cfg *config.Config,
	h *handler.UserHandler,
	locationService location.LocationService,
	memCache cache.Cache,
	rl ratelimiter.RateLimiter,
	db database.Database,
	cacheClient cache.Cache,
	log logger.Logger,
) (*server.Server, error) {
	healthMgr := buildHealthManager(db, cacheClient, cfg)
	healthHandler := health.NewHandler(healthMgr)

	r := buildRouter(h, locationService, memCache, rl, healthHandler, log)

	return server.New(&server.Config{
		Host:           cfg.Server.Host,
		Port:           cfg.Server.Port,
		ReadTimeout:    cfg.Server.ReadTimeout,
		WriteTimeout:   cfg.Server.WriteTimeout,
		IdleTimeout:    cfg.Server.IdleTimeout,
		MaxHeaderBytes: cfg.Server.MaxHeaderBytes,
		Handler:        r.Mux(),
	}, log)
}

func buildHealthManager(db database.Database, cacheClient cache.Cache, cfg *config.Config) *health.Manager {
	mgr := health.NewManager(cfg.Service.Name, cfg.Service.Version)
	if db != nil {
		mgr.RegisterChecker(checkers.NewDatabaseChecker(db))
	}
	if cacheClient != nil && cfg.Cache.Enabled {
		mgr.RegisterChecker(checkers.NewCacheChecker(cacheClient))
		mgr.RegisterChecker(checkers.NewCachePerformanceChecker(cacheClient))
	}
	return mgr
}

func buildRouter(
	h *handler.UserHandler,
	locationService location.LocationService,
	memCache cache.Cache,
	rl ratelimiter.RateLimiter,
	healthHandler *health.Handler,
	log logger.Logger,
) *router.Router {
	skipPaths := []string{"/username-exists", "/health", "/metrics", "/live", "/ready", "/health/liveness", "/health/readiness"}

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

	return registerUserRoutes(builder, h, rl).Build()
}

// registerUserRoutes attaches every user-facing endpoint. Split out from
// buildRouter so the middleware/health setup stays readable.
func registerUserRoutes(builder *router.Builder, h *handler.UserHandler, rl ratelimiter.RateLimiter) *router.Builder {
	userIDFromCtx := func(req request.RequestHandler) string {
		uid, _ := request.GetUserIDFromContext(req.Context())
		return uid
	}

	builder = builder.WithRoutes(func(r *router.Router) {
		r.Get("/username-exists", request.Adapt(h.UsernameExists))
	})

	builder = builder.WithRoutesGroup("/me", func(rg *router.RouteGroup) {
		rg.Get("", request.Adapt(h.GetMyProfile))
		rg.Post("", request.Adapt(h.CreateMyProfile))
		rg.Put(
			"",
			request.Adapt(h.UpdateMyProfile),
			middleware.RateLimit(
				func(ctx context.Context, key string, limit int64) (bool, int64, error) {
					return rl.AllowProfileUpdate(ctx, key, limit)
				},
				userIDFromCtx,
				10,
			),
		)
		rg.Post("/block", request.Adapt(h.BlockUser))
		rg.Delete("/block/{user_id}", request.Adapt(h.UnblockUser))
	})

	builder = builder.WithRoutes(func(r *router.Router) {
		r.Get("/@{username}", request.Adapt(h.GetProfile))
		r.Get(
			"/search",
			request.Adapt(h.SearchProfiles),
			middleware.RateLimit(
				func(ctx context.Context, key string, limit int64) (bool, int64, error) {
					return rl.AllowSearch(ctx, key, limit)
				},
				userIDFromCtx,
				30,
			),
		)
	})

	builder = builder.WithRoutesGroup("/me/contacts", func(rg *router.RouteGroup) {
		rg.Get("", request.Adapt(h.ListContacts))
		rg.Post(
			"",
			request.Adapt(h.AddContact),
			middleware.RateLimit(
				func(ctx context.Context, key string, limit int64) (bool, int64, error) {
					return rl.AllowContactAdd(ctx, key, limit)
				},
				userIDFromCtx,
				20,
			),
		)
		rg.Put("/{contact_id}", request.Adapt(h.UpdateContact))
		rg.Delete("/{contact_id}", request.Adapt(h.RemoveContact))
		rg.Post("/{contact_id}/accept", request.Adapt(h.AcceptContact))
		rg.Post("/{contact_id}/reject", request.Adapt(h.RejectContact))
	})

	builder = builder.WithRoutes(func(r *router.Router) {
		r.Get("/contacts/groups", request.Adapt(h.ListContactGroups))
		r.Post("/contacts/groups", request.Adapt(h.CreateContactGroup))
		r.Put("/contacts/groups/{group_id}", request.Adapt(h.UpdateContactGroup))
		r.Delete("/contacts/groups/{group_id}", request.Adapt(h.DeleteContactGroup))

		r.Get("/settings", request.Adapt(h.GetSettings))
		r.Put("/settings", request.Adapt(h.UpdateSettings))

		r.Get("/blocked", request.Adapt(h.ListBlocked))
		r.Post(
			"/blocked",
			request.Adapt(h.BlockUser),
			middleware.RateLimit(
				func(ctx context.Context, key string, limit int64) (bool, int64, error) {
					return rl.AllowBlockAction(ctx, key, limit)
				},
				userIDFromCtx,
				10,
			),
		)

		r.Post("/status", request.Adapt(h.SetStatus))
		r.Get("/status/{user_id}", request.Adapt(h.GetStatus))
		r.Delete("/status/{status_id}", request.Adapt(h.DeleteStatus))
		r.Get("/status/{status_id}/views", request.Adapt(h.GetStatusViews))

		r.Get("/devices", request.Adapt(h.ListDevices))
		r.Delete("/devices/{device_id}", request.Adapt(h.RemoveDevice))
		r.Put("/devices/{device_id}/push-token", request.Adapt(h.UpdateDevice))

		r.Get("/preferences", request.Adapt(h.GetPreferences))
		r.Put("/preferences", request.Adapt(h.UpdatePreferences))

		r.Get("/achievements", request.Adapt(h.ListAchievements))
		r.Post("/reports", request.Adapt(h.SubmitReport))
	})

	return builder
}
