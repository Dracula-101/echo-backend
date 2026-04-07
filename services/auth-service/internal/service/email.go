package service

import (
	"auth-service/internal/error"
	authErrors "auth-service/internal/error"
	"context"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
)

// ============================================================================
// Email Validation
// ============================================================================

func (s *AuthService) IsEmailTaken(ctx context.Context, email string) (bool, *error.AuthError) {
	s.log.Info("Checking if email is taken", logger.String("email", email))
	email = normalizeEmail(email)

	exists, err := s.repo.ExistsByEmail(ctx, email)
	if err != nil {
		return false, &error.AuthError{
			Message: "Failed to check email existence",
			Code:    authErrors.CodeDatabaseError,
			Error: pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to check email existence").
				WithService(authErrors.ServiceName).
				WithDetail("email", email),
		}
	}
	return exists, nil
}
