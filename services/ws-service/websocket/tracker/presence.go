package tracker

import (
	"sync"
	"time"

	"shared/pkg/logger"

	"github.com/google/uuid"
)

type PresenceStatus string

const (
	StatusOnline    PresenceStatus = "online"
	StatusOffline   PresenceStatus = "offline"
	StatusAway      PresenceStatus = "away"
	StatusBusy      PresenceStatus = "busy"
	StatusInvisible PresenceStatus = "invisible"
	StatusDND       PresenceStatus = "dnd"
)

func (ps PresenceStatus) IsValid() bool {
	switch ps {
	case StatusOnline, StatusOffline, StatusAway, StatusBusy, StatusInvisible, StatusDND:
		return true
	default:
		return false
	}
}

func (ps PresenceStatus) String() string {
	return string(ps)
}

type PresenceInfo struct {
	UserID        uuid.UUID              `json:"user_id"`
	Status        PresenceStatus         `json:"status"`
	CustomStatus  string                 `json:"custom_status,omitempty"`
	StatusMessage string                 `json:"status_message,omitempty"`
	LastSeenAt    time.Time              `json:"last_seen_at"`
	LastActiveAt  time.Time              `json:"last_active_at,omitempty"`
	DeviceCount   int                    `json:"device_count"`
	Devices       []DevicePresence       `json:"devices,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

type DevicePresence struct {
	DeviceID   string         `json:"device_id"`
	Platform   string         `json:"platform"`
	Status     PresenceStatus `json:"status"`
	LastSeenAt time.Time      `json:"last_seen_at"`
}

type PresenceChangeEvent struct {
	UserID       uuid.UUID      `json:"user_id"`
	OldStatus    PresenceStatus `json:"old_status"`
	NewStatus    PresenceStatus `json:"new_status"`
	CustomStatus string         `json:"custom_status,omitempty"`
	ChangedAt    time.Time      `json:"changed_at"`
}

type PresenceChangeCallback func(event PresenceChangeEvent)

type PresenceTracker struct {
	presences       map[uuid.UUID]*PresenceInfo
	devicePresences map[uuid.UUID]map[string]*DevicePresence
	callbacks       []PresenceChangeCallback
	idleTimeout     time.Duration
	awayTimeout     time.Duration
	mu              sync.RWMutex
	log             logger.Logger
}

type PresenceTrackerConfig struct {
	IdleTimeout time.Duration
	AwayTimeout time.Duration
}

func NewPresenceTracker(log logger.Logger, cfg *PresenceTrackerConfig) *PresenceTracker {
	idleTimeout := 5 * time.Minute
	awayTimeout := 15 * time.Minute

	if cfg != nil {
		if cfg.IdleTimeout > 0 {
			idleTimeout = cfg.IdleTimeout
		}
		if cfg.AwayTimeout > 0 {
			awayTimeout = cfg.AwayTimeout
		}
	}

	return &PresenceTracker{
		presences:       make(map[uuid.UUID]*PresenceInfo),
		devicePresences: make(map[uuid.UUID]map[string]*DevicePresence),
		callbacks:       make([]PresenceChangeCallback, 0),
		idleTimeout:     idleTimeout,
		awayTimeout:     awayTimeout,
		log:             log,
	}
}

func (pt *PresenceTracker) OnChange(callback PresenceChangeCallback) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.callbacks = append(pt.callbacks, callback)
}

func (pt *PresenceTracker) notifyChange(event PresenceChangeEvent) {
	for _, cb := range pt.callbacks {
		go cb(event)
	}
}

func (pt *PresenceTracker) OnUserConnected(userID uuid.UUID, deviceID, platform string) PresenceChangeEvent {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	now := time.Now()
	presence, exists := pt.presences[userID]
	oldStatus := StatusOffline

	if exists {
		oldStatus = presence.Status
	} else {
		presence = &PresenceInfo{
			UserID:   userID,
			Metadata: make(map[string]interface{}),
		}
		pt.presences[userID] = presence
	}

	if pt.devicePresences[userID] == nil {
		pt.devicePresences[userID] = make(map[string]*DevicePresence)
	}

	pt.devicePresences[userID][deviceID] = &DevicePresence{
		DeviceID:   deviceID,
		Platform:   platform,
		Status:     StatusOnline,
		LastSeenAt: now,
	}

	presence.DeviceCount = len(pt.devicePresences[userID])
	presence.Status = StatusOnline
	presence.LastSeenAt = now
	presence.LastActiveAt = now

	devices := make([]DevicePresence, 0, len(pt.devicePresences[userID]))
	for _, d := range pt.devicePresences[userID] {
		devices = append(devices, *d)
	}
	presence.Devices = devices

	pt.log.Debug("User presence updated (connected)",
		logger.String("user_id", userID.String()),
		logger.String("device_id", deviceID),
		logger.Int("device_count", presence.DeviceCount),
	)

	event := PresenceChangeEvent{
		UserID:    userID,
		OldStatus: oldStatus,
		NewStatus: StatusOnline,
		ChangedAt: now,
	}

	if oldStatus != StatusOnline {
		pt.notifyChange(event)
	}

	return event
}

func (pt *PresenceTracker) OnUserDisconnected(userID uuid.UUID, deviceID string) PresenceChangeEvent {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	now := time.Now()
	presence, exists := pt.presences[userID]
	if !exists {
		return PresenceChangeEvent{}
	}

	oldStatus := presence.Status

	if devices, ok := pt.devicePresences[userID]; ok {
		delete(devices, deviceID)
		presence.DeviceCount = len(devices)

		if len(devices) == 0 {
			delete(pt.devicePresences, userID)
			presence.Status = StatusOffline
			presence.LastSeenAt = now
			presence.Devices = nil
		} else {
			deviceList := make([]DevicePresence, 0, len(devices))
			for _, d := range devices {
				deviceList = append(deviceList, *d)
			}
			presence.Devices = deviceList
		}
	}

	if presence.DeviceCount == 0 {
		pt.log.Debug("User presence removed (all devices disconnected)",
			logger.String("user_id", userID.String()),
		)
	} else {
		pt.log.Debug("User presence updated (disconnected)",
			logger.String("user_id", userID.String()),
			logger.String("device_id", deviceID),
			logger.Int("device_count", presence.DeviceCount),
		)
	}

	event := PresenceChangeEvent{
		UserID:    userID,
		OldStatus: oldStatus,
		NewStatus: presence.Status,
		ChangedAt: now,
	}

	if oldStatus != presence.Status {
		pt.notifyChange(event)
	}

	return event
}

func (pt *PresenceTracker) UpdatePresence(userID uuid.UUID, status PresenceStatus, customStatus string) PresenceChangeEvent {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	now := time.Now()
	presence, exists := pt.presences[userID]
	if !exists {
		presence = &PresenceInfo{
			UserID:   userID,
			Metadata: make(map[string]interface{}),
		}
		pt.presences[userID] = presence
	}

	oldStatus := presence.Status
	presence.Status = status
	presence.CustomStatus = customStatus
	presence.LastSeenAt = now

	event := PresenceChangeEvent{
		UserID:       userID,
		OldStatus:    oldStatus,
		NewStatus:    status,
		CustomStatus: customStatus,
		ChangedAt:    now,
	}

	if oldStatus != status {
		pt.notifyChange(event)
	}

	return event
}

func (pt *PresenceTracker) UpdateActivity(userID uuid.UUID) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if presence, exists := pt.presences[userID]; exists {
		presence.LastActiveAt = time.Now()
	}
}

func (pt *PresenceTracker) GetPresence(userID uuid.UUID) *PresenceInfo {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	if presence, exists := pt.presences[userID]; exists {
		result := *presence
		return &result
	}

	return &PresenceInfo{
		UserID: userID,
		Status: StatusOffline,
	}
}

func (pt *PresenceTracker) GetBulkPresence(userIDs []uuid.UUID) map[uuid.UUID]*PresenceInfo {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	result := make(map[uuid.UUID]*PresenceInfo, len(userIDs))
	for _, userID := range userIDs {
		if presence, exists := pt.presences[userID]; exists {
			p := *presence
			result[userID] = &p
		} else {
			result[userID] = &PresenceInfo{
				UserID: userID,
				Status: StatusOffline,
			}
		}
	}

	return result
}

func (pt *PresenceTracker) IsOnline(userID uuid.UUID) bool {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	if presence, exists := pt.presences[userID]; exists {
		return presence.Status == StatusOnline
	}
	return false
}

func (pt *PresenceTracker) GetOnlineUsers() []uuid.UUID {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	users := make([]uuid.UUID, 0)
	for userID, presence := range pt.presences {
		if presence.Status == StatusOnline {
			users = append(users, userID)
		}
	}
	return users
}

func (pt *PresenceTracker) GetOnlineCount() int {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	count := 0
	for _, presence := range pt.presences {
		if presence.Status == StatusOnline {
			count++
		}
	}
	return count
}

func (pt *PresenceTracker) GetStats() PresenceStats {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	stats := PresenceStats{
		StatusCounts: make(map[PresenceStatus]int),
	}

	for _, presence := range pt.presences {
		stats.TotalUsers++
		stats.StatusCounts[presence.Status]++
		stats.TotalDevices += presence.DeviceCount
	}

	return stats
}

func (pt *PresenceTracker) CleanupStale() []PresenceChangeEvent {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	now := time.Now()
	events := make([]PresenceChangeEvent, 0)

	for userID, presence := range pt.presences {
		if presence.Status == StatusOnline {
			idleDuration := now.Sub(presence.LastActiveAt)

			if idleDuration > pt.awayTimeout {
				oldStatus := presence.Status
				presence.Status = StatusAway
				events = append(events, PresenceChangeEvent{
					UserID:    userID,
					OldStatus: oldStatus,
					NewStatus: StatusAway,
					ChangedAt: now,
				})
			}
		}
	}

	for _, event := range events {
		pt.notifyChange(event)
	}

	return events
}

type PresenceStats struct {
	TotalUsers   int                    `json:"total_users"`
	TotalDevices int                    `json:"total_devices"`
	StatusCounts map[PresenceStatus]int `json:"status_counts"`
}
