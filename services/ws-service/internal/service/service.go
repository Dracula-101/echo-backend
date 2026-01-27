package service

import (
	"context"
	"encoding/json"
	"fmt"
	"shared/pkg/cache"
	"shared/pkg/logger"
	"shared/server/websocket/hub"
	"time"
	"ws-service/internal/domain"
	"ws-service/internal/repo"

	"github.com/google/uuid"
)

type wsService struct {
	userRepo repo.UserRepository
	cache    cache.Cache
	hub      *hub.Hub
	log      logger.Logger
}

func NewWSService(userRepo repo.UserRepository, cache cache.Cache, h *hub.Hub, log logger.Logger) WSService {
	return &wsService{
		userRepo: userRepo,
		cache:    cache,
		hub:      h,
		log:      log,
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

func (s *wsService) ValidateUserExists(ctx context.Context, userID uuid.UUID) (bool, error) {
	cacheKey := fmt.Sprintf("user:exists:%s", userID.String())

	existsInCache, cacheErr := s.cache.GetBool(ctx, cacheKey)
	if cacheErr == nil {
		s.log.Debug("User existence check (cached)",
			logger.String("user_id", userID.String()),
			logger.Bool("exists", existsInCache),
		)
		return existsInCache, nil
	}

	exists, err := s.userRepo.ValidateUserAccess(ctx, userID)
	if err != nil {
		return false, err
	}

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

func (s *wsService) HandleClientConnect(ctx context.Context, userID uuid.UUID, deviceID string) error {
	s.log.Info("Client connected",
		logger.String("user_id", userID.String()),
		logger.String("device_id", deviceID),
	)

	if err := s.userRepo.UpdateLastSeen(ctx, userID); err != nil {
		s.log.Error("Failed to update last seen on connect",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
	}

	if err := s.userRepo.UpdateOnlineStatus(ctx, userID, "online"); err != nil {
		s.log.Error("Failed to update online status on connect",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
	}

	contacts, err := s.userRepo.GetUserContacts(ctx, userID)
	if err != nil {
		s.log.Error("Failed to get user contacts for presence broadcast",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return nil
	}

	if len(contacts) > 0 {
		presenceEvent := &domain.BroadcastRequest{
			EventType:  domain.EventPresenceOnline,
			Recipients: contacts,
			Sender:     &userID,
			Payload: map[string]any{
				"user_id":   userID.String(),
				"status":    "online",
				"device_id": deviceID,
			},
			Priority: 1,
		}

		if _, err := s.BroadcastEvent(ctx, presenceEvent); err != nil {
			s.log.Warn("Failed to broadcast presence online event",
				logger.String("user_id", userID.String()),
				logger.Error(err),
			)
		}
	}

	return nil
}

func (s *wsService) HandleClientDisconnect(ctx context.Context, userID uuid.UUID, deviceID string) error {
	userStillOnline := s.hub.IsOnline(userID)

	s.log.Info("Client disconnected",
		logger.String("user_id", userID.String()),
		logger.String("device_id", deviceID),
		logger.Bool("user_still_online", userStillOnline),
	)

	if err := s.userRepo.UpdateLastSeen(ctx, userID); err != nil {
		s.log.Error("Failed to update last seen on disconnect",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
	}

	if !userStillOnline {
		if err := s.userRepo.UpdateOnlineStatus(ctx, userID, "offline"); err != nil {
			s.log.Error("Failed to update online status on disconnect",
				logger.String("user_id", userID.String()),
				logger.Error(err),
			)
		}

		contacts, err := s.userRepo.GetUserContacts(ctx, userID)
		if err != nil {
			s.log.Error("Failed to get user contacts for presence broadcast",
				logger.String("user_id", userID.String()),
				logger.Error(err),
			)
			return nil
		}

		if len(contacts) > 0 {
			presenceEvent := &domain.BroadcastRequest{
				EventType:  domain.EventPresenceOffline,
				Recipients: contacts,
				Sender:     &userID,
				Payload: map[string]any{
					"user_id": userID.String(),
					"status":  "offline",
				},
				Priority: 1,
			}

			if _, err := s.BroadcastEvent(ctx, presenceEvent); err != nil {
				s.log.Warn("Failed to broadcast presence offline event",
					logger.String("user_id", userID.String()),
					logger.Error(err),
				)
			}
		}
	}

	return nil
}

func (s *wsService) BroadcastEvent(ctx context.Context, req *domain.BroadcastRequest) (*domain.BroadcastResponse, error) {
	if len(req.Recipients) == 0 {
		return nil, fmt.Errorf("no recipients specified")
	}

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

	onlineCount := 0
	for _, recipientID := range req.Recipients {
		if s.hub.IsOnline(recipientID) {
			data, err := s.marshalEvent(event)
			if err != nil {
				s.log.Error("Failed to marshal event",
					logger.String("event_id", event.ID.String()),
					logger.Error(err),
				)
				continue
			}

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

func (s *wsService) marshalEvent(event *domain.RealtimeEvent) ([]byte, error) {
	return json.Marshal(event)
}

func (s *wsService) IsUserOnline(ctx context.Context, userID uuid.UUID) (bool, error) {
	isOnline := s.hub.IsOnline(userID)

	s.log.Debug("Checked user online status",
		logger.String("user_id", userID.String()),
		logger.Bool("is_online", isOnline),
	)

	return isOnline, nil
}

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
