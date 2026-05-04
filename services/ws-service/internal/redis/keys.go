package redis

import (
	"errors"
	"time"
)

var (
	ErrMaxSubscriptionsReached = errors.New("max subscriptions reached")
	ErrConnectionNotFound      = errors.New("connection not found")
	ErrUserNotFound            = errors.New("user not found")
)

const (
	PrefixSubscription = "ws:sub:"
	PrefixConnSubs     = "ws:conn:subs:"
	PrefixPresence     = "ws:presence:"
	PrefixDevices      = "ws:devices:"
	PrefixTyping       = "ws:typing:"
	PrefixConnection   = "ws:conn:"
	PrefixUserConns    = "ws:user:conns:"
	PrefixInstance     = "ws:instance:"

	ChannelBroadcast = "ws:broadcast"
	ChannelPresence  = "ws:presence:events"
	ChannelTyping    = "ws:typing:events"

	TTLTyping     = 8 * time.Second
	TTLPresence   = 35 * time.Second
	TTLConnection = 35 * time.Second

	MaxSubscriptionsPerConn = 100
)

func SubscriptionKey(topic, resource string) string {
	return PrefixSubscription + topic + ":" + resource
}

func ConnSubsKey(connID string) string {
	return PrefixConnSubs + connID
}

func PresenceKey(userID string) string {
	return PrefixPresence + userID
}

func DevicesKey(userID string) string {
	return PrefixDevices + userID
}

func TypingKey(convID string) string {
	return PrefixTyping + convID
}

func ConnectionKey(connID string) string {
	return PrefixConnection + connID
}

func UserConnsKey(userID string) string {
	return PrefixUserConns + userID
}

func InstanceKey(instanceID string) string {
	return PrefixInstance + instanceID
}
