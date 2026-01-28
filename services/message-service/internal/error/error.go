package error

import (
	pkgErrors "shared/pkg/errors"
)

const ServiceName = "message_service"

type MessageErrorCode string

func (c MessageErrorCode) String() string {
	return string(c)
}

type MessageError struct {
	Code    MessageErrorCode
	Message string
	Error   pkgErrors.AppError
}
