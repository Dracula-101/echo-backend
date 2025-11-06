package request

import (
	"fmt"
	"net/http"
	"strconv"
)

// QueryParam extracts a query parameter from the request
func QueryParam(r *http.Request, key string) string {
	return r.URL.Query().Get(key)
}

// QueryParamDefault extracts a query parameter with a default value
func QueryParamDefault(r *http.Request, key, defaultVal string) string {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultVal
	}
	return val
}

// GetIntQueryParam extracts an integer query parameter with a default value
func GetIntQueryParam(r *http.Request, key string, defaultVal int) int {
	str := r.URL.Query().Get(key)
	if str == "" {
		return defaultVal
	}

	val, err := strconv.Atoi(str)
	if err != nil {
		return defaultVal
	}

	return val
}

// QueryParamInt extracts an integer query parameter
func QueryParamInt(r *http.Request, key string, defaultVal int) (int, error) {
	str := r.URL.Query().Get(key)
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
func QueryParamInt64(r *http.Request, key string, defaultVal int64) (int64, error) {
	str := r.URL.Query().Get(key)
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
func QueryParamFloat(r *http.Request, key string, defaultVal float64) (float64, error) {
	str := r.URL.Query().Get(key)
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
func QueryParamBool(r *http.Request, key string, defaultVal bool) (bool, error) {
	str := r.URL.Query().Get(key)
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
func QueryParamArray(r *http.Request, key string) []string {
	return r.URL.Query()[key]
}

// Helper functions for internal use
func parseInt(s string) (int, error) {
	return strconv.Atoi(s)
}

func parseInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}
