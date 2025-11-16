package router

import (
	"net/http"
	"os"
	"reflect"
	"runtime"

	"shared/pkg/logger"
	"shared/pkg/logger/adapter"
	"shared/server/middleware"

	"github.com/gorilla/mux"
)

type Middleware func(http.Handler) http.Handler

type Handler func(http.ResponseWriter, *http.Request)

type Builder struct {
	router             *Router
	earlyMiddleware    []Middleware
	lateMiddleware     []Middleware
	systemEndpoints    []Endpoint
	routes             []func(*Router)
	routeGroups        []routeGroupRegistration
	notFoundHandler    Handler
	notAllowedHandler  Handler
	enableSystemRoutes bool
	logger             logger.Logger
}

type routeGroupRegistration struct {
	prefix    string
	registrar func(*RouteGroup)
}

func NewBuilder() *Builder {
	log, err := adapter.NewZap(logger.Config{
		Level:      logger.GetLoggerLevel(),
		Output:     os.Stdout,
		Format:     logger.GetLoggerFormat(),
		TimeFormat: logger.GetLoggerTimeFormat(),
		Service:    "router-builder",
	})
	if err != nil {
		panic("failed to create logger for router builder: " + err.Error())
	}
	log.Debug("Router builder logger initialized")
	return &Builder{
		router:             New(),
		earlyMiddleware:    make([]Middleware, 0),
		lateMiddleware:     make([]Middleware, 0),
		systemEndpoints:    make([]Endpoint, 0),
		routes:             make([]func(*Router), 0),
		routeGroups:        make([]routeGroupRegistration, 0),
		enableSystemRoutes: true,
		logger:             log,
	}
}

func (b *Builder) WithEarlyMiddleware(middleware ...Middleware) *Builder {
	b.earlyMiddleware = append(b.earlyMiddleware, middleware...)
	return b
}

func (b *Builder) WithLateMiddleware(middleware ...Middleware) *Builder {
	b.lateMiddleware = append(b.lateMiddleware, middleware...)
	return b
}

func (b *Builder) WithMiddleware(middleware ...Middleware) *Builder {
	return b.WithLateMiddleware(middleware...)
}

func (b *Builder) WithMiddlewareChain(chain *middleware.Chain) *Builder {
	for _, mw := range chain.Middleware() {
		b.lateMiddleware = append(b.lateMiddleware, Middleware(mw))
	}
	return b
}

func getFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func (b *Builder) WithHealthEndpoint(path string, handler Handler) *Builder {
	b.systemEndpoints = append(b.systemEndpoints, Endpoint{
		Path:    path,
		Handler: http.HandlerFunc(handler),
		Method:  http.MethodGet,
	})
	b.logger.Debug("Health endpoint queued", logger.String("path", path))
	return b
}

func (b *Builder) WithMetricsEndpoint(path string, handler Handler) *Builder {
	b.systemEndpoints = append(b.systemEndpoints, Endpoint{
		Path:    path,
		Handler: http.HandlerFunc(handler),
		Method:  http.MethodGet,
	})
	b.logger.Debug("Metrics endpoint queued", logger.String("path", path))
	return b
}

func (b *Builder) WithVersionEndpoint(path string, handler Handler) *Builder {
	b.systemEndpoints = append(b.systemEndpoints, Endpoint{
		Path:    path,
		Handler: http.HandlerFunc(handler),
		Method:  http.MethodGet,
	})
	b.logger.Debug("Version endpoint queued", logger.String("path", path))
	return b
}

func (b *Builder) WithStatusEndpoint(path string, handler Handler) *Builder {
	b.systemEndpoints = append(b.systemEndpoints, Endpoint{
		Path:    path,
		Handler: http.HandlerFunc(handler),
		Method:  http.MethodGet,
	})
	b.logger.Debug("Status endpoint queued", logger.String("path", path))
	return b
}

