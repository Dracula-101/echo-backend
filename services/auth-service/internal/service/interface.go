package service

// ============================================================================
// Service Interfaces
// ============================================================================

// Compile-time interface compliance checks
var (
	_ AuthServiceInterface     = (*AuthService)(nil)
	_ SessionServiceInterface  = (*SessionService)(nil)
	_ LocationServiceInterface = (*LocationService)(nil)
)
