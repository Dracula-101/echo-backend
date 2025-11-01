package router

import (
	"net/http"
	"shared/server/middleware"

	"github.com/gorilla/mux"
)

type Router struct {
	mux    *mux.Router
	routes []Route
}

type Route struct {
	Name        string
	Method      string
	Pattern     string
	Handler     http.HandlerFunc
	Middlewares []mux.MiddlewareFunc
}

type Endpoint struct {
	Path    string
	Handler http.HandlerFunc
	Method  string
}

func (r *Router) Mux() *mux.Router {
	return r.mux
}

func (r *Router) Routes() []Route {
	return r.routes
}

func (r *Router) Handle(method, pattern string, handler http.HandlerFunc) *mux.Route {
	route := r.mux.HandleFunc(pattern, handler).Methods(method)
	r.routes = append(r.routes, Route{
		Method:  method,
		Pattern: pattern,
		Handler: handler,
	})
	return route
}

func (r *Router) HandleFunc(pattern string, handler http.HandlerFunc) *mux.Route {
	route := r.mux.HandleFunc(pattern, handler)
	r.routes = append(r.routes, Route{
		Pattern: pattern,
		Handler: handler,
	})
	return route
}

func (r *Router) HandlePrefix(prefix string, handler http.HandlerFunc) *mux.Route {
	route := r.mux.PathPrefix(prefix).HandlerFunc(handler)
	r.routes = append(r.routes, Route{
		Pattern: prefix + "*",
		Handler: handler,
	})
	return route
}

func (r *Router) RegisterRoute(route Route) *mux.Route {
	r.routes = append(r.routes, route)
	return r.mux.HandleFunc(route.Pattern, route.Handler).Methods(route.Method)
}

func (r *Router) Get(pattern string, handler http.HandlerFunc) *mux.Route {
	return r.Handle(http.MethodGet, pattern, handler)
}

func (r *Router) Post(pattern string, handler http.HandlerFunc) *mux.Route {
	return r.Handle(http.MethodPost, pattern, handler)
}

func (r *Router) Put(pattern string, handler http.HandlerFunc) *mux.Route {
	return r.Handle(http.MethodPut, pattern, handler)
}

func (r *Router) Delete(pattern string, handler http.HandlerFunc) *mux.Route {
	return r.Handle(http.MethodDelete, pattern, handler)
}

func (r *Router) Patch(pattern string, handler http.HandlerFunc) *mux.Route {
	return r.Handle(http.MethodPatch, pattern, handler)
}

func (r *Router) Options(pattern string, handler http.HandlerFunc) *mux.Route {
	return r.Handle(http.MethodOptions, pattern, handler)
}

func (r *Router) PathPrefix(prefix string) *mux.Route {
	return r.mux.PathPrefix(prefix)
}

func (r *Router) Use(middleware ...mux.MiddlewareFunc) {
	r.mux.Use(middleware...)
}

func (r *Router) UseChain(chain *middleware.Chain) {
	r.mux.Use(chain.Middleware()...)
}

func (r *Router) Group(prefix string, middlewares ...mux.MiddlewareFunc) *RouteGroup {
	subrouter := r.mux.PathPrefix(prefix).Subrouter()
	subrouter.Use(middlewares...)
	return &RouteGroup{
		prefix: prefix,
		router: subrouter,
		parent: r,
	}
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

type RouteGroup struct {
	prefix string
	router *mux.Router
	parent *Router
}

func (g *RouteGroup) HandleProxy(handler http.HandlerFunc, methods ...string) *mux.Route {
	route := g.router.HandleFunc("/{rest:.*}", handler).Methods(methods...)
	return route
}

func (g *RouteGroup) Handle(method, pattern string, handler http.HandlerFunc) *mux.Route {
	route := g.router.HandleFunc(pattern, handler).Methods(method)
	return route
}

func (g *RouteGroup) Get(pattern string, handler http.HandlerFunc) *mux.Route {
	return g.Handle(http.MethodGet, pattern, handler)
}

func (g *RouteGroup) Post(pattern string, handler http.HandlerFunc) *mux.Route {
	return g.Handle(http.MethodPost, pattern, handler)
}

func (g *RouteGroup) Put(pattern string, handler http.HandlerFunc) *mux.Route {
	return g.Handle(http.MethodPut, pattern, handler)
}

func (g *RouteGroup) Delete(pattern string, handler http.HandlerFunc) *mux.Route {
	return g.Handle(http.MethodDelete, pattern, handler)
}

func (g *RouteGroup) Patch(pattern string, handler http.HandlerFunc) *mux.Route {
	return g.Handle(http.MethodPatch, pattern, handler)
}

func (g *RouteGroup) UseChain(middleware middleware.Chain) {
	g.router.Use(middleware.Middleware()...)
}

func (g *RouteGroup) Group(prefix string, middlewares ...mux.MiddlewareFunc) *RouteGroup {
	subrouter := g.router.PathPrefix(prefix).Subrouter()
	subrouter.Use(middlewares...)
	return &RouteGroup{
		prefix: g.prefix + prefix,
		router: subrouter,
		parent: g.parent,
	}
}
