package request

import (
	"fmt"
	"strconv"

	"github.com/gorilla/mux"
)

// PathParam extracts a path parameter from the request using gorilla/mux
func (h *RequestHandler) PathParam(key string) string {
	vars := mux.Vars(h.request)
	return vars[key]
}

// PathParamInt extracts an integer path parameter
func (h *RequestHandler) PathParamInt(key string) (int, error) {
	val := h.PathParam(key)
	if val == "" {
		return 0, nil
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("invalid integer value for %s", key)
	}
	return intVal, nil
}

// PathParamInt64 extracts an int64 path parameter
func (h *RequestHandler) PathParamInt64(key string) (int64, error) {
	val := h.PathParam(key)
	if val == "" {
		return 0, nil
	}
	int64Val, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid int64 value for %s", key)
	}
	return int64Val, nil
}
