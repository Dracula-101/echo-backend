package state

import (
	"context"
	"strconv"
	"time"

	"shared/pkg/logger"
	"ws-service/internal/redis"

	"github.com/google/uuid"
)

type RedisPresenceStore struct {
	client  *redis.Client
	scripts *redis.Scripts
	log     logger.Logger
}

func NewRedisPresenceStore(client *redis.Client, scripts *redis.Scripts, log logger.Logger) *RedisPresenceStore {
	return &RedisPresenceStore{
		client:  client,
		scripts: scripts,
		log:     log,
	}
}

func (s *RedisPresenceStore) SetOnline(ctx context.Context, userID uuid.UUID, deviceID, platform, connID string) error {
	ttlSeconds := int(redis.TTLPresence.Seconds())
	return s.scripts.SetPresence(ctx, userID.String(), deviceID, platform, connID, "online", ttlSeconds)
}

func (s *RedisPresenceStore) SetOffline(ctx context.Context, userID uuid.UUID, deviceID, connID string) (bool, error) {
	result, err := s.scripts.RemovePresence(ctx, userID.String(), deviceID, connID)
	if err != nil {
		return false, err
	}
	return result == "offline", nil
}

func (s *RedisPresenceStore) UpdateStatus(ctx context.Context, userID uuid.UUID, status, customStatus string) error {
	key := redis.PresenceKey(userID.String())
	fields := map[string]any{
		"status":        status,
		"custom_status": customStatus,
		"updated_at":    time.Now().Unix(),
	}
	return s.client.HSet(ctx, key, fields)
}

func (s *RedisPresenceStore) GetPresence(ctx context.Context, userID uuid.UUID) (*PresenceInfo, error) {
	presenceKey := redis.PresenceKey(userID.String())
	devicesKey := redis.DevicesKey(userID.String())

	presenceData, err := s.client.HGetAll(ctx, presenceKey)
	if err != nil {
		return nil, err
	}

	if len(presenceData) == 0 {
		return &PresenceInfo{
			UserID: userID,
			Status: "offline",
		}, nil
	}

	devicesData, err := s.client.HGetAll(ctx, devicesKey)
	if err != nil {
		return nil, err
	}

	devices := make([]DeviceInfo, 0, len(devicesData))
	for deviceID, platform := range devicesData {
		devices = append(devices, DeviceInfo{
			DeviceID: deviceID,
			Platform: platform,
		})
	}

	info := &PresenceInfo{
		UserID:       userID,
		Status:       presenceData["status"],
		CustomStatus: presenceData["custom_status"],
		Devices:      devices,
	}

	if lastSeen, ok := presenceData["updated_at"]; ok {
		if ts, err := strconv.ParseInt(lastSeen, 10, 64); err == nil {
			info.LastSeenAt = time.Unix(ts, 0)
		}
	}

	return info, nil
}

func (s *RedisPresenceStore) GetPresenceBatch(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*PresenceInfo, error) {
	result := make(map[uuid.UUID]*PresenceInfo, len(userIDs))

	for _, userID := range userIDs {
		info, err := s.GetPresence(ctx, userID)
		if err != nil {
			s.log.Error("failed to get presence",
				logger.String("user_id", userID.String()),
				logger.Error(err),
			)
			continue
		}
		result[userID] = info
	}

	return result, nil
}

func (s *RedisPresenceStore) IsOnline(ctx context.Context, userID uuid.UUID) (bool, error) {
	key := redis.PresenceKey(userID.String())
	status, err := s.client.HGet(ctx, key, "status")
	if err != nil {
		return false, err
	}
	return status == "online", nil
}
