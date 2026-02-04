package handler

import (
	"context"
	"encoding/json"
	"time"

	"shared/pkg/logger"
	"ws-service/internal/protocol"
)

type MarkReadHandler struct {
	deps *Dependencies
	log  logger.Logger
}

func NewMarkReadHandler(deps *Dependencies, log logger.Logger) *MarkReadHandler {
	return &MarkReadHandler{deps: deps, log: log}
}

func (h *MarkReadHandler) Handle(ctx context.Context, hctx *Context) error {
	if !hctx.Conn.IsAuthenticated() {
		return sendError(hctx, protocol.ErrCodeUnauthorized, "User not authenticated")
	}

	var payload protocol.ReadReceiptPayload
	if err := json.Unmarshal(hctx.Payload, &payload); err != nil {
		return err
	}

	userID := hctx.Conn.UserID()
	now := time.Now()

	if h.deps.Publisher != nil {
		event := &ReadReceiptEvent{
			UserID:         userID,
			ConversationID: payload.ConversationID,
			MessageIDs:     payload.MessageIDs,
		}
		if err := h.deps.Publisher.PublishReadReceipt(ctx, event); err != nil {
			h.log.Error("failed to publish read receipt event",
				logger.String("user_id", userID.String()),
				logger.String("conversation_id", payload.ConversationID.String()),
				logger.Error(err),
			)
		}
	}

	return h.deps.Broadcaster.ToConversation(payload.ConversationID, string(protocol.MsgTypeMessageRead),
		protocol.ReadReceiptEvent{
			UserID:         userID,
			ConversationID: payload.ConversationID,
			MessageIDs:     payload.MessageIDs,
			ReadAt:         now,
		}, userID)
}

type MarkDeliveredHandler struct {
	deps *Dependencies
	log  logger.Logger
}

func NewMarkDeliveredHandler(deps *Dependencies, log logger.Logger) *MarkDeliveredHandler {
	return &MarkDeliveredHandler{deps: deps, log: log}
}

func (h *MarkDeliveredHandler) Handle(ctx context.Context, hctx *Context) error {
	if !hctx.Conn.IsAuthenticated() {
		return sendError(hctx, protocol.ErrCodeUnauthorized, "User not authenticated")
	}

	var payload protocol.DeliveredReceiptPayload
	if err := json.Unmarshal(hctx.Payload, &payload); err != nil {
		return err
	}

	userID := hctx.Conn.UserID()
	now := time.Now()

	if h.deps.Publisher != nil {
		event := &DeliveredReceiptEvent{
			UserID:         userID,
			ConversationID: payload.ConversationID,
			MessageIDs:     payload.MessageIDs,
		}
		if err := h.deps.Publisher.PublishDeliveredReceipt(ctx, event); err != nil {
			h.log.Error("failed to publish delivered receipt event",
				logger.String("user_id", userID.String()),
				logger.String("conversation_id", payload.ConversationID.String()),
				logger.Error(err),
			)
		}
	}

	return h.deps.Broadcaster.ToConversation(payload.ConversationID, string(protocol.MsgTypeMessageDelivered),
		protocol.DeliveredReceiptEvent{
			UserID:         userID,
			ConversationID: payload.ConversationID,
			MessageIDs:     payload.MessageIDs,
			DeliveredAt:    now,
		}, userID)
}
