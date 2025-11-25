package websocket

import (
	"context"
	"net/http"
	"strings"
	"time"

	"shared/pkg/logger"
	"shared/server/headers"
	req "shared/server/request"
	"shared/server/response"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Upgrader handles HTTP to WebSocket upgrades
type Upgrader struct {
	upgrader *websocket.Upgrader
	hub      *Hub
	config   *Config
	log      logger.Logger

	// User validation
	validateUser UserValidator

	// Pre-upgrade hook
	beforeUpgrade BeforeUpgradeHook

	// Post-upgrade hook
	afterUpgrade AfterUpgradeHook
}

// UserValidator validates if a user exists and is allowed to connect
type UserValidator func(ctx context.Context, userID uuid.UUID) (bool, error)

// BeforeUpgradeHook is called before upgrading the connection
type BeforeUpgradeHook func(w http.ResponseWriter, r *http.Request, userID uuid.UUID) error

// AfterUpgradeHook is called after successful upgrade
type AfterUpgradeHook func(ctx context.Context, client *Client) error

// NewUpgrader creates a new WebSocket upgrader
func NewUpgrader(hub *Hub, config *Config, log logger.Logger) *Upgrader {
	if config == nil {
		config = DefaultConfig()
	}

	upgrader := &websocket.Upgrader{
		ReadBufferSize:   config.ReadBufferSize,
		WriteBufferSize:  config.WriteBufferSize,
		HandshakeTimeout: config.HandshakeTimeout,
		CheckOrigin:      createOriginChecker(config),
	}

	if config.EnableCompression {
		upgrader.EnableCompression = true
	}

	return &Upgrader{
		upgrader: upgrader,
		hub:      hub,
		config:   config,
		log:      log,
	}
}

// SetUserValidator sets the user validator function
func (u *Upgrader) SetUserValidator(validator UserValidator) {
	u.validateUser = validator
}

// SetBeforeUpgrade sets the before upgrade hook
func (u *Upgrader) SetBeforeUpgrade(hook BeforeUpgradeHook) {
	u.beforeUpgrade = hook
}

// SetAfterUpgrade sets the after upgrade hook
func (u *Upgrader) SetAfterUpgrade(hook AfterUpgradeHook) {
	u.afterUpgrade = hook
}

// HandleUpgrade handles the HTTP upgrade to WebSocket
func (u *Upgrader) HandleUpgrade(w http.ResponseWriter, r *http.Request, messageHandler MessageHandler) {
	handler := req.NewHandler(r, w)
	requestID := handler.GetRequestID()

	u.log.Info("WebSocket upgrade request received",
		logger.String("request_id", requestID),
		logger.String("client_ip", handler.GetClientIP()),
		logger.String("url", r.URL.String()),
		logger.String("origin", r.Header.Get("Origin")),
		logger.String("user_agent", r.UserAgent()),
	)

	// Extract and validate user_id
	userID, err := u.extractUserID(r)
	if err != nil {
		u.log.Warn("Invalid user_id",
			logger.String("request_id", requestID),
			logger.Error(err),
		)
		response.BadRequestError(r.Context(), r, w, "Invalid user_id format", nil)
		return
	}

	// Validate user exists (if validator is set)
	if u.validateUser != nil {
		exists, err := u.validateUser(r.Context(), userID)
		if err != nil {
			u.log.Error("Failed to validate user",
				logger.String("request_id", requestID),
				logger.String("user_id", userID.String()),
				logger.Error(err),
			)
			response.InternalServerError(r.Context(), r, w, "Failed to validate user", err)
			return
		}

		if !exists {
			u.log.Warn("User not found",
				logger.String("request_id", requestID),
				logger.String("user_id", userID.String()),
			)
			response.NotFoundError(r.Context(), r, w, "User not found")
			return
		}
	}

	// Call before upgrade hook
	if u.beforeUpgrade != nil {
		if err := u.beforeUpgrade(w, r, userID); err != nil {
			u.log.Error("Before upgrade hook failed",
				logger.String("request_id", requestID),
				logger.String("user_id", userID.String()),
				logger.Error(err),
			)
			response.InternalServerError(r.Context(), r, w, "Upgrade validation failed", err)
			return
		}
	}

	// Extract device info
	deviceInfo := handler.GetDeviceInfo()

	// Build client metadata
	metadata := ClientMetadata{
		UserID:      userID,
		DeviceID:    deviceInfo.ID,
		IPAddress:   handler.GetClientIP(),
		UserAgent:   handler.GetUserAgent(),
		Platform:    deviceInfo.Platform,
		AppVersion:  r.Header.Get("X-App-Version"),
		DeviceName:  deviceInfo.Name,
		DeviceType:  deviceInfo.Type,
		ConnectedAt: time.Now(),
		LastPingAt:  time.Now(),
		CustomData:  make(map[string]string),
	}

	// Extract custom headers (if any)
	for key, values := range r.Header {
		if strings.HasPrefix(key, "X-Custom-") {
			customKey := strings.TrimPrefix(key, "X-Custom-")
			if len(values) > 0 {
				metadata.CustomData[customKey] = values[0]
			}
		}
	}

	u.log.Debug("Upgrading connection to WebSocket",
		logger.String("user_id", userID.String()),
		logger.String("device_id", deviceInfo.ID),
		logger.String("platform", deviceInfo.Platform),
	)

	// Upgrade connection
	conn, err := u.upgrader.Upgrade(w, r, nil)
	if err != nil {
		u.log.Error("Failed to upgrade connection",
			logger.String("user_id", userID.String()),
			logger.Error(err),
		)
		return
	}

	// Wrap connection
	wrappedConn := WrapConn(conn)

	// Create client
	client := NewClient(
		userID,
		deviceInfo.ID,
		wrappedConn,
		u.hub,
		metadata,
		u.config,
		u.log,
	)

	// Set message handler
	if messageHandler != nil {
		client.SetMessageHandler(messageHandler)
	}

	// Set disconnect callback from hub
	if u.hub.onDisconnect != nil {
		client.SetOnDisconnect(u.hub.onDisconnect)
	}

	// Register client with hub
	u.hub.Register(client)

	u.log.Info("WebSocket connection established",
		logger.String("user_id", userID.String()),
		logger.String("device_id", deviceInfo.ID),
		logger.String("client_id", client.ID),
	)

	// Call after upgrade hook
	if u.afterUpgrade != nil {
		if err := u.afterUpgrade(r.Context(), client); err != nil {
			u.log.Error("After upgrade hook failed",
				logger.String("client_id", client.ID),
				logger.Error(err),
			)
			// Don't close connection, just log the error
		}
	}

	// Send welcome message
	welcomeMsg := map[string]interface{}{
		"type":      "connected",
		"client_id": client.ID,
		"timestamp": time.Now().Unix(),
		"message":   "WebSocket connection established",
	}
	if err := client.SendMessage(welcomeMsg); err != nil {
		u.log.Error("Failed to send welcome message",
			logger.String("client_id", client.ID),
			logger.Error(err),
		)
	}

	// Start client read and write pumps
	go client.WritePump()
	go client.ReadPump()
}

// extractUserID extracts and validates user_id from request
func (u *Upgrader) extractUserID(r *http.Request) (uuid.UUID, error) {
	// Try query parameter first
	userIDStr := r.URL.Query().Get("user_id")

	// If not in query, try header
	if userIDStr == "" {
		userIDStr = r.Header.Get(headers.XUserID)
	}

	if userIDStr == "" {
		return uuid.Nil, NewValidationError("user_id", "", "user_id is required")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, NewValidationError("user_id", userIDStr, "invalid UUID format")
	}

	return userID, nil
}

// createOriginChecker creates an origin checker function
func createOriginChecker(config *Config) func(r *http.Request) bool {
	return func(r *http.Request) bool {
		// If origin checking is disabled, allow all
		if !config.CheckOrigin {
			return true
		}

		// If no allowed origins configured, deny all
		if len(config.AllowedOrigins) == 0 {
			return false
		}

		origin := r.Header.Get("Origin")
		if origin == "" {
			// No origin header, might be same-origin request
			return true
		}

		// Check if origin is in allowed list
		for _, allowed := range config.AllowedOrigins {
			if origin == allowed {
				return true
			}

			// Support wildcard patterns
			if strings.HasPrefix(allowed, "*") {
				suffix := strings.TrimPrefix(allowed, "*")
				if strings.HasSuffix(origin, suffix) {
					return true
				}
			}
		}

		return false
	}
}

// UpgradeHandler returns an http.HandlerFunc that handles WebSocket upgrades
func (u *Upgrader) UpgradeHandler(messageHandler MessageHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u.HandleUpgrade(w, r, messageHandler)
	}
}

// UpgradeHandlerWithContext returns an http.HandlerFunc with context support
func (u *Upgrader) UpgradeHandlerWithContext(
	messageHandlerFactory func(ctx context.Context) MessageHandler,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		messageHandler := messageHandlerFactory(ctx)
		u.HandleUpgrade(w, r, messageHandler)
	}
}
