package messaging

import (
	"context"
)

type MiddlewareFunc func(Handler) Handler

func ChainMiddleware(handler Handler, middlewares ...MiddlewareFunc) Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

type ErrorHandler interface {
	HandleError(ctx context.Context, message *Message, err error) error
}

type ErrorHandlerFunc func(ctx context.Context, message *Message, err error) error

func (f ErrorHandlerFunc) HandleError(ctx context.Context, message *Message, err error) error {
	return f(ctx, message, err)
}
