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
	RefreshToken(handler *req.RequestHandler)
	Logout(handler *req.RequestHandler)

	// Password management endpoints
	ForgotPassword(handler *req.RequestHandler)
	ResetPassword(handler *req.RequestHandler)
	ChangePassword(handler *req.RequestHandler)

	// Email verification endpoints
	VerifyEmail(handler *req.RequestHandler)
	ResendVerification(handler *req.RequestHandler)

	// Session management endpoints
	GetSessions(handler *req.RequestHandler)
	DeleteSession(handler *req.RequestHandler)

	// MFA endpoints
	MFAEnable(handler *req.RequestHandler)
	MFADisable(handler *req.RequestHandler)
	MFAVerify(handler *req.RequestHandler)

	// Account management endpoints
	DeleteAccount(handler *req.RequestHandler)
}

// Compile-time interface compliance check
var _ AuthHandlerInterface = (*AuthHandler)(nil)
