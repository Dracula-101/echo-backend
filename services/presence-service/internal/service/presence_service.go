package service

import (
	"context"
	"fmt"
	"presence-service/internal/model"
	"presence-service/internal/repo"
	"presence-service/internal/websocket"

	"shared/pkg/cache"
	"shared/pkg/logger"

	"github.com/google/uuid"
)

// WebSocketHub interface for broadcasting (to avoid circular dependency)
// This matches the hub in internal/websocket package
type WebSocketHub interface {
	IsUserOnline(userID uuid.UUID) bool
	GetOnlineUsers() []uuid.UUID
	BroadcastPresenceUpdate(update *websocket.HubPresenceUpdate)
	BroadcastTypingIndicator(typing *websocket.HubTypingBroadcast)
}

type PresenceService interface {
	// Presence management
	UpdatePresence(ctx context.Context, update *model.PresenceUpdate) (*model.UserPresence, error)
	GetPresence(ctx context.Context, userID uuid.UUID, requesterID uuid.UUID) (*model.UserPresence, error)
	GetBulkPresence(ctx context.Context, userIDs []uuid.UUID, requesterID uuid.UUID) (map[uuid.UUID]*model.UserPresence, error)
	Heartbeat(ctx context.Context, userID uuid.UUID, deviceID string) error
	GetActiveDevices(ctx context.Context, userID uuid.UUID) ([]*model.Device, error)

	// Typing indicators
	SetTypingIndicator(ctx context.Context, indicator *model.TypingIndicator) error
	GetTypingIndicators(ctx context.Context, conversationID uuid.UUID) ([]*model.TypingIndicator, error)

	// Centralized event broadcasting (for other services)
	BroadcastEvent(ctx context.Context, event *model.RealtimeEvent) error
	IsUserOnline(ctx context.Context, userID uuid.UUID) (bool, error)
	GetOnlineUsers(ctx context.Context) ([]uuid.UUID, error)
}

type presenceService struct {
	repo  repo.PresenceRepository
	cache cache.Cache
	hub   WebSocketHub
	log   logger.Logger
}

func NewPresenceService(repo repo.PresenceRepository, cache cache.Cache, hub WebSocketHub, log logger.Logger) PresenceService {
	return &presenceService{
		repo:  repo,
		cache: cache,
		hub:   hub,
		log:   log,
	}
}

func (s *presenceService) UpdatePresence(ctx context.Context, update *model.PresenceUpdate) (*model.UserPresence, error) {
	validStatuses := map[string]bool{
		"online": true, "offline": true, "away": true, "busy": true, "invisible": true,
	}
	if !validStatuses[update.OnlineStatus] {
		return nil, fmt.Errorf("invalid status: %s", update.OnlineStatus)
	}

	if err := s.repo.UpdatePresence(ctx, update); err != nil {
		return nil, err
	}

	presence, err := s.repo.GetPresence(ctx, update.UserID)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		cacheKey := fmt.Sprintf("presence:%s", update.UserID.String())
		_ = s.cache.Delete(ctx, cacheKey)
	}

	return presence, nil
}

func (s *presenceService) GetPresence(ctx context.Context, userID uuid.UUID, requesterID uuid.UUID) (*model.UserPresence, error) {
	presence, err := s.repo.GetPresence(ctx, userID)
	if err != nil {
		return nil, err
	}

	privacy, err := s.repo.GetPrivacySettings(ctx, userID)
	if err != nil {
		s.log.Warn("Failed to get privacy settings", logger.Error(err))
	} else {
		presence = s.applyPrivacyFilters(presence, privacy, requesterID, userID)
	}

	return presence, nil
}

