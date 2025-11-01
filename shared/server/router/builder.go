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
	notFoundHandler    Handler
	notAllowedHandler  Handler
	enableSystemRoutes bool
	logger             logger.Logger
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
		router:             &Router{mux: mux.NewRouter()},
		earlyMiddleware:    make([]Middleware, 0),
		lateMiddleware:     make([]Middleware, 0),
		systemEndpoints:    make([]Endpoint, 0),
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
	b.logger.Debug("Health endpoint registered", logger.String("path", path))
	return b
}

func (b *Builder) WithMetricsEndpoint(path string, handler Handler) *Builder {
	b.systemEndpoints = append(b.systemEndpoints, Endpoint{
		Path:    path,
		Handler: http.HandlerFunc(handler),
		Method:  http.MethodGet,
	})
	b.logger.Debug("Metrics endpoint registered", logger.String("path", path))
	return b
}

func (b *Builder) WithVersionEndpoint(path string, handler Handler) *Builder {
	b.systemEndpoints = append(b.systemEndpoints, Endpoint{
		Path:    path,
		Handler: http.HandlerFunc(handler),
		Method:  http.MethodGet,
	})
	b.logger.Debug("Version endpoint registered", logger.String("path", path))
	return b
}

func (b *Builder) WithStatusEndpoint(path string, handler Handler) *Builder {
	b.systemEndpoints = append(b.systemEndpoints, Endpoint{
		Path:    path,
		Handler: http.HandlerFunc(handler),
		Method:  http.MethodGet,
	})
	b.logger.Debug("Status endpoint registered", logger.String("path", path))
	return b
}

func (b *Builder) WithNotFoundHandler(handler Handler) *Builder {
	b.notFoundHandler = handler
	b.logger.Debug("Not Found handler added")
	return b
}

func (b *Builder) WithMethodNotAllowedHandler(handler Handler) *Builder {
	b.notAllowedHandler = handler
	b.logger.Debug("Method Not Allowed handler added")
	return b
}

func (b *Builder) DisableSystemRoutes() *Builder {
	b.enableSystemRoutes = false
	return b
}

func (b *Builder) Build() *Router {
	b.logger.Debug("Building router with middleware chain")
	for _, mw := range b.earlyMiddleware {
		b.router.Use(mux.MiddlewareFunc(mw))
		b.logger.Debug("Applied early middleware", logger.String("middleware", getFunctionName(mw)))
	}

	for _, mw := range b.lateMiddleware {
		b.router.Use(mux.MiddlewareFunc(mw))
		b.logger.Debug("Applied late middleware", logger.String("middleware", getFunctionName(mw)))
	}

	if b.enableSystemRoutes {
		for _, endpoint := range b.systemEndpoints {
			b.router.Handle(endpoint.Path, endpoint.Method, endpoint.Handler)
			b.logger.Debug("System endpoint registered",
				logger.String("path", endpoint.Path),
				logger.String("method", endpoint.Method),
				logger.String("handler", getFunctionName(endpoint.Handler)),
			)
		}
	}

	if b.notFoundHandler != nil {
		b.logger.Debug("Not Found handler registered with middleware chain")
		b.router.Mux().NotFoundHandler = http.HandlerFunc(b.notFoundHandler)
	} else {
		b.logger.Warn("No Not Found handler registered")
	}

	if b.notAllowedHandler != nil {
		b.logger.Debug("Method Not Allowed handler registered with middleware chain")
		b.router.Mux().MethodNotAllowedHandler = http.HandlerFunc(b.notAllowedHandler)
	} else {
		b.logger.Warn("No Method Not Allowed handler registered")
	}

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

type RouteRegistrar func(*Router)

func (b *Builder) WithRoutes(registrar RouteRegistrar) *Builder {
	registrar(b.router)
	return b
}

func (b *Builder) WithRoutesGroup(prefix string, registrar func(*RouteGroup)) *Builder {
	group := b.router.Group(prefix)
	registrar(group)
	return b
}
