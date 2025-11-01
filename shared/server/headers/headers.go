package headers

import (
	"net/http"
)

func GetHeader(r *http.Request, key string) string {
	return r.Header.Get(key)
}

func SetHeader(w http.ResponseWriter, key, value string) {
	w.Header().Set(key, value)
}

func AddHeader(w http.ResponseWriter, key, value string) {
	w.Header().Add(key, value)
}

func DelHeader(w http.ResponseWriter, key string) {
	w.Header().Del(key)
}
