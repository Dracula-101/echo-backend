package handler

import (
	"net/http"
)

// ============================================================================
// Handler Interface
// ============================================================================

type UserHandlerInterface interface {
	// Profile endpoints
	GetProfile(w http.ResponseWriter, r *http.Request)
	CreateProfile(w http.ResponseWriter, r *http.Request)
}

// Compile-time interface compliance check
var _ UserHandlerInterface = (*UserHandler)(nil)
