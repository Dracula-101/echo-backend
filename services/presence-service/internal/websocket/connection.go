package websocket

import (
	"context"
	"net/http"
	"time"

	req "shared/server/request"
	"shared/server/response"

	"shared/pkg/logger"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// ConnectionHandler handles WebSocket HTTP connection upgrades
type ConnectionHandler struct {
	manager  *Manager
	logger   logger.Logger
	upgrader websocket.Upgrader
}

// NewConnectionHandler creates a new WebSocket connection handler
func NewConnectionHandler(manager *Manager, log logger.Logger) *ConnectionHandler {
	return &ConnectionHandler{
		manager: manager,
		logger:  log,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// TODO: Implement proper origin checking in production
				return true
			},
		},
	}
}

// HandleConnection upgrades HTTP connection to WebSocket for presence tracking
func (h *ConnectionHandler) HandleConnection(w http.ResponseWriter, r *http.Request) {
	handler := req.NewHandler(r, w)
	requestID := handler.GetRequestID()

	h.logger.Info("WebSocket connection request received",
		logger.String("service", "presence-service"),
		logger.String("request_id", requestID),
		logger.String("client_ip", handler.GetClientIP()),
		logger.String("url", r.URL.String()),
		logger.String("raw_query", r.URL.RawQuery),
	)

	userID := r.URL.Query().Get("user_id")

	h.logger.Debug("Extracted user_id from query",
		logger.String("user_id", userID),
		logger.String("query_params", r.URL.RawQuery),
	)

	// If not in query params, try to get from context (for middleware-based auth)
	if userID == "" {
		contextUserID, ok := req.GetUserIDFromContext(r.Context())
		if ok {
			userID = contextUserID
		}
	}

	if userID == "" {
		h.logger.Warn("User ID not provided for WebSocket connection",
			logger.String("request_id", requestID),
		)
		response.UnauthorizedError(r.Context(), r, w, "User ID required. Provide via query parameter: ?user_id=xxx", nil)
		return
	}

	// Parse and validate user ID format
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		h.logger.Warn("Invalid user ID format",
			logger.String("request_id", requestID),
			logger.String("user_id", userID),
			logger.Error(err),
		)
		response.BadRequestError(r.Context(), r, w, "Invalid user ID format. Must be a valid UUID", err)
		return
	}

	// Validate user exists in database
	exists, err := h.manager.ValidateUserExists(r.Context(), userUUID)
	if err != nil {
		h.logger.Error("Failed to validate user existence",
			logger.String("request_id", requestID),
			logger.String("user_id", userID),
			logger.Error(err),
		)
		response.InternalServerError(r.Context(), r, w, "Failed to validate user", err)
		return
	}

	if !exists {
		h.logger.Warn("User not found in database",
			logger.String("request_id", requestID),
			logger.String("user_id", userID),
		)
		response.NotFoundError(r.Context(), r, w, "User not found")
		return
	}

	deviceInfo := handler.GetDeviceInfo()

	metadata := ClientMetadata{
		IPAddress:   handler.GetClientIP(),
		UserAgent:   handler.GetUserAgent(),
		Platform:    deviceInfo.Platform,
		AppVersion:  r.Header.Get("X-App-Version"),
		DeviceName:  deviceInfo.Name,
		DeviceType:  deviceInfo.Type,
		ConnectedAt: time.Now(),
	}

	h.logger.Debug("Upgrading connection to WebSocket",
		logger.String("user_id", userID),
		logger.String("device_id", deviceInfo.ID),
		logger.String("platform", deviceInfo.Platform),
	)

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade connection",
			logger.String("user_id", userID),
			logger.Error(err),
		)
		return
	}

	client := NewClient(
		uuid.MustParse(userID),
		deviceInfo.ID,
		conn,
		h.manager.GetHub(),
		h.logger,
		metadata,
	)

	h.manager.GetHub().Register(client)

	h.logger.Info("WebSocket connection established",
		logger.String("user_id", userID),
		logger.String("device_id", deviceInfo.ID),
		logger.String("client_id", client.ID),
	)

	if err := h.manager.HandleClientConnect(r.Context(), client); err != nil {
		h.logger.Error("Failed to handle client connect",
			logger.String("client_id", client.ID),
			logger.Error(err),
		)
	}

	messageHandler := h.createMessageHandler(context.Background(), client)

	go client.WritePump()
	go client.ReadPump(messageHandler)
}

// createMessageHandler creates a message handler with all callbacks wired to the manager
func (h *ConnectionHandler) createMessageHandler(ctx context.Context, client *Client) MessageHandler {
	return CreateMessageHandler(
		h.logger,
		// onPresenceUpdate
		func(userID uuid.UUID, status string, customStatus string) {
			if err := h.manager.HandlePresenceUpdate(ctx, client, status, customStatus); err != nil {
				h.logger.Error("Failed to handle presence update",
					logger.String("user_id", userID.String()),
					logger.Error(err),
				)
			}
		},
		// onHeartbeat
		func(userID uuid.UUID, deviceID string) {
			if err := h.manager.HandleHeartbeat(ctx, client); err != nil {
				h.logger.Error("Failed to handle heartbeat",
					logger.String("user_id", userID.String()),
					logger.Error(err),
				)
			}
		},
		// onTyping
		func(userID uuid.UUID, conversationID uuid.UUID, isTyping bool) {
			if err := h.manager.HandleTypingIndicator(ctx, client, conversationID, isTyping); err != nil {
				h.logger.Error("Failed to handle typing indicator",
					logger.String("user_id", userID.String()),
					logger.Error(err),
				)
			}
		},
	)
}
