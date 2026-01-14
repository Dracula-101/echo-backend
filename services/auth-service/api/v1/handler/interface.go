package handler

import req "shared/server/request"

// ============================================================================
// Handler Interfaces
// ============================================================================

// AuthHandlerInterface defines the contract for authentication HTTP handlers
type AuthHandlerInterface interface {
	// Authentication endpoints
	Register(handler *req.RequestHandler)
	Login(handler *req.RequestHandler)
}

// Compile-time interface compliance check
var _ AuthHandlerInterface = (*AuthHandler)(nil)
