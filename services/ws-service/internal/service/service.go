package service

import (
	"context"
	"fmt"
	"shared/pkg/cache"
	"shared/pkg/database"
	"shared/pkg/logger"
	"shared/server/websocket/hub"
	"time"
	"ws-service/internal/domain"

	"github.com/google/uuid"
)

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

type WSService interface {
	// User validation
	ValidateUserExists(ctx context.Context, userID uuid.UUID) (bool, error)

	// Connection lifecycle
	HandleClientConnect(ctx context.Context, userID uuid.UUID, deviceID string) error
	HandleClientDisconnect(ctx context.Context, userID uuid.UUID, deviceID string) error

	// Broadcasting
	BroadcastEvent(ctx context.Context, req *domain.BroadcastRequest) (*domain.BroadcastResponse, error)

	// Presence queries
	IsUserOnline(ctx context.Context, userID uuid.UUID) (bool, error)
	GetOnlineUsers(ctx context.Context) ([]uuid.UUID, error)

	// Statistics
	GetStats(ctx context.Context) (*domain.StatsResponse, error)
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
func (s *wsService) BroadcastEvent(ctx context.Context, req *domain.BroadcastRequest) (*domain.BroadcastResponse, error) {
	// Validate request
	if len(req.Recipients) == 0 {
		return nil, fmt.Errorf("no recipients specified")
	}

	// Create event
	event := &domain.RealtimeEvent{
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

	return &domain.BroadcastResponse{
		EventID:          event.ID,
		Recipients:       len(req.Recipients),
		OnlineRecipients: onlineCount,
		Timestamp:        event.Timestamp,
	}, nil
}

// marshalEvent marshals an event to JSON bytes
func (s *wsService) marshalEvent(event *domain.RealtimeEvent) ([]byte, error) {
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
func (s *wsService) GetStats(ctx context.Context) (*domain.StatsResponse, error) {
	stats := &domain.StatsResponse{
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
func (s *wsService) getEventCategory(eventType domain.EventType) domain.EventCategory {
	switch {
	case eventType >= domain.EventPresenceOnline && eventType <= domain.EventPresenceUpdate:
		return domain.CategoryPresence
	case eventType >= domain.EventMessageNew && eventType <= domain.EventMessageDeleted:
		return domain.CategoryMessaging
	case eventType >= domain.EventTypingStart && eventType <= domain.EventTypingStop:
		return domain.CategoryTyping
	case eventType >= domain.EventCallIncoming && eventType <= domain.EventCallMissed:
		return domain.CategoryCall
	case eventType >= domain.EventNotificationNew && eventType <= domain.EventNotificationRead:
		return domain.CategoryNotification
	case eventType >= domain.EventUserProfileUpdated && eventType <= domain.EventUserUnblocked:
		return domain.CategoryUser
	case eventType >= domain.EventSystemMaintenance && eventType <= domain.EventSystemAnnouncement:
		return domain.CategorySystem
	default:
		return domain.CategorySystem
	}
}
