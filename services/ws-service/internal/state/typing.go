package state

import (
	"context"
	"strconv"
	"time"

	"shared/pkg/logger"
	"ws-service/internal/redis"

	"github.com/google/uuid"
)

type RedisTypingStore struct {
	client  *redis.Client
	scripts *redis.Scripts
	log     logger.Logger
}

func NewRedisTypingStore(client *redis.Client, scripts *redis.Scripts, log logger.Logger) *RedisTypingStore {
	return &RedisTypingStore{
		client:  client,
		scripts: scripts,
		log:     log,
	}
}

func (s *RedisTypingStore) StartTyping(ctx context.Context, convID, userID uuid.UUID) error {
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	ttlSeconds := int(redis.TTLTyping.Seconds())
	return s.scripts.SetTyping(ctx, convID.String(), userID.String(), timestamp, ttlSeconds)
}

func (s *RedisTypingStore) StopTyping(ctx context.Context, convID, userID uuid.UUID) error {
	return s.scripts.RemoveTyping(ctx, convID.String(), userID.String())
}

func (s *RedisTypingStore) GetTypingUsers(ctx context.Context, convID uuid.UUID) ([]uuid.UUID, error) {
	key := redis.TypingKey(convID.String())
	data, err := s.client.HGetAll(ctx, key)
	if err != nil {
		return nil, err
	}

	now := time.Now().UnixMilli()
	ttlMs := redis.TTLTyping.Milliseconds()
	users := make([]uuid.UUID, 0, len(data))

	for userIDStr, timestampStr := range data {
		timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			continue
		}

		if now-timestamp > ttlMs {
			continue
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			continue
		}

		users = append(users, userID)
	}

	return users, nil
}
