package main

import (
	"fmt"
	"net/http"
)

func main() {
	// Auth service owns its routes
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "OK")
	})
	
	http.HandleFunc("/api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"message":"login success"}`)
	})
	
	http.HandleFunc("/api/v1/auth/register", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"message":"register success"}`)
	})
	
	fmt.Println("Auth service running on :8081")
	http.ListenAndServe(":8081", nil)
}
