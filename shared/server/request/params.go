package request

import (
	"net/http"

	"github.com/gorilla/mux"
)

// PathParam extracts a path parameter from the request using gorilla/mux
func PathParam(r *http.Request, key string) string {
	vars := mux.Vars(r)
	return vars[key]
}

// PathParamInt extracts an integer path parameter
func PathParamInt(r *http.Request, key string) (int, error) {
	val := PathParam(r, key)
	if val == "" {
		return 0, nil
	}
	return parseInt(val)
}

// PathParamInt64 extracts an int64 path parameter
func PathParamInt64(r *http.Request, key string) (int64, error) {
	val := PathParam(r, key)
	if val == "" {
		return 0, nil
	}
	return parseInt64(val)
}
