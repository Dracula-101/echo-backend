package router

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"shared/server/websocket"
	"ws-service/internal/ws/protocol"

	"shared/pkg/logger"
)

type MessageHandler interface {
	Handle(ctx context.Context, client *websocket.Client, msg *protocol.ClientMessage) error
	MessageType() protocol.ClientMessageType
}

type MessageRouter struct {
	handlers map[protocol.ClientMessageType]MessageHandler
	log      logger.Logger
}

func NewMessageRouter(log logger.Logger) *MessageRouter {
	return &MessageRouter{
		handlers: make(map[protocol.ClientMessageType]MessageHandler),
		log:      log,
	}
}

func (r *MessageRouter) RegisterHandler(handler MessageHandler) {
	r.handlers[handler.MessageType()] = handler
	r.log.Info("Registered message handler",
		logger.String("type", string(handler.MessageType())),
	)
}

func (r *MessageRouter) validateMessage(msg *protocol.ClientMessage) error {
	if msg.ID == "" {
		return fmt.Errorf("message ID is required")
	}
	if msg.Type == "" {
		return fmt.Errorf("message type is required")
	}
	return nil
}

func (r *MessageRouter) sendError(client *websocket.Client, requestID, code, message string, details interface{}) {
	errorMsg := protocol.ServerMessage{
		ID:        fmt.Sprintf("err_%d", time.Now().UnixNano()),
		Type:      protocol.ServerTypeError,
		RequestID: requestID,
		Payload: protocol.ErrorPayload{
			Code:    code,
			Message: message,
			Details: fmt.Sprintf("%v", details),
		},
		Timestamp: time.Now(),
	}

	r.log.Info("Sending error message",
		logger.String("client_id", client.ID),
		logger.String("error_code", code),
		logger.String("error_message", message),
		logger.String("request_id", requestID),
	)

	if err := client.SendMessage(errorMsg); err != nil {
		r.log.Error("Failed to send error message",
			logger.String("client_id", client.ID),
			logger.Error(err),
		)
	}
}

// Route routes a message to the appropriate handler
func (r *MessageRouter) Route(ctx context.Context, client *websocket.Client, rawMessage []byte) error {
	r.log.Info("Message router received message",
		logger.String("client_id", client.ID),
		logger.String("raw_message", string(rawMessage)),
	)

	var msg protocol.ClientMessage
	
	if err := json.Unmarshal(rawMessage, &msg); err != nil {
		r.log.Error("Invalid JSON message",
			logger.String("client_id", client.ID),
			logger.String("raw_message", string(rawMessage)),
			logger.Error(err),
		)
		r.sendError(client, "", "invalid_json", "Message must be valid JSON", nil)
		return nil
	}

	r.log.Info("Parsed message successfully",
		logger.String("client_id", client.ID),
		logger.String("message_id", msg.ID),
		logger.String("message_type", string(msg.Type)),
	)

	if err := r.validateMessage(&msg); err != nil {
		r.log.Warn("Invalid message structure",
			logger.String("client_id", client.ID),
			logger.String("message_id", msg.ID),
			logger.Error(err),
		)
		r.sendError(client, msg.ID, "invalid_structure", err.Error(), nil)
		return nil
	}

	handler, ok := r.handlers[msg.Type]
	if !ok {
		r.log.Warn("Unknown message type",
			logger.String("client_id", client.ID),
			logger.String("message_id", msg.ID),
			logger.String("type", string(msg.Type)),
		)
		r.sendError(client, msg.ID, "unknown_type", fmt.Sprintf("Unknown message type: %s", msg.Type), nil)
		return nil
	}

	r.log.Info("Calling message handler",
		logger.String("client_id", client.ID),
		logger.String("message_id", msg.ID),
		logger.String("type", string(msg.Type)),
	)

	if err := handler.Handle(ctx, client, &msg); err != nil {
		r.log.Error("Handler error",
			logger.String("client_id", client.ID),
			logger.String("message_id", msg.ID),
			logger.String("type", string(msg.Type)),
			logger.Error(err),
		)
		r.sendError(client, msg.ID, "handler_error", "Failed to process message", nil)
		return nil
	}

	r.log.Info("Message processed successfully",
		logger.String("client_id", client.ID),
		logger.String("message_id", msg.ID),
		logger.String("type", string(msg.Type)),
	)

	return nil
}


