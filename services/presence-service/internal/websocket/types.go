package websocket

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// HubPresenceUpdate represents a presence update to broadcast
type HubPresenceUpdate struct {
	UserID       uuid.UUID
	OnlineStatus string
	CustomStatus string
	BroadcastTo  []uuid.UUID // Users to notify about this presence change
}

// HubTypingBroadcast represents a typing indicator to broadcast
type HubTypingBroadcast struct {
	ConversationID uuid.UUID
	UserID         uuid.UUID
	IsTyping       bool
	Participants   []uuid.UUID // Users in the conversation to notify
}

// ClientMetadata contains additional client information
type ClientMetadata struct {
	IPAddress   string
	UserAgent   string
	Platform    string // ios, android, web, desktop
	AppVersion  string
	DeviceName  string
	DeviceType  string // mobile, tablet, desktop
	ConnectedAt time.Time
}

// IncomingMessage represents messages received from clients
type IncomingMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// MessageType constants
const (
	MessageTypePresenceUpdate = "presence_update"
	MessageTypeHeartbeat      = "heartbeat"
	MessageTypeTyping         = "typing"
	MessageTypePing           = "ping"
	MessageTypePong           = "pong"
)

// Response message types
const (
	ResponseConnectionAck     = "connection_ack"
	ResponsePresenceUpdateAck = "presence_update_ack"
	ResponseHeartbeatAck      = "heartbeat_ack"
	ResponsePresenceUpdate    = "presence_update"
	ResponseTypingIndicator   = "typing_indicator"
)
