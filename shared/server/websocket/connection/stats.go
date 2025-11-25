package connection

import "time"

// Stats represents connection statistics
type Stats struct {
	ID               string        `json:"id"`
	State            string        `json:"state"`
	CreatedAt        time.Time     `json:"created_at"`
	LastActivity     time.Time     `json:"last_activity"`
	MessagesSent     int64         `json:"messages_sent"`
	MessagesReceived int64         `json:"messages_received"`
	BytesSent        int64         `json:"bytes_sent"`
	BytesReceived    int64         `json:"bytes_received"`
	Uptime           time.Duration `json:"uptime"`
}
