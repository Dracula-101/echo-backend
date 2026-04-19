package devices

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"user-service/internal/domain"
	userErrors "user-service/internal/errors"

	"shared/pkg/database"
	dbModels "shared/pkg/database/postgres/models"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/pkg/utils"
)

type deviceRepository struct {
	db  database.Database
	log logger.Logger
}

func NewDeviceRepository(db database.Database, log logger.Logger) DeviceRepository {
	if db == nil {
		panic("database is required for DeviceRepository")
	}
	if log == nil {
		panic("logger is required for DeviceRepository")
	}
	return &deviceRepository{db: db, log: log}
}

func (r *deviceRepository) AddDevice(ctx context.Context, input *domain.UserDevice, isCurrentDevice bool) pkgErrors.AppError {
	model := dbModels.Device{
		UserID:          input.UserID,
		DeviceID:        input.DeviceID,
		DeviceName:      utils.PtrString(input.DeviceName),
		AppVersion:      input.AppVersion,
		IsCurrentDevice: isCurrentDevice,
		IsActive:        input.FCMToken != nil || input.APNSToken != nil,
		LastActiveAt:    time.Now(),
		RegisteredAt:    time.Now(),
		FCMToken:        input.FCMToken,
		APNSToken:       input.APNSToken,
		PushEnabled:     input.PushEnabled,
		Metadata:        json.RawMessage("{}"),
	}
	if input.DeviceType != nil {
		s := string(*input.DeviceType)
		model.DeviceType = &s
	}
	model.DeviceModel = input.DeviceModel
	model.DeviceManufacturer = input.DeviceManufacturer
	model.OSName = input.OSName
	model.OSVersion = input.OSVersion

	if _, dbErr := r.db.Insert(ctx, &model); dbErr != nil {
		r.log.Error("failed to add device",
			logger.String("user_id", input.UserID),
			logger.String("device_id", input.DeviceID),
			logger.Error(dbErr),
		)
		return pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "failed to add device")
	}
	return nil
}

func (r *deviceRepository) GetDevices(ctx context.Context, userID string) ([]*domain.UserDevice, pkgErrors.AppError) {
	query := `SELECT * FROM users.devices WHERE user_id = $1 AND is_active = true`
	rows, dbErr := r.db.Query(ctx, query, userID)
	if dbErr != nil {
		r.log.Error("failed to get devices",
			logger.String("user_id", userID),
			logger.Error(dbErr),
		)
		return nil, pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "failed to get devices")
	}
	defer rows.Close()

	var devices []*domain.UserDevice
	for rows.Next() {
		var d dbModels.Device
		if scanErr := rows.ScanModel(&d); scanErr != nil {
			r.log.Error("failed to scan device",
				logger.String("user_id", userID),
				logger.Error(scanErr),
			)
			continue
		}
		devices = append(devices, domain.NewUserDevice(d))
	}
	return devices, nil
}

func (r *deviceRepository) UpdateDevice(ctx context.Context, input *domain.UpdateUserDevice, userID, deviceID string) pkgErrors.AppError {
	setClauses := []string{"last_active_at = NOW()"}
	args := []any{}
	argIdx := 1

	addParam := func(col string, val any) {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", col, argIdx))
		args = append(args, val)
		argIdx++
	}

	if input.PushEnabled != nil {
		addParam("push_enabled", *input.PushEnabled)
	}
	if input.FCMToken != nil {
		addParam("fcm_token", *input.FCMToken)
	}
	if input.APNSToken != nil {
		addParam("apns_token", *input.APNSToken)
	}
	if input.IsActive != nil {
		addParam("is_active", *input.IsActive)
	}
	if input.AppVersion != nil {
		addParam("app_version", *input.AppVersion)
	}

	args = append(args, userID, deviceID)
	query := fmt.Sprintf(`
		UPDATE users.devices SET %s
		WHERE user_id = $%d AND device_id = $%d`,
		strings.Join(setClauses, ", "), argIdx, argIdx+1,
	)

	if _, dbErr := r.db.Exec(ctx, query, args...); dbErr != nil {
		r.log.Error("failed to update device",
			logger.String("user_id", userID),
			logger.String("device_id", deviceID),
			logger.Error(dbErr),
		)
		return pkgErrors.FromError(dbErr, userErrors.ErrCodeDatabaseError, "failed to update device")
	}
	return nil
}
