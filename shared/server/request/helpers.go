package request

import (
	"net/http"
	"strconv"
)

type QueryParser struct{}

func NewQueryParser() *QueryParser {
	return &QueryParser{}
}

func (q *QueryParser) GetString(r *http.Request, key string, defaultVal ...string) string {
	val := r.URL.Query().Get(key)
	if val == "" && len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return val
}

func (q *QueryParser) GetInt(r *http.Request, key string, defaultVal int) (int, error) {
	str := r.URL.Query().Get(key)
	if str == "" {
		return defaultVal, nil
	}

	val, err := strconv.Atoi(str)
	if err != nil {
		return defaultVal, err
	}
	return val, nil
}

func (q *QueryParser) GetInt64(r *http.Request, key string, defaultVal int64) (int64, error) {
	str := r.URL.Query().Get(key)
	if str == "" {
		return defaultVal, nil
	}

	val, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return defaultVal, err
	}
	return val, nil
}

func (q *QueryParser) GetBool(r *http.Request, key string, defaultVal bool) (bool, error) {
	str := r.URL.Query().Get(key)
	if str == "" {
		return defaultVal, nil
	}

	val, err := strconv.ParseBool(str)
	if err != nil {
		return defaultVal, err
	}
	return val, nil
}

func (q *QueryParser) GetArray(r *http.Request, key string) []string {
	return r.URL.Query()[key]
}

func (q *QueryParser) GetPathParam(r *http.Request, key string) string {
	return r.PathValue(key)
}

