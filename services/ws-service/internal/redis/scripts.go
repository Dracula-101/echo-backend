package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
)

var (
	subscribeScript = redis.NewScript(`
		local subKey = KEYS[1]
		local connSubsKey = KEYS[2]
		local connID = ARGV[1]
		local topicKey = ARGV[2]
		local maxSubs = tonumber(ARGV[3])

		local currentCount = redis.call('SCARD', connSubsKey)
		if currentCount >= maxSubs then
			return {err = "max_subscriptions_reached"}
		end

		redis.call('SADD', subKey, connID)
		redis.call('SADD', connSubsKey, topicKey)
		return {ok = "subscribed"}
	`)

	unsubscribeScript = redis.NewScript(`
		local subKey = KEYS[1]
		local connSubsKey = KEYS[2]
		local connID = ARGV[1]
		local topicKey = ARGV[2]

		redis.call('SREM', subKey, connID)
		redis.call('SREM', connSubsKey, topicKey)
		return redis.call('SCARD', connSubsKey)
	`)

	unsubscribeAllScript = redis.NewScript(`
		local connSubsKey = KEYS[1]
		local connID = ARGV[1]
		local keyPrefix = ARGV[2]

		local topics = redis.call('SMEMBERS', connSubsKey)
		for _, topic in ipairs(topics) do
			local subKey = keyPrefix .. topic
			redis.call('SREM', subKey, connID)
		end
		redis.call('DEL', connSubsKey)
		return topics
	`)

	setPresenceScript = redis.NewScript(`
		local presenceKey = KEYS[1]
		local devicesKey = KEYS[2]
		local userConnsKey = KEYS[3]
		local connKey = KEYS[4]

		local userID = ARGV[1]
		local deviceID = ARGV[2]
		local platform = ARGV[3]
		local connID = ARGV[4]
		local status = ARGV[5]
		local ttl = tonumber(ARGV[6])

		redis.call('HSET', presenceKey, 'status', status, 'user_id', userID)
		redis.call('EXPIRE', presenceKey, ttl)

		redis.call('HSET', devicesKey, deviceID, platform)
		redis.call('EXPIRE', devicesKey, ttl)

		redis.call('SADD', userConnsKey, connID)
		redis.call('EXPIRE', userConnsKey, ttl)

		redis.call('HSET', connKey, 'user_id', userID, 'device_id', deviceID, 'platform', platform)
		redis.call('EXPIRE', connKey, ttl)

		return {ok = "presence_set"}
	`)

	removePresenceScript = redis.NewScript(`
		local presenceKey = KEYS[1]
		local devicesKey = KEYS[2]
		local userConnsKey = KEYS[3]
		local connKey = KEYS[4]

		local deviceID = ARGV[1]
		local connID = ARGV[2]

		redis.call('HDEL', devicesKey, deviceID)
		redis.call('SREM', userConnsKey, connID)
		redis.call('DEL', connKey)

		local remainingConns = redis.call('SCARD', userConnsKey)
		if remainingConns == 0 then
			redis.call('HSET', presenceKey, 'status', 'offline')
			return "offline"
		end
		return "online"
	`)

	setTypingScript = redis.NewScript(`
		local typingKey = KEYS[1]
		local userID = ARGV[1]
		local timestamp = ARGV[2]
		local ttl = tonumber(ARGV[3])

		redis.call('HSET', typingKey, userID, timestamp)
		redis.call('EXPIRE', typingKey, ttl)
		return {ok = "typing_set"}
	`)

	removeTypingScript = redis.NewScript(`
		local typingKey = KEYS[1]
		local userID = ARGV[1]

		redis.call('HDEL', typingKey, userID)
		return {ok = "typing_removed"}
	`)

	registerConnectionScript = redis.NewScript(`
		local connKey = KEYS[1]
		local userConnsKey = KEYS[2]
		local instanceKey = KEYS[3]

		local connID = ARGV[1]
		local userID = ARGV[2]
		local deviceID = ARGV[3]
		local platform = ARGV[4]
		local instanceID = ARGV[5]
		local ttl = tonumber(ARGV[6])

		redis.call('HSET', connKey,
			'user_id', userID,
			'device_id', deviceID,
			'platform', platform,
			'instance_id', instanceID,
			'connected_at', ARGV[7]
		)
		redis.call('EXPIRE', connKey, ttl)

		redis.call('SADD', userConnsKey, connID)
		redis.call('EXPIRE', userConnsKey, ttl)

		redis.call('SADD', instanceKey, connID)
		return {ok = "registered"}
	`)

	unregisterConnectionScript = redis.NewScript(`
		local connKey = KEYS[1]
		local userConnsKey = KEYS[2]
		local instanceKey = KEYS[3]
		local connSubsKey = KEYS[4]

		local connID = ARGV[1]
		local subKeyPrefix = ARGV[2]

		local topics = redis.call('SMEMBERS', connSubsKey)
		for _, topic in ipairs(topics) do
			local subKey = subKeyPrefix .. topic
			redis.call('SREM', subKey, connID)
		end

		redis.call('DEL', connKey)
		redis.call('DEL', connSubsKey)
		redis.call('SREM', userConnsKey, connID)
		redis.call('SREM', instanceKey, connID)

		return topics
	`)
)