func (s *presenceService) GetBulkPresence(ctx context.Context, userIDs []uuid.UUID, requesterID uuid.UUID) (map[uuid.UUID]*model.UserPresence, error) {
	presences, err := s.repo.GetBulkPresence(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	for userID, presence := range presences {
		privacy, err := s.repo.GetPrivacySettings(ctx, userID)
		if err != nil {
			s.log.Warn("Failed to get privacy settings for user",
				logger.String("user_id", userID.String()),
				logger.Error(err),
			)
			continue
		}
		presences[userID] = s.applyPrivacyFilters(presence, privacy, requesterID, userID)
	}

	return presences, nil
}

func (s *presenceService) Heartbeat(ctx context.Context, userID uuid.UUID, deviceID string) error {
	if err := s.repo.UpdateHeartbeat(ctx, userID, deviceID); err != nil {
		return err
	}

	if s.cache != nil {
		cacheKey := fmt.Sprintf("presence:%s", userID.String())
		_ = s.cache.Delete(ctx, cacheKey)
	}

	return nil
}

func (s *presenceService) GetActiveDevices(ctx context.Context, userID uuid.UUID) ([]*model.Device, error) {
	return s.repo.GetActiveDevices(ctx, userID)
}

func (s *presenceService) SetTypingIndicator(ctx context.Context, indicator *model.TypingIndicator) error {
	if s.cache != nil {
		cacheKey := fmt.Sprintf("typing:%s:%s", indicator.ConversationID.String(), indicator.UserID.String())
		if indicator.IsTyping {
			_ = s.cache.Set(ctx, cacheKey, []byte("1"), 10)
		} else {
			_ = s.cache.Delete(ctx, cacheKey)
		}
	}

	return s.repo.SetTypingIndicator(ctx, indicator)
}

func (s *presenceService) GetTypingIndicators(ctx context.Context, conversationID uuid.UUID) ([]*model.TypingIndicator, error) {
	return s.repo.GetTypingIndicators(ctx, conversationID)
}

func (s *presenceService) applyPrivacyFilters(
	presence *model.UserPresence,
	privacy *model.PresencePrivacy,
	requesterID uuid.UUID,
	targetUserID uuid.UUID,
) *model.UserPresence {
	if requesterID == targetUserID {
		return presence
	}

	filtered := *presence

	switch privacy.LastSeenVisibility {
	case "nobody":
		filtered.LastSeenAt = nil
	case "contacts":
	}

	switch privacy.OnlineStatusVisibility {
	case "nobody":
		filtered.OnlineStatus = "offline"
	case "contacts":
	}

	return &filtered
}

// BroadcastEvent broadcasts a real-time event to connected users via WebSocket
func (s *presenceService) BroadcastEvent(ctx context.Context, event *model.RealtimeEvent) error {
	s.log.Debug("Broadcasting event",
		logger.String("event_type", string(event.Type)),
		logger.String("category", string(event.Category)),
		logger.Int("recipients", len(event.Recipients)),
	)

	// Broadcast based on event category
	switch event.Category {
	case model.CategoryPresence:
		// Handle presence events
		if payload, ok := event.Payload.(map[string]interface{}); ok {
			s.hub.BroadcastPresenceUpdate(&websocket.HubPresenceUpdate{
				UserID:       event.Recipients[0], // TODO: handle multiple recipients properly
				OnlineStatus: payload["online_status"].(string),
				CustomStatus: getStringFromMap(payload, "custom_status"),
				BroadcastTo:  event.Recipients,
			})
		}

	case model.CategoryTyping:
		// Handle typing events
		if payload, ok := event.Payload.(map[string]interface{}); ok {
			conversationID, _ := uuid.Parse(payload["conversation_id"].(string))
			userID, _ := uuid.Parse(payload["user_id"].(string))
			s.hub.BroadcastTypingIndicator(&websocket.HubTypingBroadcast{
				ConversationID: conversationID,
				UserID:         userID,
				IsTyping:       payload["is_typing"].(bool),
				Participants:   event.Recipients,
			})
		}

	default:
		// For other event types, log for now
		// TODO: Implement generic broadcast in hub
		s.log.Info("Broadcasting generic event",
			logger.String("event_type", string(event.Type)),
		)
	}

	return nil
}

func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// IsUserOnline checks if a user has any active WebSocket connections
func (s *presenceService) IsUserOnline(ctx context.Context, userID uuid.UUID) (bool, error) {
	if s.hub == nil {
		return false, fmt.Errorf("WebSocket hub not initialized")
	}

	isOnline := s.hub.IsUserOnline(userID)

	s.log.Debug("Checked user online status",
		logger.String("user_id", userID.String()),
		logger.Bool("is_online", isOnline),
	)

	return isOnline, nil
}

// GetOnlineUsers returns list of all currently online users
func (s *presenceService) GetOnlineUsers(ctx context.Context) ([]uuid.UUID, error) {
	if s.hub == nil {
		return nil, fmt.Errorf("WebSocket hub not initialized")
	}

	onlineUsers := s.hub.GetOnlineUsers()

	s.log.Debug("Retrieved online users",
		logger.Int("count", len(onlineUsers)),
	)

	return onlineUsers, nil
}
