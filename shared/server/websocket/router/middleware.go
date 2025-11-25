package router

// Middleware is a function that wraps a handler
type Middleware func(Handler) Handler

// Chain chains multiple middleware together
func Chain(middleware ...Middleware) Middleware {
	return func(final Handler) Handler {
		for i := len(middleware) - 1; i >= 0; i-- {
			final = middleware[i](final)
		}
		return final
	}
}
