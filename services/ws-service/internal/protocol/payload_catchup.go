package protocol

import "encoding/json"

// CatchupPayload is sent on reconnect when ws-service finds undelivered
// messages in delivery_status. The client merges these with whatever it has
// in local storage. has_more=true means more than the page-size existed and
// the client should fall back to its REST history endpoint for the rest.
type CatchupPayload struct {
	Messages []CatchupMessage `json:"messages"`
	HasMore  bool             `json:"has_more"`
}

// CatchupMessage is a stripped-down view of messages.messages — just enough
// for the client to render until it pulls the full record via REST.
type CatchupMessage struct {
	MessageID      string          `json:"message_id"`
	ConversationID string          `json:"conversation_id"`
	SenderUserID   string          `json:"sender_user_id"`
	Content        json.RawMessage `json:"content"`
	CreatedAt      string          `json:"created_at"`
}

// SessionRevokedPayload tells the client this connection is being force-closed
// due to a logout/revocation elsewhere. Reason mirrors auth-service codes.
type SessionRevokedPayload struct {
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

// ServerShutdownPayload is sent before the server closes connections during
// graceful shutdown so clients reconnect to a different pod with backoff.
type ServerShutdownPayload struct {
	ReconnectAfterMs int    `json:"reconnect_after_ms"`
	Reason           string `json:"reason"`
}

// TokenExpiringPayload warns the client that the WebSocket auth token is
// about to expire and they should call POST /auth/token/refresh and send a
// refresh.token frame.
type TokenExpiringPayload struct {
	ExpiresAt    string `json:"expires_at"`
	RefreshAfter string `json:"refresh_after"`
}

// RefreshTokenPayload is the inbound frame carrying a freshly-minted access
// token to swap into the connection's metadata.
type RefreshTokenPayload struct {
	AccessToken string `json:"access_token"`
}
