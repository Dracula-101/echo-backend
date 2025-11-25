package handlers

import (
	"context"
	"encoding/json"
	"time"
	"shared/server/websocket"
	"ws-service/internal/ws/presence"
	"ws-service/internal/ws/protocol"

	"shared/pkg/logger"
)

// PresenceUpdateHandler handles presence update messages
type PresenceUpdateHandler struct {
	presenceTracker *presence.Tracker
	log             logger.Logger
}

// NewPresenceUpdateHandler creates a new presence update handler
func NewPresenceUpdateHandler(presenceTracker *presence.Tracker, log logger.Logger) *PresenceUpdateHandler {
	return &PresenceUpdateHandler{
		presenceTracker: presenceTracker,
		log:             log,
	}
}

// Handle handles a presence update message
func (h *PresenceUpdateHandler) Handle(ctx context.Context, client *websocket.Client, msg *protocol.ClientMessage) error {
	var payload protocol.PresenceUpdatePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	h.log.Info("Processing presence update",
		logger.String("client_id", client.ID),
		logger.String("user_id", client.UserID.String()),
		logger.String("status", payload.Status),
	)

	// Update presence
	if err := h.presenceTracker.UpdatePresence(client.UserID, payload.Status, payload.CustomStatus); err != nil {
		h.log.Error("Failed to update presence",
			logger.String("user_id", client.UserID.String()),
			logger.Error(err),
		)
		return err
	}

	// Broadcast presence update to subscribers
	h.presenceTracker.BroadcastPresenceUpdate(client.UserID, payload.Status, payload.CustomStatus)

	return nil
}

// MessageType returns the message type this handler handles
func (h *PresenceUpdateHandler) MessageType() protocol.ClientMessageType {
	return protocol.TypePresenceUpdate
}

// PresenceQueryHandler handles presence query messages
type PresenceQueryHandler struct {
	presenceTracker *presence.Tracker
	log             logger.Logger
}

// NewPresenceQueryHandler creates a new presence query handler
func NewPresenceQueryHandler(presenceTracker *presence.Tracker, log logger.Logger) *PresenceQueryHandler {
	return &PresenceQueryHandler{
		presenceTracker: presenceTracker,
		log:             log,
	}
}

// Handle handles a presence query message
func (h *PresenceQueryHandler) Handle(ctx context.Context, client *websocket.Client, msg *protocol.ClientMessage) error {
	var payload protocol.PresenceQueryPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	h.log.Debug("Processing presence query",
		logger.String("client_id", client.ID),
		logger.Int("user_count", len(payload.UserIDs)),
	)

	// Get presence for requested users
	presences := h.presenceTracker.GetBulkPresence(payload.UserIDs)

	// Send response
	responseMsg := protocol.ServerMessage{
		ID:        generateResponseID(msg.ID),
		Type:      protocol.ServerTypePresenceUpdate,
		RequestID: msg.ID,
		Payload: map[string]interface{}{
			"presences": presences,
		},
		Timestamp: time.Now(),
	}

	return client.SendMessage(responseMsg)
}

// MessageType returns the message type this handler handles
func (h *PresenceQueryHandler) MessageType() protocol.ClientMessageType {
	return protocol.TypePresenceQuery
}
