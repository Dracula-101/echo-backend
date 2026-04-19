package devices

import (
	"context"

	"user-service/internal/domain"

	pkgErrors "shared/pkg/errors"
)

type DeviceRepository interface {
	AddDevice(ctx context.Context, input *domain.UserDevice, isCurrentDevice bool) pkgErrors.AppError
	GetDevices(ctx context.Context, userID string) ([]*domain.UserDevice, pkgErrors.AppError)
	UpdateDevice(ctx context.Context, input *domain.UpdateUserDevice, userID, deviceID string) pkgErrors.AppError
}

var _ DeviceRepository = (*deviceRepository)(nil)
