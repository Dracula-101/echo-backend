package state

import (
	"context"
	"time"

	"shared/pkg/logger"
	"ws-service/internal/redis"

	"github.com/google/uuid"
)

type RedisConnectionRegistry struct {
	client     *redis.Client
	scripts    *redis.Scripts
	instanceID string
	log        logger.Logger
}

func NewRedisConnectionRegistry(client *redis.Client, scripts *redis.Scripts, instanceID string, log logger.Logger) *RedisConnectionRegistry {
	return &RedisConnectionRegistry{
		client:     client,
		scripts:    scripts,
		instanceID: instanceID,
		log:        log,
	}
}

func (r *RedisConnectionRegistry) Register(ctx context.Context, info *ConnectionInfo) error {
	connectedAt := info.ConnectedAt.Format(time.RFC3339)
	ttlSeconds := int(redis.TTLConnection.Seconds())

	instanceID := info.InstanceID
	if instanceID == "" {
		instanceID = r.instanceID
	}

	return r.scripts.RegisterConnection(ctx,
		info.ConnID,
		info.UserID.String(),
		info.DeviceID,
		info.Platform,
		instanceID,
		connectedAt,
		ttlSeconds,
	)
}

func (r *RedisConnectionRegistry) Unregister(ctx context.Context, connID string) ([]string, error) {
	connInfo, err := r.GetConnection(ctx, connID)
	if err != nil {
		return nil, err
	}
	if connInfo == nil {
		return nil, nil
	}

	instanceID := connInfo.InstanceID
	if instanceID == "" {
		instanceID = r.instanceID
	}

	return r.scripts.UnregisterConnection(ctx, connID, connInfo.UserID.String(), instanceID)
}

func (r *RedisConnectionRegistry) GetConnection(ctx context.Context, connID string) (*ConnectionInfo, error) {
	key := redis.ConnectionKey(connID)
	data, err := r.client.HGetAll(ctx, key)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, nil
	}

	userID, err := uuid.Parse(data["user_id"])
	if err != nil {
		return nil, err
	}

	connectedAt, _ := time.Parse(time.RFC3339, data["connected_at"])

	return &ConnectionInfo{
		ConnID:      connID,
		UserID:      userID,
		DeviceID:    data["device_id"],
		Platform:    data["platform"],
		InstanceID:  data["instance_id"],
		ConnectedAt: connectedAt,
	}, nil
}

func (r *RedisConnectionRegistry) GetUserConnections(ctx context.Context, userID uuid.UUID) ([]string, error) {
	key := redis.UserConnsKey(userID.String())
	return r.client.SMembers(ctx, key)
}

func (r *RedisConnectionRegistry) GetInstanceConnections(ctx context.Context, instanceID string) ([]string, error) {
	key := redis.InstanceKey(instanceID)
	return r.client.SMembers(ctx, key)
}

func (r *RedisConnectionRegistry) RefreshTTL(ctx context.Context, connID string) error {
	key := redis.ConnectionKey(connID)
	return r.client.Expire(ctx, key, redis.TTLConnection)
}
