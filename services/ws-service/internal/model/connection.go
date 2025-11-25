package model

import (
	"time"

	"github.com/google/uuid"
)

// ConnectionInfo represents information about a WebSocket connection
type ConnectionInfo struct {
	UserID      uuid.UUID `json:"user_id"`
	DeviceID    string    `json:"device_id"`
	ConnectionID string   `json:"connection_id"`
	Platform    string    `json:"platform"`
	ConnectedAt time.Time `json:"connected_at"`
}

// OnlineUsersResponse represents response for online users query
type OnlineUsersResponse struct {
	Users []UserStatus `json:"users"`
	Total int          `json:"total"`
}

// UserStatus represents a user's online status
type UserStatus struct {
	UserID      uuid.UUID `json:"user_id"`
	IsOnline    bool      `json:"is_online"`
	DeviceCount int       `json:"device_count"`
}

// StatsResponse represents WebSocket hub statistics
type StatsResponse struct {
	TotalUsers       int   `json:"total_users"`
	TotalDevices     int   `json:"total_devices"`
	TotalConnections int64 `json:"total_connections"`
}

// ConnectionRecord represents a WebSocket connection record in the database
type ConnectionRecord struct {
	ID             uuid.UUID  `json:"id"`
	UserID         uuid.UUID  `json:"user_id"`
	DeviceID       string     `json:"device_id"`
	ClientID       string     `json:"client_id"`
	IPAddress      string     `json:"ip_address"`
	UserAgent      string     `json:"user_agent"`
	Platform       string     `json:"platform"`
	AppVersion     string     `json:"app_version"`
	ConnectedAt    time.Time  `json:"connected_at"`
	DisconnectedAt *time.Time `json:"disconnected_at,omitempty"`
	Status         string     `json:"status"` // active, disconnected, stale
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// TableName returns the table name for this model
func (ConnectionRecord) TableName() string {
	return "websocket.connections"
}

// PrimaryKey returns the primary key field name
func (ConnectionRecord) PrimaryKey() string {
	return "id"
}
