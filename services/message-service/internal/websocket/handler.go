package websocket

import (
	"net/http"
	"time"

	req "shared/server/request"
	"shared/server/response"

	"shared/pkg/logger"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Handler handles WebSocket HTTP connections
type Handler struct {
	hub      *Hub
	logger   logger.Logger
	upgrader websocket.Upgrader
}

// NewHandler creates a new WebSocket HTTP handler
func NewHandler(hub *Hub, log logger.Logger) *Handler {
	return &Handler{
		hub:    hub,
		logger: log,
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

// HandleConnection upgrades HTTP connection to WebSocket
func (h *Handler) HandleConnection(w http.ResponseWriter, r *http.Request) {
	handler := req.NewHandler(r, w)
	requestID := handler.GetRequestID()

	h.logger.Info("WebSocket connection request received",
		logger.String("service", "message-service"),
		logger.String("request_id", requestID),
		logger.String("client_ip", handler.GetClientIP()),
	)

	// Extract user_id from context (set by auth middleware in API Gateway)
	userID, ok := req.GetUserIDFromContext(r.Context())
	if !ok {
		h.logger.Warn("User not authenticated for WebSocket connection",
			logger.String("request_id", requestID),
		)
		response.UnauthorizedError(r.Context(), r, w, "User not authenticated", nil)
		return
	}

	// Extract device ID from headers (optional)
	deviceID := r.Header.Get("X-Device-ID")
	if deviceID == "" {
		deviceID = "unknown"
	}

	// Extract platform from headers (optional)
	platform := r.Header.Get("X-Platform")
	if platform == "" {
		platform = "unknown"
	}

	h.logger.Debug("Upgrading connection to WebSocket",
		logger.String("user_id", userID),
		logger.String("device_id", deviceID),
		logger.String("platform", platform),
	)

	// Upgrade HTTP connection to WebSocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade connection",
			logger.String("user_id", userID),
			logger.Error(err),
		)
		// Upgrader already sent error response
		return
	}

	// Create client metadata
	metadata := ClientMetadata{
		IPAddress:   handler.GetClientIP(),
		UserAgent:   handler.GetUserAgent(),
		Platform:    platform,
		AppVersion:  r.Header.Get("X-App-Version"),
		ConnectedAt: time.Now(),
	}

	// Create client
	client := NewClient(
		uuid.MustParse(userID),
		deviceID,
		conn,
		h.hub,
		h.logger,
		metadata,
	)

	// Register client with hub
	h.hub.Register(client)

	h.logger.Info("WebSocket connection established",
		logger.String("user_id", userID),
		logger.String("device_id", deviceID),
		logger.String("client_id", client.ID),
	)

	// Start client read/write pumps
	go client.WritePump()
	go client.ReadPump(nil) // TODO: Add message handler for incoming WebSocket messages
}
