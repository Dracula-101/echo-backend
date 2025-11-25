package middleware

import (
	"context"
)

// Handler is a WebSocket message handler
type Handler func(ctx context.Context, data []byte) error

// Middleware is a function that wraps a handler
type Middleware func(Handler) Handler

// Chain represents a middleware chain
type Chain struct {
	middleware []Middleware
}

// NewChain creates a new middleware chain
func NewChain(middleware ...Middleware) *Chain {
	return &Chain{
		middleware: middleware,
	}
}

// Append adds middleware to the chain
func (c *Chain) Append(middleware ...Middleware) *Chain {
	c.middleware = append(c.middleware, middleware...)
	return c
}

// Then wraps a handler with the middleware chain
func (c *Chain) Then(handler Handler) Handler {
	// Apply middleware in reverse order
	for i := len(c.middleware) - 1; i >= 0; i-- {
		handler = c.middleware[i](handler)
	}
	return handler
}
