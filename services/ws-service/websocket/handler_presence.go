package websocket

import (
	"context"
	"encoding/json"
	"time"

	"shared/pkg/logger"
	"shared/server/websocket/router"
	"ws-service/internal/protocol"
	"ws-service/websocket/tracker"

	"github.com/google/uuid"
)

func (m *Manager) handlePresenceUpdate(ctx context.Context, msg *router.Message) error {
	m.log.Debug("Handling presence update message")
	conn, ok := m.getConnection(msg)
	if !ok {
		return nil
	}

	userID, ok := m.getUserID(conn)
	if !ok {
		return m.sendError(conn, m.getRequestID(msg), protocol.ErrCodeUnauthorized, "User not authenticated")
	}

	var payload protocol.PresenceUpdatePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	if ok := protocol.ValidatePresenceUpdatePayload(payload); !ok {
		m.log.Warn("Invalid presence update payload",
			logger.String("user_id", userID.String()),
			logger.String("status", payload.Status),
		)
		return m.sendError(conn, m.getRequestID(msg), protocol.ErrCodeInvalidPayload, "Invalid presence update payload")
	}

	status := tracker.PresenceStatus(payload.Status)
	if !status.IsValid() {
		m.log.Warn("Invalid presence status received",
			logger.String("user_id", userID.String()),
			logger.String("status", payload.Status),
		)
		return m.sendError(conn, m.getRequestID(msg), protocol.ErrCodeInvalidPayload, "Invalid presence status")
	}

	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := m.userRepo.UpdateOnlineStatus(dbCtx, userID, status.String())
	if err != nil {
		m.log.Error("Failed to update online status in database",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
	} else {
		m.presence.UpdatePresence(userID, status, payload.CustomStatus)
		m.log.Debug("Updated online status in database",
			logger.String("user_id", userID.String()),
			logger.String("status", status.String()),
		)
		m.SendMessageToUser(userID, protocol.NewSuccessMessage(
			m.getRequestID(msg),
			"Presence update successful",
		))
	}
	return nil
}

func (m *Manager) handlePresenceQuery(ctx context.Context, msg *router.Message) error {
	conn, ok := m.getConnection(msg)
	if !ok {
		return nil
	}

	var payload protocol.PresenceQueryPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	presenceMap := m.presence.GetBulkPresence(payload.UserIDs)
	presences := make([]protocol.PresenceInfo, 0, len(payload.UserIDs))
	offlineUsers := make([]uuid.UUID, 0)

	for _, userID := range payload.UserIDs {
		presence := presenceMap[userID]
		if presence == nil {
			presences = append(presences, protocol.PresenceInfo{
				UserID:     userID,
				Status:     string(tracker.StatusOffline),
				LastSeenAt: time.Now().UTC(),
			})
			offlineUsers = append(offlineUsers, userID)
			continue
		}

		if presence.Status == tracker.StatusOffline {
			offlineUsers = append(offlineUsers, userID)
		}

		presences = append(presences, protocol.PresenceInfo{
			UserID:       presence.UserID,
			Status:       string(presence.Status),
			CustomStatus: presence.CustomStatus,
			LastSeenAt:   presence.LastSeenAt,
			DeviceCount:  presence.DeviceCount,
		})
	}

	response := protocol.NewServerMessage(protocol.MsgTypePresenceResponse, m.getRequestID(msg)).WithPayload(
		protocol.PresenceResponsePayload{
			Presences:    presences,
			OfflineUsers: offlineUsers,
		},
	)

	data, _ := json.Marshal(response)
	return conn.Send(data)
}
