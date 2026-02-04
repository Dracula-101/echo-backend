package handler

import (
	"context"
	"encoding/json"
	"time"

	"shared/pkg/logger"
	"ws-service/internal/protocol"

	"github.com/google/uuid"
)

type PresenceUpdateHandler struct {
	deps *Dependencies
	log  logger.Logger
}

func NewPresenceUpdateHandler(deps *Dependencies, log logger.Logger) *PresenceUpdateHandler {
	return &PresenceUpdateHandler{deps: deps, log: log}
}

func (h *PresenceUpdateHandler) Handle(ctx context.Context, hctx *Context) error {
	if !hctx.Conn.IsAuthenticated() {
		return sendError(hctx, protocol.ErrCodeUnauthorized, "User not authenticated")
	}

	var payload protocol.PresenceUpdatePayload
	if err := json.Unmarshal(hctx.Payload, &payload); err != nil {
		return err
	}

	userID := hctx.Conn.UserID()
	h.deps.Presence.UpdateStatus(userID, payload.Status, payload.CustomStatus)
	return nil
}

type PresenceQueryHandler struct {
	deps *Dependencies
	log  logger.Logger
}

func NewPresenceQueryHandler(deps *Dependencies, log logger.Logger) *PresenceQueryHandler {
	return &PresenceQueryHandler{deps: deps, log: log}
}

func (h *PresenceQueryHandler) Handle(ctx context.Context, hctx *Context) error {
	var payload protocol.PresenceQueryPayload
	if err := json.Unmarshal(hctx.Payload, &payload); err != nil {
		return err
	}

	presenceMap := h.deps.Presence.GetBulkPresence(payload.UserIDs)
	presences := make([]protocol.PresenceInfo, 0, len(payload.UserIDs))
	offlineUsers := make([]uuid.UUID, 0)

	for _, userID := range payload.UserIDs {
		presence := presenceMap[userID]
		if presence == nil {
			presences = append(presences, protocol.PresenceInfo{
				UserID:     userID,
				Status:     "offline",
				LastSeenAt: time.Now().UTC(),
			})
			offlineUsers = append(offlineUsers, userID)
			continue
		}

		if presence.Status == "offline" {
			offlineUsers = append(offlineUsers, userID)
		}

		lastSeen, _ := time.Parse(time.RFC3339, presence.LastSeenAt)
		presences = append(presences, protocol.PresenceInfo{
			UserID:       presence.UserID,
			Status:       presence.Status,
			CustomStatus: presence.CustomStatus,
			LastSeenAt:   lastSeen,
			DeviceCount:  presence.DeviceCount,
		})
	}

	response := protocol.NewServerMessage(protocol.MsgTypePresenceResponse, hctx.RequestID).WithPayload(
		protocol.PresenceResponsePayload{
			Presences:    presences,
			OfflineUsers: offlineUsers,
		},
	)

	data, _ := json.Marshal(response)
	return hctx.Conn.Send(data)
}

func sendError(hctx *Context, code protocol.ErrorCode, message string) error {
	errMsg := protocol.NewErrorMessage(hctx.RequestID, code, message)
	data, _ := json.Marshal(errMsg)
	return hctx.Conn.Send(data)
}
