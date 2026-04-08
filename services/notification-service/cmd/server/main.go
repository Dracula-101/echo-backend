package main

import (
	"fmt"
	"notification-service/api/v1/handler"

	"shared/pkg/logger"
	adapter "shared/pkg/logger/adapter"
	prommetrics "shared/pkg/monitoring/metrics/prometheus"
	env "shared/server/env"
	coreMiddleware "shared/server/middleware"
	req "shared/server/request"
	"shared/server/response"
	"shared/server/router"
	"shared/server/server"
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

func setupRoutes(builder *router.Builder, h *handler.NotificationHandler, log logger.Logger) *router.Builder {
	log.Debug("Registering notification routes")
	builder = builder.WithRoutes(func(r *router.Router) {
		// Notification endpoints
		r.Get("/notifications", req.Adapt(h.ListNotifications))
		r.Get("/notifications/{id}", req.Adapt(h.GetNotification))
		r.Post("/notifications/mark-read", req.Adapt(h.MarkAsRead))
		r.Post("/notifications/mark-all-read", req.Adapt(h.MarkAllAsRead))
		r.Delete("/notifications/{id}", req.Adapt(h.DeleteNotification))

		// Preference endpoints
		r.Get("/preferences", req.Adapt(h.GetPreferences))
		r.Put("/preferences", req.Adapt(h.UpdatePreferences))

		// Internal endpoints
		r.Post("/send", req.Adapt(h.SendNotification))
		r.Post("/send-bulk", req.Adapt(h.SendBulkNotifications))
	})
	log.Debug("Notification routes registered successfully")
	return builder
}

func createRouter(h *handler.NotificationHandler, log logger.Logger) (*router.Router, error) {
	builder := router.NewBuilder().
		WithHealthEndpoint("/health", func(rh req.RequestHandler) {
			response.JSONWithMessage(rh.Context(), rh.Request(), rh.Writer(), 200, "OK", nil)
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

func main() {
	env.LoadEnv()

	log := createLogger("notification-service")
	defer log.Sync()

	h := handler.NewNotificationHandler(log)

	routerInstance, err := createRouter(h, log)
	if err != nil {
		log.Fatal("Failed to create router", logger.Error(err))
	}

	srv, err := server.New(&server.Config{
		Host:    "0.0.0.0",
		Port:    8080,
		Handler: routerInstance.Mux(),
	}, log)
	if err != nil {
		log.Fatal("Failed to create server", logger.Error(err))
	}

	log.Info("Starting Notification Service", logger.Int("port", 8080))
	if err := srv.Start(); err != nil && !server.IsServerDownErr(err) {
		log.Fatal("Server error", logger.Error(err))
	}
}
