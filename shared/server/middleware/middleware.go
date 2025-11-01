package middleware

import (
	"net/http"

	"github.com/gorilla/mux"
)

type Chain struct {
	middlewares []mux.MiddlewareFunc
}

func (c *Chain) Append(middleware Handler) {
	c.middlewares = append(c.middlewares, func(next http.Handler) http.Handler {
		return middleware(next)
	})
}

func NewChain() *Chain {
	return &Chain{
		middlewares: make([]mux.MiddlewareFunc, 0),
	}
}

func (c *Chain) Use(middleware mux.MiddlewareFunc) *Chain {
	c.middlewares = append(c.middlewares, middleware)
	return c
}

func (c *Chain) Middleware() []mux.MiddlewareFunc {
	return c.middlewares
}

func (c *Chain) Apply(handler http.Handler) http.Handler {
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		handler = c.middlewares[i](handler)
	}
	return handler
}

func (c *Chain) ApplyFunc(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c.Apply(handler).ServeHTTP(w, r)
	}
}

func (c *Chain) ApplyToRouter(r *mux.Router) {
	for _, mw := range c.middlewares {
		r.Use(mw)
	}
}

func (c *Chain) Merge(other *Chain) *Chain {
	c.middlewares = append(c.middlewares, other.middlewares...)
	return c
}

func (c *Chain) Copy() *Chain {
	newChain := NewChain()
	newChain.middlewares = make([]mux.MiddlewareFunc, len(c.middlewares))
	copy(newChain.middlewares, c.middlewares)
	return newChain
}

func ApplyMiddleware(handler http.HandlerFunc, middlewares []mux.MiddlewareFunc) http.HandlerFunc {
	var h http.Handler = handler
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
	}
}
