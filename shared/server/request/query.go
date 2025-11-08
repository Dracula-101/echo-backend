package request

import (
	"fmt"
	"strconv"
)

// QueryParam extracts a query parameter from the request
func (h *RequestHandler) QueryParam(key string) string {
	return h.request.URL.Query().Get(key)
}

// QueryParamDefault extracts a query parameter with a default value
func (h *RequestHandler) QueryParamDefault(key, defaultVal string) string {
	val := h.request.URL.Query().Get(key)
	if val == "" {
		return defaultVal
	}
	return val
}

// QueryParamInt extracts an integer query parameter
func (h *RequestHandler) QueryParamInt(key string, defaultVal int) (int, error) {
	str := h.request.URL.Query().Get(key)
	if str == "" {
		return defaultVal, nil
	}

	val, err := strconv.Atoi(str)
	if err != nil {
		return defaultVal, fmt.Errorf("invalid integer value for %s", key)
	}

	return val, nil
}

// QueryParamInt64 extracts an int64 query parameter
func (h *RequestHandler) QueryParamInt64(key string, defaultVal int64) (int64, error) {
	str := h.request.URL.Query().Get(key)
	if str == "" {
		return defaultVal, nil
	}

	val, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return defaultVal, fmt.Errorf("invalid integer value for %s", key)
	}

	return val, nil
}

// QueryParamFloat extracts a float64 query parameter
func (h *RequestHandler) QueryParamFloat(key string, defaultVal float64) (float64, error) {
	str := h.request.URL.Query().Get(key)
	if str == "" {
		return defaultVal, nil
	}

	val, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return defaultVal, fmt.Errorf("invalid float value for %s", key)
	}

	return val, nil
}

// QueryParamBool extracts a boolean query parameter
func (h *RequestHandler) QueryParamBool(key string, defaultVal bool) (bool, error) {
	str := h.request.URL.Query().Get(key)
	if str == "" {
		return defaultVal, nil
	}

	val, err := strconv.ParseBool(str)
	if err != nil {
		return defaultVal, fmt.Errorf("invalid boolean value for %s", key)
	}

	return val, nil
}

// QueryParamArray extracts an array of query parameters
func (h *RequestHandler) QueryParamArray(key string) []string {
	return h.request.URL.Query()[key]
}
