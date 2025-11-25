package presence

import (
	"sync"
	"time"
	"shared/server/websocket"

	"shared/pkg/logger"

	"github.com/google/uuid"
)

// UserPresence represents a user's presence information
type UserPresence struct {
	UserID       uuid.UUID
	Status       string // online, away, busy, offline
	CustomStatus string
	LastSeenAt   time.Time
	DeviceCount  int
}

// Tracker tracks user presence through WebSocket connections
type Tracker struct {
	hub       *websocket.Hub
	presences map[uuid.UUID]*UserPresence
	mu        sync.RWMutex
	log       logger.Logger
}

// NewTracker creates a new presence tracker
func NewTracker(hub *websocket.Hub, log logger.Logger) *Tracker {
	return &Tracker{
		hub:       hub,
		presences: make(map[uuid.UUID]*UserPresence),
		log:       log,
	}
}

// UpdatePresence updates a user's presence status
func (t *Tracker) UpdatePresence(userID uuid.UUID, status, customStatus string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	presence, exists := t.presences[userID]
	if !exists {
		presence = &UserPresence{
			UserID: userID,
		}
		t.presences[userID] = presence
	}

	presence.Status = status
	presence.CustomStatus = customStatus
	presence.LastSeenAt = time.Now()
	presence.DeviceCount = t.hub.GetUserDeviceCount(userID)

	t.log.Debug("Presence updated",
		logger.String("user_id", userID.String()),
		logger.String("status", status),
		logger.Int("devices", presence.DeviceCount),
	)

	return nil
}

// GetPresence returns a user's presence
func (t *Tracker) GetPresence(userID uuid.UUID) *UserPresence {
	t.mu.RLock()
	defer t.mu.RUnlock()

	presence, exists := t.presences[userID]
	if !exists {
		// Check if user is online via hub
		if t.hub.IsUserOnline(userID) {
			return &UserPresence{
				UserID:      userID,
				Status:      "online",
				LastSeenAt:  time.Now(),
				DeviceCount: t.hub.GetUserDeviceCount(userID),
			}
		}

		return &UserPresence{
			UserID:      userID,
			Status:      "offline",
			DeviceCount: 0,
		}
	}

	// Update device count from hub
	presence.DeviceCount = t.hub.GetUserDeviceCount(userID)

	// Auto-update status based on online state
	if presence.DeviceCount > 0 && presence.Status == "offline" {
		presence.Status = "online"
	} else if presence.DeviceCount == 0 && presence.Status != "offline" {
		presence.Status = "offline"
		presence.LastSeenAt = time.Now()
	}

	return presence
}

// GetBulkPresence returns presence for multiple users
func (t *Tracker) GetBulkPresence(userIDs []uuid.UUID) map[uuid.UUID]*UserPresence {
	result := make(map[uuid.UUID]*UserPresence)

	for _, userID := range userIDs {
		result[userID] = t.GetPresence(userID)
	}

	return result
}

// BroadcastPresenceUpdate broadcasts a presence update to subscribers
func (t *Tracker) BroadcastPresenceUpdate(userID uuid.UUID, status, customStatus string) {
	// TODO: Implement broadcasting to presence subscribers
	// This would integrate with the subscription manager and broadcaster

	t.log.Debug("Broadcasting presence update",
		logger.String("user_id", userID.String()),
		logger.String("status", status),
	)
}

// OnUserConnected handles user connection event
func (t *Tracker) OnUserConnected(userID uuid.UUID) {
	t.UpdatePresence(userID, "online", "")

	t.log.Info("User connected",
		logger.String("user_id", userID.String()),
		logger.Int("devices", t.hub.GetUserDeviceCount(userID)),
	)

	// Broadcast online status
	t.BroadcastPresenceUpdate(userID, "online", "")
}

// OnUserDisconnected handles user disconnection event
func (t *Tracker) OnUserDisconnected(userID uuid.UUID) {
	// Check if user still has active devices
	deviceCount := t.hub.GetUserDeviceCount(userID)

	if deviceCount == 0 {
		t.UpdatePresence(userID, "offline", "")

		t.log.Info("User disconnected (all devices)",
			logger.String("user_id", userID.String()),
		)

		// Broadcast offline status
		t.BroadcastPresenceUpdate(userID, "offline", "")
	} else {
		t.log.Debug("User still has active devices",
			logger.String("user_id", userID.String()),
			logger.Int("devices", deviceCount),
		)
	}
}

// GetOnlineUsers returns all currently online users
func (t *Tracker) GetOnlineUsers() []uuid.UUID {
	return t.hub.GetOnlineUsers()
}

// GetOnlineCount returns the number of online users
func (t *Tracker) GetOnlineCount() int {
	return len(t.hub.GetOnlineUsers())
}

// CleanupStalePresence removes stale presence data
func (t *Tracker) CleanupStalePresence(maxAge time.Duration) int {
	t.mu.Lock()
	defer t.mu.Unlock()

	cleaned := 0
	now := time.Now()

	for userID, presence := range t.presences {
		// Skip online users
		if presence.DeviceCount > 0 {
			continue
		}

		// Remove if last seen is too old
		if now.Sub(presence.LastSeenAt) > maxAge {
			delete(t.presences, userID)
			cleaned++
		}
	}

	if cleaned > 0 {
		t.log.Info("Cleaned up stale presence data",
			logger.Int("count", cleaned),
		)
	}

	return cleaned
}

// GetStats returns presence statistics
func (t *Tracker) GetStats() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_tracked"] = len(t.presences)
	stats["online_users"] = t.GetOnlineCount()

	statusCount := make(map[string]int)
	for _, presence := range t.presences {
		statusCount[presence.Status]++
	}
	stats["by_status"] = statusCount

	return stats
}
