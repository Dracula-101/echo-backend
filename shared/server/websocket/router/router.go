package router

import (
	"context"
	"fmt"
	"sync"
)

// Handler is a function that handles a message
type Handler func(ctx context.Context, msg *Message) error

// Message represents a routed message
type Message struct {
	Type    string
	Payload []byte
	Metadata map[string]interface{}
}

// Router routes messages to handlers based on message type
type Router struct {
	handlers map[string]Handler
	mu       sync.RWMutex

	// Fallback handler for unknown message types
	fallback Handler

	// Middleware chain
	middleware []Middleware
}

// New creates a new router
func New() *Router {
	return &Router{
		handlers:   make(map[string]Handler),
		middleware: make([]Middleware, 0),
	}
}

// Register registers a handler for a message type
func (r *Router) Register(messageType string, handler Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[messageType] = handler
}

// Unregister removes a handler
func (r *Router) Unregister(messageType string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.handlers, messageType)
}

// Use adds middleware to the router
func (r *Router) Use(mw Middleware) {
	r.middleware = append(r.middleware, mw)
}

// SetFallback sets the fallback handler
func (r *Router) SetFallback(handler Handler) {
	r.fallback = handler
}

// Route routes a message to its handler
func (r *Router) Route(ctx context.Context, msg *Message) error {
	r.mu.RLock()
	handler, exists := r.handlers[msg.Type]
	r.mu.RUnlock()

	if !exists {
		if r.fallback != nil {
			return r.applyMiddleware(ctx, msg, r.fallback)
		}
		return fmt.Errorf("no handler for message type: %s", msg.Type)
	}

	return r.applyMiddleware(ctx, msg, handler)
}

// applyMiddleware applies middleware chain
func (r *Router) applyMiddleware(ctx context.Context, msg *Message, handler Handler) error {
	finalHandler := handler

	// Apply middleware in reverse order
	for i := len(r.middleware) - 1; i >= 0; i-- {
		finalHandler = r.middleware[i](finalHandler)
	}

	return finalHandler(ctx, msg)
}

// HasHandler checks if a handler exists for a message type
func (r *Router) HasHandler(messageType string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.handlers[messageType]
	return exists
}
