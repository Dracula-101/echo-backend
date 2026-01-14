package request

import "net/http"

// HandlerAdapter converts a RequestHandler-aware function into a standard http.HandlerFunc.
func Adapt(fn func(*RequestHandler)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqHandler := NewHandler(r, w)
		fn(reqHandler)
	}
}
