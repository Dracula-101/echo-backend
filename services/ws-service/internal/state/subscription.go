package state

import (
	"context"

	"shared/pkg/logger"
	"ws-service/internal/redis"
)

type RedisSubscriptionStore struct {
	client  *redis.Client
	scripts *redis.Scripts
	log     logger.Logger
}

func NewRedisSubscriptionStore(client *redis.Client, scripts *redis.Scripts, log logger.Logger) *RedisSubscriptionStore {
	return &RedisSubscriptionStore{
		client:  client,
		scripts: scripts,
		log:     log,
	}
}

func (s *RedisSubscriptionStore) Subscribe(ctx context.Context, connID, topicKey string) error {
	topic, resource := parseTopicKey(topicKey)
	return s.scripts.Subscribe(ctx, connID, topic, resource, redis.MaxSubscriptionsPerConn)
}

func (s *RedisSubscriptionStore) Unsubscribe(ctx context.Context, connID, topicKey string) error {
	topic, resource := parseTopicKey(topicKey)
	_, err := s.scripts.Unsubscribe(ctx, connID, topic, resource)
	return err
}

func (s *RedisSubscriptionStore) UnsubscribeAll(ctx context.Context, connID string) ([]string, error) {
	return s.scripts.UnsubscribeAll(ctx, connID)
}

func (s *RedisSubscriptionStore) GetSubscribers(ctx context.Context, topicKey string) ([]string, error) {
	key := redis.PrefixSubscription + topicKey
	return s.client.SMembers(ctx, key)
}

func (s *RedisSubscriptionStore) GetSubscriptions(ctx context.Context, connID string) ([]string, error) {
	key := redis.ConnSubsKey(connID)
	return s.client.SMembers(ctx, key)
}

func (s *RedisSubscriptionStore) IsSubscribed(ctx context.Context, connID, topicKey string) (bool, error) {
	key := redis.PrefixSubscription + topicKey
	return s.client.SIsMember(ctx, key, connID)
}

func (s *RedisSubscriptionStore) CountSubscriptions(ctx context.Context, connID string) (int, error) {
	key := redis.ConnSubsKey(connID)
	count, err := s.client.SCard(ctx, key)
	return int(count), err
}

func parseTopicKey(topicKey string) (topic, resource string) {
	for i := 0; i < len(topicKey); i++ {
		if topicKey[i] == ':' {
			return topicKey[:i], topicKey[i+1:]
		}
	}
	return topicKey, "default"
}
