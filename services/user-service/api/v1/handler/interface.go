package handler

import req "shared/server/request"

// ============================================================================
// Handler Interface
// ============================================================================

type UserHandlerInterface interface {
	// Profile endpoints
	GetProfile(handler *req.RequestHandler)
	CreateProfile(handler *req.RequestHandler)
}

// Compile-time interface compliance check
var _ UserHandlerInterface = (*UserHandler)(nil)
