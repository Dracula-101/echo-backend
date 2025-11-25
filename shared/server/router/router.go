package router

import (
	"net/http"
	"shared/server/middleware"
	"strings"

	"github.com/gorilla/mux"
)

type Router struct {
	mux            *mux.Router
	routes         []RouteInfo
	strictPriority bool
}

type RouteInfo struct {
	Name    string
	Method  string
	Pattern string
	Handler http.HandlerFunc
	Type    RouteType
}

type RouteType string

const (
	RouteTypeExact  RouteType = "exact"
	RouteTypePrefix RouteType = "prefix"
	RouteTypeCatch  RouteType = "catchall"
)

type Endpoint struct {
	Path    string
	Handler http.Handler
	Method  string
}

func New() *Router {
	return &Router{
		mux:            mux.NewRouter().StrictSlash(true),
		routes:         make([]RouteInfo, 0),
		strictPriority: true,
	}
}

func (r *Router) Mux() *mux.Router {
	return r.mux
}

func (r *Router) Routes() []RouteInfo {
	return r.routes
}

func (r *Router) StrictPriority(enabled bool) {
	r.strictPriority = enabled
}

func (r *Router) RegisterExact(method, path string, handler http.Handler) *mux.Route {
	route := r.mux.NewRoute().Path(path).Methods(method).Handler(handler)
	r.routes = append(r.routes, RouteInfo{
		Method:  method,
		Pattern: path,
		Type:    RouteTypeExact,
	})
	return route
}

func (r *Router) RegisterFuncExact(method, path string, handler http.HandlerFunc) *mux.Route {
	route := r.mux.NewRoute().Path(path).Methods(method).HandlerFunc(handler)
	r.routes = append(r.routes, RouteInfo{
		Method:  method,
		Pattern: path,
		Type:    RouteTypeExact,
	})
	return route
}

func (r *Router) With(middlewares ...middleware.Handler) *Router {
	for _, m := range middlewares {
		r.mux.Use(func(h http.Handler) http.Handler {
			return m(h)
		})
	}
	return r
}

func (r *Router) Handle(path string, method string, handler http.Handler) *mux.Route {
	return r.RegisterExact(method, path, handler)
}

func (r *Router) HandleFunc(path string, method string, handler http.HandlerFunc) *mux.Route {
	return r.RegisterFuncExact(method, path, handler)
}

func (r *Router) Get(path string, handler http.HandlerFunc) *mux.Route {
	return r.RegisterExact(http.MethodGet, path, handler)
}

func (r *Router) Post(path string, handler http.HandlerFunc) *mux.Route {
	return r.RegisterExact(http.MethodPost, path, handler)
}

func (r *Router) Put(path string, handler http.HandlerFunc) *mux.Route {
	return r.RegisterExact(http.MethodPut, path, handler)
}

func (r *Router) Delete(path string, handler http.HandlerFunc) *mux.Route {
	return r.RegisterExact(http.MethodDelete, path, handler)
}

func (r *Router) Patch(path string, handler http.HandlerFunc) *mux.Route {
	return r.RegisterExact(http.MethodPatch, path, handler)
}

func (r *Router) Options(path string, handler http.HandlerFunc) *mux.Route {
	return r.RegisterExact(http.MethodOptions, path, handler)
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
	route := g.router.PathPrefix("/").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, g.prefix)
		if path == "" {
			path = "/"
		}
		vars := map[string]string{
			"rest": strings.TrimPrefix(path, "/"),
			"path": strings.TrimPrefix(path, "/"),
		}
		r = mux.SetURLVars(r, vars)
		handler(w, r)
	})).Methods(methods...)

	// Also register on parent router to handle exact prefix without trailing slash
	g.parent.mux.Path(g.prefix).Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := map[string]string{
			"rest": "",
			"path": "",
		}
		r = mux.SetURLVars(r, vars)
		handler(w, r)
	})).Methods(methods...)

	g.parent.routes = append(g.parent.routes, RouteInfo{
		Pattern: g.prefix + "/*",
		Type:    RouteTypeCatch,
	})
	return route
}

func (g *RouteGroup) Handle(path string, method string, handler http.HandlerFunc) *mux.Route {
	route := g.router.Path(path).Methods(method).HandlerFunc(handler)
	g.parent.routes = append(g.parent.routes, RouteInfo{
		Method:  method,
		Pattern: g.prefix + path,
		Type:    RouteTypeExact,
	})
	return route
}

func (g *RouteGroup) Get(path string, handler http.HandlerFunc) *mux.Route {
	return g.Handle(path, http.MethodGet, handler)
}

func (g *RouteGroup) Post(path string, handler http.HandlerFunc) *mux.Route {
	return g.Handle(path, http.MethodPost, handler)
}

func (g *RouteGroup) Put(path string, handler http.HandlerFunc) *mux.Route {
	return g.Handle(path, http.MethodPut, handler)
}

func (g *RouteGroup) Delete(path string, handler http.HandlerFunc) *mux.Route {
	return g.Handle(path, http.MethodDelete, handler)
}

func (g *RouteGroup) Patch(path string, handler http.HandlerFunc) *mux.Route {
	return g.Handle(path, http.MethodPatch, handler)
}

func (g *RouteGroup) Use(middlewares ...mux.MiddlewareFunc) {
	g.router.Use(middlewares...)
}

func (g *RouteGroup) UseChain(chain middleware.Chain) {
	g.router.Use(chain.Middleware()...)
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
