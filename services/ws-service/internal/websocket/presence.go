package websocket

import (
	"sync"
	"time"

	"shared/pkg/logger"

	"github.com/google/uuid"
)

// PresenceStatus represents user presence status
type PresenceStatus string

const (
	StatusOnline  PresenceStatus = "online"
	StatusOffline PresenceStatus = "offline"
	StatusAway    PresenceStatus = "away"
	StatusBusy    PresenceStatus = "busy"
)

// PresenceInfo represents user presence information
type PresenceInfo struct {
	UserID       uuid.UUID      `json:"user_id"`
	Status       PresenceStatus `json:"status"`
	CustomStatus string         `json:"custom_status,omitempty"`
	LastSeenAt   time.Time      `json:"last_seen_at"`
	DeviceCount  int            `json:"device_count"`
}

// PresenceTracker tracks user presence (application-specific)
type PresenceTracker struct {
	presences map[uuid.UUID]*PresenceInfo
	mu        sync.RWMutex
	log       logger.Logger
}

// NewPresenceTracker creates a new presence tracker
func NewPresenceTracker(log logger.Logger) *PresenceTracker {
	return &PresenceTracker{
		presences: make(map[uuid.UUID]*PresenceInfo),
		log:       log,
	}
}

// OnUserConnected handles user connection event
func (pt *PresenceTracker) OnUserConnected(userID uuid.UUID) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	presence, exists := pt.presences[userID]
	if !exists {
		presence = &PresenceInfo{
			UserID: userID,
			Status: StatusOnline,
		}
		pt.presences[userID] = presence
	}

	presence.DeviceCount++
	presence.Status = StatusOnline
	presence.LastSeenAt = time.Now()

	pt.log.Debug("User presence updated (connected)",
		logger.String("user_id", userID.String()),
		logger.Int("device_count", presence.DeviceCount),
	)
}

// OnUserDisconnected handles user disconnection event
func (pt *PresenceTracker) OnUserDisconnected(userID uuid.UUID) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	presence, exists := pt.presences[userID]
	if !exists {
		return
	}

	presence.DeviceCount--
	if presence.DeviceCount <= 0 {
		presence.DeviceCount = 0
		presence.Status = StatusOffline
		presence.LastSeenAt = time.Now()
	}

	pt.log.Debug("User presence updated (disconnected)",
		logger.String("user_id", userID.String()),
		logger.Int("device_count", presence.DeviceCount),
	)
}

// UpdatePresence updates user presence status
func (pt *PresenceTracker) UpdatePresence(userID uuid.UUID, status PresenceStatus, customStatus string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	presence, exists := pt.presences[userID]
	if !exists {
		presence = &PresenceInfo{
			UserID: userID,
		}
		pt.presences[userID] = presence
	}

	presence.Status = status
	presence.CustomStatus = customStatus
	presence.LastSeenAt = time.Now()
}

// GetPresence returns user presence
func (pt *PresenceTracker) GetPresence(userID uuid.UUID) *PresenceInfo {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	if presence, exists := pt.presences[userID]; exists {
		return presence
	}

	return &PresenceInfo{
		UserID: userID,
		Status: StatusOffline,
	}
}

// GetBulkPresence returns presence for multiple users
func (pt *PresenceTracker) GetBulkPresence(userIDs []uuid.UUID) map[uuid.UUID]*PresenceInfo {
	result := make(map[uuid.UUID]*PresenceInfo)

	for _, userID := range userIDs {
		result[userID] = pt.GetPresence(userID)
	}

	return result
}
