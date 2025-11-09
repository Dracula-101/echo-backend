package handler

import (
	"net/http"
)

// ============================================================================
// Handler Interfaces
// ============================================================================

// AuthHandlerInterface defines the contract for authentication HTTP handlers
type AuthHandlerInterface interface {
	// Authentication endpoints
	Register(w http.ResponseWriter, r *http.Request)
	Login(w http.ResponseWriter, r *http.Request)
}

// Compile-time interface compliance check
var _ AuthHandlerInterface = (*AuthHandler)(nil)
