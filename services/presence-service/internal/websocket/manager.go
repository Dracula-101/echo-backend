package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"presence-service/internal/model"
	"presence-service/internal/repo"
	"time"

	"shared/pkg/cache"
	"shared/pkg/logger"

	"github.com/google/uuid"
)

// Manager is the central WebSocket manager that handles all real-time presence operations
// It combines the Hub (connection management) with Service (business logic)
type Manager struct {
	hub   *Hub
	repo  repo.PresenceRepository
	cache cache.Cache
	log   logger.Logger
}

// NewManager creates a new centralized WebSocket manager
func NewManager(repo repo.PresenceRepository, cache cache.Cache, log logger.Logger) *Manager {
	hub := NewHub(log)

	m := &Manager{
		hub:   hub,
		repo:  repo,
		cache: cache,
		log:   log,
	}

	// Set up disconnect callback
	hub.onDisconnect = func(client *Client) {
		ctx := context.Background()
		if err := m.HandleClientDisconnect(ctx, client); err != nil {
			log.Error("Failed to handle client disconnect",
				logger.String("client_id", client.ID),
				logger.String("user_id", client.UserID.String()),
				logger.Error(err),
			)
		}
	}

	return m
}

// Start starts the WebSocket hub
func (m *Manager) Start() {
	go m.hub.Run()
	m.log.Info("WebSocket Manager started")
}

// Shutdown gracefully shuts down the manager
func (m *Manager) Shutdown() {
	m.hub.Shutdown()
	m.log.Info("WebSocket Manager shutdown complete")
}

// GetHub returns the underlying hub (needed for handler)
func (m *Manager) GetHub() *Hub {
	return m.hub
}