func (b *Builder) WithNotFoundHandler(handler Handler) *Builder {
	b.notFoundHandler = handler
	b.logger.Debug("Not Found handler queued")
	return b
}

func (b *Builder) WithMethodNotAllowedHandler(handler Handler) *Builder {
	b.notAllowedHandler = handler
	b.logger.Debug("Method Not Allowed handler queued")
	return b
}

func (b *Builder) DisableSystemRoutes() *Builder {
	b.enableSystemRoutes = false
	return b
}

func (b *Builder) WithRoutes(registrar func(*Router)) *Builder {
	b.routes = append(b.routes, registrar)
	b.logger.Debug("Routes queued for registration")
	return b
}

func (b *Builder) WithRoutesGroup(prefix string, registrar func(*RouteGroup)) *Builder {
	b.routeGroups = append(b.routeGroups, routeGroupRegistration{
		prefix:    prefix,
		registrar: registrar,
	})
	b.logger.Debug("Route group queued", logger.String("prefix", prefix))
	return b
}

func (b *Builder) Build() *Router {
	b.logger.Debug("Building router - registering routes in priority order")

	if b.enableSystemRoutes {
		for _, endpoint := range b.systemEndpoints {
			b.router.Handle(endpoint.Path, endpoint.Method, endpoint.Handler)
			b.logger.Debug("System endpoint registered",
				logger.String("path", endpoint.Path),
				logger.String("method", endpoint.Method),
			)
		}
	}

	appMux := mux.NewRouter().StrictSlash(true)
	appRouter := &Router{
		mux:            appMux,
		routes:         make([]RouteInfo, 0),
		strictPriority: b.router.strictPriority,
	}

	for _, routeRegistrar := range b.routes {
		routeRegistrar(appRouter)
		b.logger.Debug("Routes registered on app router")
	}

	for _, rg := range b.routeGroups {
		group := appRouter.Group(rg.prefix)
		rg.registrar(group)
		b.logger.Debug("Route group registered", logger.String("prefix", rg.prefix))
	}

	for _, mw := range b.earlyMiddleware {
		appRouter.Use(mux.MiddlewareFunc(mw))
		b.logger.Debug("Applied early middleware to app router", logger.String("name", getFunctionName(mw)))
	}

	for _, mw := range b.lateMiddleware {
		appRouter.Use(mux.MiddlewareFunc(mw))
		b.logger.Debug("Applied late middleware to app router", logger.String("name", getFunctionName(mw)))
	}

	b.router.Mux().PathPrefix("/").Handler(appMux)

	if b.notFoundHandler != nil {
		b.router.Mux().NotFoundHandler = http.HandlerFunc(b.notFoundHandler)
		b.logger.Debug("Not Found handler registered")
	}

	if b.notAllowedHandler != nil {
		b.router.Mux().MethodNotAllowedHandler = http.HandlerFunc(b.notAllowedHandler)
		b.logger.Debug("Method Not Allowed handler registered")
	}

	b.logger.Info("Router built successfully",
		logger.Int("system_endpoints", len(b.systemEndpoints)),
		logger.Int("route_groups", len(b.routeGroups)),
		logger.Int("early_middleware", len(b.earlyMiddleware)),
		logger.Int("late_middleware", len(b.lateMiddleware)),
	)

	return b.router
}

type SystemEndpointsConfig struct {
	HealthPath       string
	LivenessPath     string
	ReadinessPath    string
	MetricsPath      string
	VersionPath      string
	StatusPath       string
	HealthHandler    Handler
	LivenessHandler  Handler
	ReadinessHandler Handler
	MetricsHandler   Handler
	VersionHandler   Handler
	StatusHandler    Handler
}

func DefaultSystemEndpointsConfig() *SystemEndpointsConfig {
	return &SystemEndpointsConfig{
		HealthPath:    "/health",
		LivenessPath:  "/health/live",
		ReadinessPath: "/health/ready",
		MetricsPath:   "/metrics",
		VersionPath:   "/version",
		StatusPath:    "/status",
	}
}
