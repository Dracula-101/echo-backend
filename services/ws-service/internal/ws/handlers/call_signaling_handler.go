package handlers

import (
	"context"
	"encoding/json"
	"time"
	"shared/server/websocket"
	"ws-service/internal/ws/broadcast"
	"ws-service/internal/ws/protocol"

	"shared/pkg/logger"
)

// CallOfferHandler handles WebRTC call offer messages
type CallOfferHandler struct {
	broadcaster *broadcast.Broadcaster
	hub         *websocket.Hub
	log         logger.Logger
}

// NewCallOfferHandler creates a new call offer handler
func NewCallOfferHandler(broadcaster *broadcast.Broadcaster, hub *websocket.Hub, log logger.Logger) *CallOfferHandler {
	return &CallOfferHandler{
		broadcaster: broadcaster,
		hub:         hub,
		log:         log,
	}
}

// Handle handles a call offer message
func (h *CallOfferHandler) Handle(ctx context.Context, client *websocket.Client, msg *protocol.ClientMessage) error {
	var payload protocol.CallSignalingPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	h.log.Info("Processing call offer",
		logger.String("client_id", client.ID),
		logger.String("call_id", payload.CallID.String()),
		logger.Int("participants", len(payload.Participants)),
	)

	// Forward offer to participants
	offerMsg := protocol.ServerMessage{
		ID:   generateBroadcastID(),
		Type: protocol.ServerTypeCallOffer,
		Payload: map[string]interface{}{
			"call_id":      payload.CallID,
			"from_user_id": client.UserID,
			"sdp":          payload.SDP,
			"participants": payload.Participants,
		},
		Timestamp: time.Now(),
	}

	// Send to all participants except sender
	for _, participantID := range payload.Participants {
		if participantID != client.UserID {
			h.broadcaster.BroadcastToUser(participantID, offerMsg)
		}
	}

	return nil
}

// MessageType returns the message type this handler handles
func (h *CallOfferHandler) MessageType() protocol.ClientMessageType {
	return protocol.TypeCallOffer
}

// CallAnswerHandler handles WebRTC call answer messages
type CallAnswerHandler struct {
	broadcaster *broadcast.Broadcaster
	log         logger.Logger
}

// NewCallAnswerHandler creates a new call answer handler
func NewCallAnswerHandler(broadcaster *broadcast.Broadcaster, log logger.Logger) *CallAnswerHandler {
	return &CallAnswerHandler{
		broadcaster: broadcaster,
		log:         log,
	}
}

// Handle handles a call answer message
func (h *CallAnswerHandler) Handle(ctx context.Context, client *websocket.Client, msg *protocol.ClientMessage) error {
	var payload protocol.CallSignalingPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	h.log.Info("Processing call answer",
		logger.String("client_id", client.ID),
		logger.String("call_id", payload.CallID.String()),
	)

	answerMsg := protocol.ServerMessage{
		ID:   generateBroadcastID(),
		Type: protocol.ServerTypeCallAnswer,
		Payload: map[string]interface{}{
			"call_id":      payload.CallID,
			"from_user_id": client.UserID,
			"sdp":          payload.SDP,
		},
		Timestamp: time.Now(),
	}

	// Forward answer to all participants except sender
	for _, participantID := range payload.Participants {
		if participantID != client.UserID {
			h.broadcaster.BroadcastToUser(participantID, answerMsg)
		}
	}

	return nil
}

// MessageType returns the message type this handler handles
func (h *CallAnswerHandler) MessageType() protocol.ClientMessageType {
	return protocol.TypeCallAnswer
}

// CallICEHandler handles WebRTC ICE candidate messages
type CallICEHandler struct {
	broadcaster *broadcast.Broadcaster
	log         logger.Logger
}

// NewCallICEHandler creates a new call ICE handler
func NewCallICEHandler(broadcaster *broadcast.Broadcaster, log logger.Logger) *CallICEHandler {
	return &CallICEHandler{
		broadcaster: broadcaster,
		log:         log,
	}
}

// Handle handles a call ICE candidate message
func (h *CallICEHandler) Handle(ctx context.Context, client *websocket.Client, msg *protocol.ClientMessage) error {
	var payload protocol.CallSignalingPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	h.log.Debug("Processing ICE candidate",
		logger.String("client_id", client.ID),
		logger.String("call_id", payload.CallID.String()),
	)

	iceMsg := protocol.ServerMessage{
		ID:   generateBroadcastID(),
		Type: protocol.ServerTypeCallICE,
		Payload: map[string]interface{}{
			"call_id":       payload.CallID,
			"from_user_id":  client.UserID,
			"ice_candidate": payload.ICECandidate,
		},
		Timestamp: time.Now(),
	}

	// Forward ICE candidate to all participants except sender
	for _, participantID := range payload.Participants {
		if participantID != client.UserID {
			h.broadcaster.BroadcastToUser(participantID, iceMsg)
		}
	}

	return nil
}

// MessageType returns the message type this handler handles
func (h *CallICEHandler) MessageType() protocol.ClientMessageType {
	return protocol.TypeCallICE
}

// CallHangupHandler handles call hangup messages
type CallHangupHandler struct {
	broadcaster *broadcast.Broadcaster
	log         logger.Logger
}

// NewCallHangupHandler creates a new call hangup handler
func NewCallHangupHandler(broadcaster *broadcast.Broadcaster, log logger.Logger) *CallHangupHandler {
	return &CallHangupHandler{
		broadcaster: broadcaster,
		log:         log,
	}
}

// Handle handles a call hangup message
func (h *CallHangupHandler) Handle(ctx context.Context, client *websocket.Client, msg *protocol.ClientMessage) error {
	var payload protocol.CallSignalingPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	h.log.Info("Processing call hangup",
		logger.String("client_id", client.ID),
		logger.String("call_id", payload.CallID.String()),
	)

	hangupMsg := protocol.ServerMessage{
		ID:   generateBroadcastID(),
		Type: protocol.ServerTypeCallEnded,
		Payload: map[string]interface{}{
			"call_id":      payload.CallID,
			"from_user_id": client.UserID,
			"reason":       "hangup",
		},
		Timestamp: time.Now(),
	}

	// Notify all participants
	for _, participantID := range payload.Participants {
		if participantID != client.UserID {
			h.broadcaster.BroadcastToUser(participantID, hangupMsg)
		}
	}

	return nil
}

// MessageType returns the message type this handler handles
func (h *CallHangupHandler) MessageType() protocol.ClientMessageType {
	return protocol.TypeCallHangup
}
