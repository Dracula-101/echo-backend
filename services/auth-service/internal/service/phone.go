package service

import (
	"auth-service/internal/domain"
	"auth-service/internal/error"
	authErrors "auth-service/internal/error"
	"context"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
)

func (s *authService) IsPhoneTaken(ctx context.Context, phone string) (*domain.User, *error.AuthError) {
	s.log.Info("Checking if phone number is taken", logger.String("phone", phone))

	user, err := s.repo.GetUserByPhone(ctx, phone)
	if err != nil {
		return nil, &error.AuthError{
			Message: "Failed to check phone number existence",
			Code:    authErrors.CodeDatabaseError,
			Error: pkgErrors.FromError(err, pkgErrors.CodeDatabaseError, "failed to check phone number existence").
				WithService(authErrors.ServiceName).
				WithDetail("phone", phone),
		}
	}
	return user, nil
}
