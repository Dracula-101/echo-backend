package connection

import "github.com/google/uuid"

const (
	MetaKeyUserID    = "user_id"
	MetaKeyDeviceID  = "device_id"
	MetaKeyPlatform  = "platform"
	MetaKeyAuthToken = "auth_token"
)

type Metadata struct {
	UserID    uuid.UUID
	DeviceID  string
	Platform  string
	AuthToken string
}

func ExtractMetadata(conn Connection) Metadata {
	m := Metadata{}

	if val, ok := conn.GetMetadata(MetaKeyUserID); ok {
		if userID, ok := val.(uuid.UUID); ok {
			m.UserID = userID
		}
	}

	if val, ok := conn.GetMetadata(MetaKeyDeviceID); ok {
		if deviceID, ok := val.(string); ok {
			m.DeviceID = deviceID
		}
	}

	if val, ok := conn.GetMetadata(MetaKeyPlatform); ok {
		if platform, ok := val.(string); ok {
			m.Platform = platform
		}
	}

	if val, ok := conn.GetMetadata(MetaKeyAuthToken); ok {
		if token, ok := val.(string); ok {
			m.AuthToken = token
		}
	}

	return m
}