type Scripts struct {
	client *Client
}

func NewScripts(client *Client) *Scripts {
	return &Scripts{client: client}
}

func (s *Scripts) Subscribe(ctx context.Context, connID, topic, resource string, maxSubs int) error {
	topicKey := topic + ":" + resource
	subKey := SubscriptionKey(topic, resource)
	connSubsKey := ConnSubsKey(connID)

	result, err := subscribeScript.Run(ctx, s.client.rdb,
		[]string{subKey, connSubsKey},
		connID, topicKey, maxSubs,
	).Result()

	if err != nil {
		return err
	}

	if m, ok := result.(map[any]any); ok {
		if _, hasErr := m["err"]; hasErr {
			return ErrMaxSubscriptionsReached
		}
	}

	return nil
}

func (s *Scripts) Unsubscribe(ctx context.Context, connID, topic, resource string) (int64, error) {
	topicKey := topic + ":" + resource
	subKey := SubscriptionKey(topic, resource)
	connSubsKey := ConnSubsKey(connID)

	return unsubscribeScript.Run(ctx, s.client.rdb,
		[]string{subKey, connSubsKey},
		connID, topicKey,
	).Int64()
}

func (s *Scripts) UnsubscribeAll(ctx context.Context, connID string) ([]string, error) {
	connSubsKey := ConnSubsKey(connID)

	result, err := unsubscribeAllScript.Run(ctx, s.client.rdb,
		[]string{connSubsKey},
		connID, PrefixSubscription,
	).Result()

	if err != nil {
		return nil, err
	}

	topics, ok := result.([]any)
	if !ok {
		return nil, nil
	}

	strs := make([]string, 0, len(topics))
	for _, t := range topics {
		if str, ok := t.(string); ok {
			strs = append(strs, str)
		}
	}
	return strs, nil
}

func (s *Scripts) SetPresence(ctx context.Context, userID, deviceID, platform, connID, status string, ttlSeconds int) error {
	presenceKey := PresenceKey(userID)
	devicesKey := DevicesKey(userID)
	userConnsKey := UserConnsKey(userID)
	connKey := ConnectionKey(connID)

	_, err := setPresenceScript.Run(ctx, s.client.rdb,
		[]string{presenceKey, devicesKey, userConnsKey, connKey},
		userID, deviceID, platform, connID, status, ttlSeconds,
	).Result()

	return err
}

func (s *Scripts) RemovePresence(ctx context.Context, userID, deviceID, connID string) (string, error) {
	presenceKey := PresenceKey(userID)
	devicesKey := DevicesKey(userID)
	userConnsKey := UserConnsKey(userID)
	connKey := ConnectionKey(connID)

	result, err := removePresenceScript.Run(ctx, s.client.rdb,
		[]string{presenceKey, devicesKey, userConnsKey, connKey},
		deviceID, connID,
	).Result()

	if err != nil {
		return "", err
	}

	return result.(string), nil
}

func (s *Scripts) SetTyping(ctx context.Context, convID, userID, timestamp string, ttlSeconds int) error {
	typingKey := TypingKey(convID)

	_, err := setTypingScript.Run(ctx, s.client.rdb,
		[]string{typingKey},
		userID, timestamp, ttlSeconds,
	).Result()

	return err
}

func (s *Scripts) RemoveTyping(ctx context.Context, convID, userID string) error {
	typingKey := TypingKey(convID)

	_, err := removeTypingScript.Run(ctx, s.client.rdb,
		[]string{typingKey},
		userID,
	).Result()

	return err
}

func (s *Scripts) RegisterConnection(ctx context.Context, connID, userID, deviceID, platform, instanceID, connectedAt string, ttlSeconds int) error {
	connKey := ConnectionKey(connID)
	userConnsKey := UserConnsKey(userID)
	instanceKey := InstanceKey(instanceID)

	_, err := registerConnectionScript.Run(ctx, s.client.rdb,
		[]string{connKey, userConnsKey, instanceKey},
		connID, userID, deviceID, platform, instanceID, ttlSeconds, connectedAt,
	).Result()

	return err
}

func (s *Scripts) UnregisterConnection(ctx context.Context, connID, userID, instanceID string) ([]string, error) {
	connKey := ConnectionKey(connID)
	userConnsKey := UserConnsKey(userID)
	instanceKey := InstanceKey(instanceID)
	connSubsKey := ConnSubsKey(connID)

	result, err := unregisterConnectionScript.Run(ctx, s.client.rdb,
		[]string{connKey, userConnsKey, instanceKey, connSubsKey},
		connID, PrefixSubscription,
	).Result()

	if err != nil {
		return nil, err
	}

	topics, ok := result.([]any)
	if !ok {
		return nil, nil
	}

	strs := make([]string, 0, len(topics))
	for _, t := range topics {
		if str, ok := t.(string); ok {
			strs = append(strs, str)
		}
	}
	return strs, nil
}
