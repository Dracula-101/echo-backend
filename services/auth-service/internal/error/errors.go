package error

import (
	pkgErrors "shared/pkg/errors"
)

type AuthError struct {
	Message string
	Code    AuthErrorCode
	Error   pkgErrors.AppError
}

const ServiceName = "auth-service"