// ValidateUserExists checks if a user exists in the database with caching
func (m *Manager) ValidateUserExists(ctx context.Context, userID uuid.UUID) (bool, error) {
	cacheKey := fmt.Sprintf("user:exists:%s", userID.String())

	existsInCache, cacheErr := m.cache.GetBool(ctx, cacheKey)
	if cacheErr == nil {
		m.log.Debug("User existence check (cached)",
			logger.String("user_id", userID.String()),
			logger.Bool("exists", existsInCache),
		)
		return existsInCache, nil
	}

	// Cache miss - check database
	exists, err := m.repo.UserExists(ctx, userID)
	if err != nil {
		m.log.Error("Failed to validate user existence",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return false, err
	}
	cacheTTL := 5 * time.Minute
	if !exists {
		cacheTTL = 30 * time.Second
	}

	if err := m.cache.SetBool(ctx, cacheKey, exists, cacheTTL); err != nil {
		m.log.Warn("Failed to cache user existence",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
	}

	m.log.Debug("User existence check (database)",
		logger.String("user_id", userID.String()),
		logger.Bool("exists", exists),
	)

	return exists, nil
}

// ===========================
// Connection Lifecycle
// ===========================

// HandleClientConnect handles new client connection
func (m *Manager) HandleClientConnect(ctx context.Context, client *Client) error {
	// Set user online in database
	update := &model.PresenceUpdate{
		UserID:       client.UserID,
		DeviceID:     client.DeviceID,
		OnlineStatus: "online",
	}

	if err := m.repo.UpdatePresence(ctx, update); err != nil {
		m.log.Error("Failed to set user online",
			logger.String("user_id", client.UserID.String()),
			logger.Error(err),
		)
		return err
	}

	// Invalidate cache
	m.invalidatePresenceCache(ctx, client.UserID)

	// Notify contacts about user coming online
	recipients, _ := m.getPresenceRecipients(ctx, client.UserID)
	m.hub.BroadcastPresenceUpdate(&HubPresenceUpdate{
		UserID:       client.UserID,
		OnlineStatus: "online",
		CustomStatus: "",
		BroadcastTo:  recipients,
	})

	m.log.Info("Client connected and user set online",
		logger.String("user_id", client.UserID.String()),
		logger.String("device_id", client.DeviceID),
	)

	return nil
}

// HandleClientDisconnect handles client disconnection
func (m *Manager) HandleClientDisconnect(ctx context.Context, client *Client) error {
	if !m.hub.IsUserOnline(client.UserID) {
		update := &model.PresenceUpdate{
			UserID:       client.UserID,
			DeviceID:     client.DeviceID,
			OnlineStatus: "offline",
		}

		if err := m.repo.UpdatePresence(ctx, update); err != nil {
			m.log.Error("Failed to set user offline",
				logger.String("user_id", client.UserID.String()),
				logger.Error(err),
			)
			return err
		}

		m.invalidatePresenceCache(ctx, client.UserID)

		recipients, _ := m.getPresenceRecipients(ctx, client.UserID)
		m.hub.BroadcastPresenceUpdate(&HubPresenceUpdate{
			UserID:       client.UserID,
			OnlineStatus: "offline",
			CustomStatus: "",
			BroadcastTo:  recipients,
		})
	}

	m.log.Info("Client disconnected",
		logger.String("user_id", client.UserID.String()),
		logger.String("device_id", client.DeviceID),
		logger.Bool("user_still_online", m.hub.IsUserOnline(client.UserID)),
	)

	return nil
}

// ===========================
// Message Handlers
// ===========================

// HandlePresenceUpdate processes presence updates from WebSocket clients
func (m *Manager) HandlePresenceUpdate(ctx context.Context, client *Client, status, customStatus string) error {
	// Validate status
	validStatuses := map[string]bool{
		"online": true, "offline": true, "away": true, "busy": true, "invisible": true,
	}
	if !validStatuses[status] {
		return fmt.Errorf("invalid status: %s", status)
	}

	// Update in database
	update := &model.PresenceUpdate{
		UserID:       client.UserID,
		OnlineStatus: status,
		CustomStatus: customStatus,
		DeviceID:     client.DeviceID,
	}

	if err := m.repo.UpdatePresence(ctx, update); err != nil {
		m.log.Error("Failed to update presence in database",
			logger.String("user_id", client.UserID.String()),
			logger.Error(err),
		)
		return err
	}

	// Invalidate cache
	m.invalidatePresenceCache(ctx, client.UserID)

	// Get recipients and broadcast
	recipients, _ := m.getPresenceRecipients(ctx, client.UserID)
	m.hub.BroadcastPresenceUpdate(&HubPresenceUpdate{
		UserID:       client.UserID,
		OnlineStatus: status,
		CustomStatus: customStatus,
		BroadcastTo:  recipients,
	})

	m.log.Debug("Presence updated via WebSocket",
		logger.String("user_id", client.UserID.String()),
		logger.String("status", status),
		logger.Int("recipients", len(recipients)),
	)

	return nil
}

// HandleHeartbeat processes heartbeat from WebSocket clients
func (m *Manager) HandleHeartbeat(ctx context.Context, client *Client) error {
	if err := m.repo.UpdateHeartbeat(ctx, client.UserID, client.DeviceID); err != nil {
		m.log.Error("Failed to update heartbeat",
			logger.String("user_id", client.UserID.String()),
			logger.Error(err),
		)
		return err
	}

	return nil
}

func (m *Manager) HandleTypingIndicator(ctx context.Context, client *Client, conversationID uuid.UUID, isTyping bool) error {
	indicator := &model.TypingIndicator{
		ConversationID: conversationID,
		UserID:         client.UserID,
		DeviceID:       client.DeviceID,
		IsTyping:       isTyping,
		UpdatedAt:      time.Now(),
	}

	if err := m.repo.SetTypingIndicator(ctx, indicator); err != nil {
		m.log.Error("Failed to store typing indicator",
			logger.String("user_id", client.UserID.String()),
			logger.String("conversation_id", conversationID.String()),
			logger.Error(err),
		)
		return err
	}

	// Get conversation participants and broadcast
	participants, _ := m.getConversationParticipants(ctx, conversationID)
	m.hub.BroadcastTypingIndicator(&HubTypingBroadcast{
		ConversationID: conversationID,
		UserID:         client.UserID,
		IsTyping:       isTyping,
		Participants:   participants,
	})

	m.log.Debug("Typing indicator sent via WebSocket",
		logger.String("user_id", client.UserID.String()),
		logger.String("conversation_id", conversationID.String()),
		logger.Bool("is_typing", isTyping),
	)

	return nil
}

// ===========================
// Query Methods (HTTP/WS)
// ===========================

// GetPresence retrieves user presence (enhanced with real-time data)
func (m *Manager) GetPresence(ctx context.Context, userID uuid.UUID, requesterID uuid.UUID) (*model.UserPresence, error) {
	// Get from database
	presence, err := m.repo.GetPresence(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Apply privacy filters
	privacy, err := m.repo.GetPrivacySettings(ctx, userID)
	if err != nil {
		m.log.Warn("Failed to get privacy settings", logger.Error(err))
	} else {
		presence = m.applyPrivacyFilters(presence, privacy, requesterID, userID)
	}

	// Enhance with real-time data from hub
	if m.hub.IsUserOnline(userID) {
		presence.OnlineStatus = "online"
		now := time.Now()
		presence.LastSeenAt = &now
	}

	return presence, nil
}

// GetBulkPresence retrieves presence for multiple users
func (m *Manager) GetBulkPresence(ctx context.Context, userIDs []uuid.UUID, requesterID uuid.UUID) (map[uuid.UUID]*model.UserPresence, error) {
	result := make(map[uuid.UUID]*model.UserPresence)

	presenceList, err := m.repo.GetBulkPresence(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	for _, presence := range presenceList {
		// Apply privacy filters
		privacy, err := m.repo.GetPrivacySettings(ctx, presence.UserID)
		if err == nil {
			presence = m.applyPrivacyFilters(presence, privacy, requesterID, presence.UserID)
		}

		// Enhance with real-time data
		if m.hub.IsUserOnline(presence.UserID) {
			presence.OnlineStatus = "online"
			now := time.Now()
			presence.LastSeenAt = &now
		}

		result[presence.UserID] = presence
	}

	return result, nil
}

// IsUserOnline checks if user is online via WebSocket
func (m *Manager) IsUserOnline(userID uuid.UUID) bool {
	return m.hub.IsUserOnline(userID)
}

// GetOnlineUsers returns all currently online user IDs
func (m *Manager) GetOnlineUsers() []uuid.UUID {
	return m.hub.GetOnlineUsers()
}

// GetActiveDevices returns active devices for a user
func (m *Manager) GetActiveDevices(ctx context.Context, userID uuid.UUID) ([]*model.Device, error) {
	// Get real-time devices from hub
	hubDevices := m.hub.GetActiveDevices(userID)
	if len(hubDevices) > 0 {
		return hubDevices, nil
	}

	// Fallback to database
	return m.repo.GetActiveDevices(ctx, userID)
}

// GetTypingIndicators retrieves typing indicators for a conversation
func (m *Manager) GetTypingIndicators(ctx context.Context, conversationID uuid.UUID) ([]*model.TypingIndicator, error) {
	return m.repo.GetTypingIndicators(ctx, conversationID)
}

// ===========================
// Event Broadcasting (Inter-service)
// ===========================

// BroadcastEvent broadcasts generic real-time events
func (m *Manager) BroadcastEvent(ctx context.Context, event *model.RealtimeEvent) error {
	switch event.Category {
	case model.CategoryPresence:
		return m.handlePresenceEvent(event)
	case model.CategoryTyping:
		return m.handleTypingEvent(event)
	default:
		m.log.Warn("Unsupported event category for broadcasting",
			logger.String("category", string(event.Category)),
		)
		return fmt.Errorf("unsupported event category: %s", event.Category)
	}
}

// ===========================
// Helper Methods
// ===========================

func (m *Manager) handlePresenceEvent(event *model.RealtimeEvent) error {
	payload, ok := event.Payload.(map[string]interface{})
	if !ok {
		payloadBytes, _ := json.Marshal(event.Payload)
		json.Unmarshal(payloadBytes, &payload)
	}

	m.hub.BroadcastPresenceUpdate(&HubPresenceUpdate{
		UserID:       event.Recipients[0],
		OnlineStatus: getStringFromMap(payload, "online_status"),
		CustomStatus: getStringFromMap(payload, "custom_status"),
		BroadcastTo:  event.Recipients,
	})

	return nil
}

func (m *Manager) handleTypingEvent(event *model.RealtimeEvent) error {
	payload, ok := event.Payload.(map[string]interface{})
	if !ok {
		payloadBytes, _ := json.Marshal(event.Payload)
		json.Unmarshal(payloadBytes, &payload)
	}

	conversationID, _ := uuid.Parse(getStringFromMap(payload, "conversation_id"))
	userID, _ := uuid.Parse(getStringFromMap(payload, "user_id"))
	isTyping, _ := payload["is_typing"].(bool)

	m.hub.BroadcastTypingIndicator(&HubTypingBroadcast{
		ConversationID: conversationID,
		UserID:         userID,
		IsTyping:       isTyping,
		Participants:   event.Recipients,
	})

	return nil
}

func (m *Manager) applyPrivacyFilters(presence *model.UserPresence, privacy *model.PresencePrivacy, requesterID, targetUserID uuid.UUID) *model.UserPresence {
	if requesterID == targetUserID {
		return presence
	}

	if privacy != nil && privacy.OnlineStatusVisibility == "nobody" {
		presence.OnlineStatus = "offline"
	}

	if privacy != nil && privacy.LastSeenVisibility == "nobody" {
		presence.LastSeenAt = nil
	}

	return presence
}

func (m *Manager) invalidatePresenceCache(ctx context.Context, userID uuid.UUID) {
	if m.cache != nil {
		cacheKey := fmt.Sprintf("presence:%s", userID.String())
		_ = m.cache.Delete(ctx, cacheKey)
	}
}

func (m *Manager) getPresenceRecipients(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	// TODO: Implement logic to get user's contacts/friends
	// For now, return empty slice
	// In production, this should query the user's contacts/friends from the database
	return []uuid.UUID{}, nil
}

func (m *Manager) getConversationParticipants(ctx context.Context, conversationID uuid.UUID) ([]uuid.UUID, error) {
	// TODO: Implement logic to get conversation participants
	// This might require calling the message service or querying a conversations table
	return []uuid.UUID{}, nil
}

func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
