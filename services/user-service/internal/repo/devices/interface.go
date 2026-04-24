package devices

import (
	"context"

	"user-service/internal/domain"

	pkgErrors "shared/pkg/errors"
)

type DeviceRepository interface {
	AddDevice(ctx context.Context, input *domain.UserDevice, isCurrentDevice bool) pkgErrors.AppError
	GetDevices(ctx context.Context, userID string) ([]*domain.UserDevice, pkgErrors.AppError)
	GetDevice(ctx context.Context, userID, deviceID string) (*domain.UserDevice, pkgErrors.AppError)
	UpdateDevice(ctx context.Context, input *domain.UpdateUserDevice, userID, deviceID string) pkgErrors.AppError
	DeactivateDevice(ctx context.Context, id string) pkgErrors.AppError
	UpdatePushToken(ctx context.Context, id string, fcmToken, apnsToken *string, pushEnabled *bool) pkgErrors.AppError
	UpdateAuthSessionPushToken(ctx context.Context, userID, deviceID string, fcmToken, apnsToken *string) pkgErrors.AppError
}

var _ DeviceRepository = (*deviceRepository)(nil)
