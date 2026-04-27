package error

import (
	pkgErrors "shared/pkg/errors"
)

type MsgError struct {
	Message string
	Code    MessageErrorCode
	Error   pkgErrors.AppError
	Detail  map[string]interface{}
}

const ServiceName = "messaging-service"
