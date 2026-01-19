package websocket

import (
	"context"
	"encoding/json"
	"time"

	"shared/server/websocket/router"
	"ws-service/internal/protocol"
	"ws-service/internal/websocket/tracker"

	"github.com/google/uuid"
)

func (m *Manager) handlePresenceUpdate(ctx context.Context, msg *router.Message) error {
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

	m.presence.UpdatePresence(userID, tracker.PresenceStatus(payload.Status), payload.CustomStatus)
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
