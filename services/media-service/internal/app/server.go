package app

import (
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

	"media-service/api/v1/handler"
	"media-service/api/v1/middleware"
	"media-service/internal/config"
	"media-service/internal/health"
	"media-service/internal/health/checkers"
)

func buildHTTPServer(cfg *config.Config, h *handler.Handler, db database.Database, cacheClient cache.Cache, log logger.Logger) (*server.Server, error) {
	healthMgr := buildHealthManager(db, cacheClient, cfg)
	healthHandler := health.NewHandler(healthMgr)

	r := buildRouter(h, healthHandler, cfg, log)

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

func buildRouter(h *handler.Handler, healthHandler *health.Handler, cfg *config.Config, log logger.Logger) *router.Router {
	return router.NewBuilder().
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
			router.Middleware(coreMiddleware.InterceptRequestID(headers.XRequestID)),
			router.Middleware(coreMiddleware.InterceptCorrelationID(headers.XCorrelationID)),
			router.Middleware(coreMiddleware.RequestReceivedLogger(log)),
			router.Middleware(coreMiddleware.InterceptUserId()),
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
		}).
		WithRoutes(func(r *router.Router) {
			r.Post("/upload", coreMiddleware.ApplyMiddlewares(
				request.Adapt(h.Upload),
				middleware.FileOnlyMultipart(log, cfg.Security.MaxBodySize, cfg.Features.ImageProcessing.AllowedFormats),
			))
			r.Post("/profile-photo", coreMiddleware.ApplyMiddlewares(
				request.Adapt(h.UploadProfilePhoto),
				middleware.FileOnlyMultipart(log, cfg.Security.MaxBodySize, cfg.Features.ImageProcessing.AllowedFormats),
			))
			r.Post("/message-media", coreMiddleware.ApplyMiddlewares(
				request.Adapt(h.UploadMessageMedia),
				middleware.FileOnlyMultipart(log, cfg.Security.MaxBodySize, cfg.Storage.AllowedTypes),
			))

			r.Get("/files/{file_id}", request.Adapt(h.GetFile))
			r.Put("/files/{file_id}", request.Adapt(h.UpdateFile))
			r.Delete("/files/{file_id}", request.Adapt(h.DeleteFile))
			r.Get("/files", request.Adapt(h.ListFiles))

			r.Post("/albums", request.Adapt(h.CreateAlbum))
			r.Get("/albums/{id}", request.Adapt(h.GetAlbum))
			r.Get("/albums", request.Adapt(h.ListAlbums))
			r.Put("/albums/{id}", request.Adapt(h.UpdateAlbum))
			r.Delete("/albums/{id}", request.Adapt(h.DeleteAlbum))
			r.Post("/albums/{id}/files", request.Adapt(h.AddFileToAlbum))
			r.Delete("/albums/{id}/files/{file_id}", request.Adapt(h.RemoveFileFromAlbum))

			r.Post("/shares", request.Adapt(h.CreateShare))
			r.Get("/shares/{id}", request.Adapt(h.GetShare))
			r.Get("/shares", request.Adapt(h.ListShares))
			r.Delete("/shares/{id}", request.Adapt(h.RevokeShare))

			r.Get("/stats", request.Adapt(h.GetStorageStats))
		}).
		Build()
}
