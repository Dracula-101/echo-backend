package domain

import (
	"time"

	"github.com/google/uuid"
)

type ConnectionInfo struct {
	UserID       uuid.UUID `json:"user_id"`
	DeviceID     string    `json:"device_id"`
	ConnectionID string    `json:"connection_id"`
	Platform     string    `json:"platform"`
	ConnectedAt  time.Time `json:"connected_at"`
}

type OnlineUsersResponse struct {
	Users []UserStatus `json:"users"`
	Total int          `json:"total"`
}

type UserStatus struct {
	UserID      uuid.UUID `json:"user_id"`
	IsOnline    bool      `json:"is_online"`
	DeviceCount int       `json:"device_count"`
}

type StatsResponse struct {
	TotalUsers       int   `json:"total_users"`
	TotalDevices     int   `json:"total_devices"`
	TotalConnections int64 `json:"total_connections"`
}

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
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (ConnectionRecord) TableName() string {
	return "websocket.connections"
}

func (ConnectionRecord) PrimaryKey() string {
	return "id"
}
