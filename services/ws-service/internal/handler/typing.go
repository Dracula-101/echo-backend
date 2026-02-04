package handler

import (
	"context"
	"encoding/json"
	"time"

	"shared/pkg/logger"
	"ws-service/internal/protocol"
)

type TypingStartHandler struct {
	deps *Dependencies
	log  logger.Logger
}

func NewTypingStartHandler(deps *Dependencies, log logger.Logger) *TypingStartHandler {
	return &TypingStartHandler{deps: deps, log: log}
}

func (h *TypingStartHandler) Handle(ctx context.Context, hctx *Context) error {
	if !hctx.Conn.IsAuthenticated() {
		return sendError(hctx, protocol.ErrCodeUnauthorized, "User not authenticated")
	}

	var payload protocol.TypingPayload
	if err := json.Unmarshal(hctx.Payload, &payload); err != nil {
		return err
	}

	userID := hctx.Conn.UserID()
	h.deps.Typing.StartTyping(payload.ConversationID, userID)

	return h.deps.Broadcaster.ToConversation(payload.ConversationID, string(protocol.MsgTypeTypingStart),
		protocol.TypingEvent{
			UserID:         userID,
			ConversationID: payload.ConversationID,
			IsTyping:       true,
			Timestamp:      time.Now(),
		}, userID)
}

type TypingStopHandler struct {
	deps *Dependencies
	log  logger.Logger
}

func NewTypingStopHandler(deps *Dependencies, log logger.Logger) *TypingStopHandler {
	return &TypingStopHandler{deps: deps, log: log}
}

func (h *TypingStopHandler) Handle(ctx context.Context, hctx *Context) error {
	if !hctx.Conn.IsAuthenticated() {
		return sendError(hctx, protocol.ErrCodeUnauthorized, "User not authenticated")
	}

	var payload protocol.TypingPayload
	if err := json.Unmarshal(hctx.Payload, &payload); err != nil {
		return err
	}

	userID := hctx.Conn.UserID()
	h.deps.Typing.StopTyping(payload.ConversationID, userID)

	return h.deps.Broadcaster.ToConversation(payload.ConversationID, string(protocol.MsgTypeTypingStop),
		protocol.TypingEvent{
			UserID:         userID,
			ConversationID: payload.ConversationID,
			IsTyping:       false,
			Timestamp:      time.Now(),
		}, userID)
}
