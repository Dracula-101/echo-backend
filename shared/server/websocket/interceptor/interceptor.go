package interceptor

import (
	"context"
)

// Message represents an intercepted message
type Message struct {
	Data     []byte
	Metadata map[string]interface{}
}

// Interceptor intercepts and potentially modifies messages
type Interceptor interface {
	// Intercept is called for each message
	Intercept(ctx context.Context, msg *Message, next func(context.Context, *Message) error) error
}

// Chain represents an interceptor chain
type Chain struct {
	interceptors []Interceptor
}

// NewChain creates a new interceptor chain
func NewChain(interceptors ...Interceptor) *Chain {
	return &Chain{
		interceptors: interceptors,
	}
}

// Add adds an interceptor to the chain
func (c *Chain) Add(interceptor Interceptor) {
	c.interceptors = append(c.interceptors, interceptor)
}

// Execute executes the interceptor chain
func (c *Chain) Execute(ctx context.Context, msg *Message, final func(context.Context, *Message) error) error {
	if len(c.interceptors) == 0 {
		return final(ctx, msg)
	}

	// Build chain
	handler := final
	for i := len(c.interceptors) - 1; i >= 0; i-- {
		interceptor := c.interceptors[i]
		next := handler
		handler = func(ctx context.Context, msg *Message) error {
			return interceptor.Intercept(ctx, msg, next)
		}
	}

	return handler(ctx, msg)
}
