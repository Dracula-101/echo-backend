package service

import (
	"context"
	"fmt"
	"time"
	"ws-service/internal/model"

	"shared/pkg/cache"
	"shared/pkg/database"
	"shared/pkg/logger"
	"shared/server/websocket/hub"

	"github.com/google/uuid"
)

type WSService interface {
	// User validation
	ValidateUserExists(ctx context.Context, userID uuid.UUID) (bool, error)

	// Connection lifecycle
	HandleClientConnect(ctx context.Context, userID uuid.UUID, deviceID string) error
	HandleClientDisconnect(ctx context.Context, userID uuid.UUID, deviceID string) error

	// Broadcasting
	BroadcastEvent(ctx context.Context, req *model.BroadcastRequest) (*model.BroadcastResponse, error)

	// Presence queries
	IsUserOnline(ctx context.Context, userID uuid.UUID) (bool, error)
	GetOnlineUsers(ctx context.Context) ([]uuid.UUID, error)

	// Statistics
	GetStats(ctx context.Context) (*model.StatsResponse, error)
}

type wsService struct {
	db    database.Database
	cache cache.Cache
	hub   *hub.Hub
	log   logger.Logger
}

func NewWSService(db database.Database, cache cache.Cache, h *hub.Hub, log logger.Logger) WSService {
	return &wsService{
		db:    db,
		cache: cache,
		hub:   h,
		log:   log,
	}
}

