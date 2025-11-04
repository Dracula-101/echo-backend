package handler

import (
	"context"
	"net/http"
	"time"

	"echo-backend/services/message-service/internal/service"
	"echo-backend/services/message-service/internal/websocket"

	"github.com/google/uuid"
	ws "github.com/gorilla/websocket"
	"shared/pkg/logger"
)

var upgrader = ws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: Implement proper origin check based on environment
		// In production, check against allowed origins
		return true
	},
}

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	hub            *websocket.Hub
	messageService service.MessageService
	logger         logger.Logger
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(hub *websocket.Hub, messageService service.MessageService, logger logger.Logger) *WebSocketHandler {
	return &WebSocketHandler{
		hub:            hub,
		messageService: messageService,
		logger:         logger,
	}
}

// HandleWebSocket upgrades HTTP connection to WebSocket
func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context or header (set by auth middleware)
	userID, err := h.getUserID(r)
	if err != nil {
		h.logger.Error("Failed to get user ID",
			logger.Error(err),
		)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get device ID from header
	deviceID := r.Header.Get("X-Device-ID")
	if deviceID == "" {
		deviceID = uuid.New().String()
	}

	// Get client metadata
	metadata := websocket.ClientMetadata{
		IPAddress:   getClientIP(r),
		UserAgent:   r.UserAgent(),
		Platform:    r.Header.Get("X-Platform"),
		AppVersion:  r.Header.Get("X-App-Version"),
		ConnectedAt: time.Now(),
	}

	// Upgrade connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade connection",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return
	}

	// Create client
	client := websocket.NewClient(userID, deviceID, conn, h.hub, h.logger, metadata)

	h.logger.Info("WebSocket connection established",
		logger.String("user_id", userID.String()),
		logger.String("device_id", deviceID),
		logger.String("ip", metadata.IPAddress),
		logger.String("platform", metadata.Platform),
	)

	// Register client with hub
	h.hub.Register(client)

	// Create message handler for incoming WebSocket messages
	messageHandler := websocket.CreateMessageHandler(
		h.logger,
		h.handleReadReceipt,
		h.handleTyping,
	)

	// Start client pumps
	go client.WritePump()
	go client.ReadPump(messageHandler)
}

// handleReadReceipt processes read receipts from WebSocket
func (h *WebSocketHandler) handleReadReceipt(userID, messageID uuid.UUID) {
	ctx := context.Background()

	if err := h.messageService.HandleReadReceipt(ctx, userID, messageID); err != nil {
		h.logger.Error("Failed to handle read receipt",
			logger.String("user_id", userID.String()),
			logger.String("message_id", messageID.String()),
			logger.Error(err),
		)
	}
}

// handleTyping processes typing indicators from WebSocket
func (h *WebSocketHandler) handleTyping(userID, conversationID uuid.UUID, isTyping bool) {
	ctx := context.Background()

	if err := h.messageService.SetTypingIndicator(ctx, conversationID, userID, isTyping); err != nil {
		h.logger.Error("Failed to handle typing indicator",
			logger.String("user_id", userID.String()),
			logger.String("conversation_id", conversationID.String()),
			logger.Bool("is_typing", isTyping),
			logger.Error(err),
		)
	}
}

// getUserID extracts user ID from request
func (h *WebSocketHandler) getUserID(r *http.Request) (uuid.UUID, error) {
	// Try to get from context (set by auth middleware)
	if userIDVal := r.Context().Value("user_id"); userIDVal != nil {
		if userIDStr, ok := userIDVal.(string); ok {
			return uuid.Parse(userIDStr)
		}
		if userID, ok := userIDVal.(uuid.UUID); ok {
			return userID, nil
		}
	}

	// Fallback: get from header (for testing without auth middleware)
	if userIDStr := r.Header.Get("X-User-ID"); userIDStr != "" {
		return uuid.Parse(userIDStr)
	}

	// Try to get from query parameter (for WebSocket connections where headers might be limited)
	if token := r.URL.Query().Get("token"); token != "" {
		// TODO: Validate JWT token and extract user ID
		// For now, we'll just return an error
		return uuid.Nil, &AuthError{Message: "token validation not implemented"}
	}

	return uuid.Nil, &AuthError{Message: "user_id not found"}
}

// getClientIP gets the real client IP address
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (if behind proxy/load balancer)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}
