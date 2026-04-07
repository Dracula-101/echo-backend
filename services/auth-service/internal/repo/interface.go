package repository

// ============================================================================
// Repository Interfaces
// ============================================================================

// Compile-time interface compliance checks
var (
	_ AuthRepositoryInterface                    = (*AuthRepository)(nil)
	_ LoginHistoryRepositoryInterface            = (*LoginHistoryRepo)(nil)
	_ SessionRepositoryInterface                 = (*SessionRepo)(nil)
	_ PasswordResetTokenRepositoryInterface      = (*PasswordResetTokenRepo)(nil)
	_ EmailVerificationTokenRepositoryInterface  = (*EmailVerificationTokenRepo)(nil)
)