// ValidateUserExists checks if a user exists in the database with caching
func (s *wsService) ValidateUserExists(ctx context.Context, userID uuid.UUID) (bool, error) {
	if s.cache == nil {
		return s.checkUserExistsInDB(ctx, userID)
	}

	cacheKey := fmt.Sprintf("user:exists:%s", userID.String())

	existsInCache, cacheErr := s.cache.GetBool(ctx, cacheKey)
	if cacheErr == nil {
		s.log.Debug("User existence check (cached)",
			logger.String("user_id", userID.String()),
			logger.Bool("exists", existsInCache),
		)
		return existsInCache, nil
	}

	// Cache miss - check database
	exists, err := s.checkUserExistsInDB(ctx, userID)
	if err != nil {
		return false, err
	}

	// Cache the result
	cacheTTL := 5 * time.Minute
	if !exists {
		cacheTTL = 30 * time.Second
	}

	if err := s.cache.SetBool(ctx, cacheKey, exists, cacheTTL); err != nil {
		s.log.Warn("Failed to cache user existence",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
	}

	return exists, nil
}

func (s *wsService) checkUserExistsInDB(ctx context.Context, userID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM users.profiles
			WHERE user_id = $1
		)
	`

	var exists bool
	err := s.db.QueryRow(ctx, query, userID).Scan(&exists)
	if err != nil {
		s.log.Error("Failed to check if user exists",
			logger.Error(err),
			logger.String("user_id", userID.String()),
		)
		return false, err
	}

	s.log.Debug("User existence check (database)",
		logger.String("user_id", userID.String()),
		logger.Bool("exists", exists),
	)

	query = `
		SELECT EXISTS(
			SELECT 1
			FROM auth.sessions
			WHERE user_id = $1
			AND revoked_at IS NULL
		)
	`
	err = s.db.QueryRow(ctx, query, userID).Scan(&exists)
	if err != nil {
		s.log.Error("Failed to check if user has active sessions",
			logger.Error(err),
			logger.String("user_id", userID.String()),
		)
		return false, err
	}

	return exists, nil
}

// HandleClientConnect handles new client connection
func (s *wsService) HandleClientConnect(ctx context.Context, userID uuid.UUID, deviceID string) error {
	s.log.Info("Client connected",
		logger.String("user_id", userID.String()),
		logger.String("device_id", deviceID),
	)

	// You can add any additional logic here, such as:
	// - Updating user's last seen
	// - Sending presence notifications to contacts
	// - Logging connection events

	return nil
}

// HandleClientDisconnect handles client disconnection
func (s *wsService) HandleClientDisconnect(ctx context.Context, userID uuid.UUID, deviceID string) error {
	s.log.Info("Client disconnected",
		logger.String("user_id", userID.String()),
		logger.String("device_id", deviceID),
		logger.Bool("user_still_online", s.hub.IsOnline(userID)),
	)

	// You can add any additional logic here, such as:
	// - Updating user's online status if no more devices
	// - Sending offline notifications to contacts

	return nil
}

// BroadcastEvent broadcasts an event to specified recipients
func (s *wsService) BroadcastEvent(ctx context.Context, req *model.BroadcastRequest) (*model.BroadcastResponse, error) {
	// Validate request
	if len(req.Recipients) == 0 {
		return nil, fmt.Errorf("no recipients specified")
	}

	// Create event
	event := &model.RealtimeEvent{
		ID:         uuid.New(),
		Type:       req.EventType,
		Category:   s.getEventCategory(req.EventType),
		Timestamp:  time.Now(),
		Recipients: req.Recipients,
		Sender:     req.Sender,
		Payload:    req.Payload,
		Priority:   req.Priority,
		TTL:        req.TTL,
	}

	// Broadcast to recipients via hub
	onlineCount := 0
	for _, recipientID := range req.Recipients {
		if s.hub.IsOnline(recipientID) {
			// Marshal event to JSON
			data, err := s.marshalEvent(event)
			if err != nil {
				s.log.Error("Failed to marshal event",
					logger.String("event_id", event.ID.String()),
					logger.Error(err),
				)
				continue
			}

			// Broadcast to all user's devices
			if err := s.hub.Broadcast(recipientID, data); err != nil {
				s.log.Warn("Failed to broadcast to user",
					logger.String("user_id", recipientID.String()),
					logger.Error(err),
				)
			} else {
				onlineCount++
			}
		}
	}

	s.log.Info("Event broadcasted",
		logger.String("event_id", event.ID.String()),
		logger.String("event_type", string(event.Type)),
		logger.Int("recipients", len(req.Recipients)),
		logger.Int("online_recipients", onlineCount),
	)

	return &model.BroadcastResponse{
		EventID:          event.ID,
		Recipients:       len(req.Recipients),
		OnlineRecipients: onlineCount,
		Timestamp:        event.Timestamp,
	}, nil
}

// marshalEvent marshals an event to JSON bytes
func (s *wsService) marshalEvent(event *model.RealtimeEvent) ([]byte, error) {
	// You can use encoding/json or your preferred JSON library
	// For now, using a simple approach
	return []byte(fmt.Sprintf(`{"id":"%s","type":"%s","payload":%v}`,
		event.ID.String(), event.Type, event.Payload)), nil
}

// IsUserOnline checks if a user has any active WebSocket connections
func (s *wsService) IsUserOnline(ctx context.Context, userID uuid.UUID) (bool, error) {
	isOnline := s.hub.IsOnline(userID)

	s.log.Debug("Checked user online status",
		logger.String("user_id", userID.String()),
		logger.Bool("is_online", isOnline),
	)

	return isOnline, nil
}

// GetOnlineUsers returns list of all currently online users
func (s *wsService) GetOnlineUsers(ctx context.Context) ([]uuid.UUID, error) {
	clients := s.hub.GetAllClients()
	onlineUsers := make([]uuid.UUID, 0, len(clients))

	for _, client := range clients {
		onlineUsers = append(onlineUsers, client.UserID)
	}

	s.log.Debug("Retrieved online users",
		logger.Int("count", len(onlineUsers)),
	)

	return onlineUsers, nil
}

// GetStats returns WebSocket hub statistics
func (s *wsService) GetStats(ctx context.Context) (*model.StatsResponse, error) {
	stats := &model.StatsResponse{
		TotalUsers:   s.hub.ClientCount(),
		TotalDevices: s.hub.ConnectionCount(),
	}

	s.log.Debug("Retrieved hub stats",
		logger.Int("total_users", stats.TotalUsers),
		logger.Int("total_devices", stats.TotalDevices),
	)

	return stats, nil
}

// getEventCategory determines the category from event type
func (s *wsService) getEventCategory(eventType model.EventType) model.EventCategory {
	switch {
	case eventType >= model.EventPresenceOnline && eventType <= model.EventPresenceUpdate:
		return model.CategoryPresence
	case eventType >= model.EventMessageNew && eventType <= model.EventMessageDeleted:
		return model.CategoryMessaging
	case eventType >= model.EventTypingStart && eventType <= model.EventTypingStop:
		return model.CategoryTyping
	case eventType >= model.EventCallIncoming && eventType <= model.EventCallMissed:
		return model.CategoryCall
	case eventType >= model.EventNotificationNew && eventType <= model.EventNotificationRead:
		return model.CategoryNotification
	case eventType >= model.EventUserProfileUpdated && eventType <= model.EventUserUnblocked:
		return model.CategoryUser
	case eventType >= model.EventSystemMaintenance && eventType <= model.EventSystemAnnouncement:
		return model.CategorySystem
	default:
		return model.CategorySystem
	}
}
