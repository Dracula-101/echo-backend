package request

import (
	sContext "shared/server/context"
	"shared/server/headers"
	"strings"
)

// GetRequestID extracts request ID from headers or context
func (h *RequestHandler) GetRequestID() string {
	if reqID := h.request.Header.Get(headers.XRequestID); reqID != "" {
		return reqID
	}

	if reqID := h.request.Context().Value(sContext.RequestIDKey); reqID != nil {
		if id, ok := reqID.(string); ok {
			return id
		}
	}

	return ""
}

// GetCorrelationID extracts correlation ID from headers or context
func (h *RequestHandler) GetCorrelationID() string {
	if corrID := h.request.Header.Get(headers.XCorrelationID); corrID != "" {
		return corrID
	}

	if corrID := h.request.Context().Value(sContext.CorrelationIDKey); corrID != nil {
		if id, ok := corrID.(string); ok {
			return id
		}
	}

	return ""
}

// GetAuthToken extracts bearer token from Authorization header
func (h *RequestHandler) GetAuthToken() string {
	auth := h.request.Header.Get(headers.Authorization)
	if auth == "" {
		return ""
	}

	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}

	return parts[1]
}

// GetBearerToken extracts bearer token from Authorization header (alias)
func (h *RequestHandler) GetBearerToken() string {
	return h.GetAuthToken()
}

// GetAcceptLanguage extracts Accept-Language header
func (h *RequestHandler) GetAcceptLanguage() string {
	return h.request.Header.Get(headers.AcceptLanguage)
}

// GetPreferredLanguage extracts the preferred language from Accept-Language header
func (h *RequestHandler) GetPreferredLanguage(defaultLang string) string {
	acceptLang := h.GetAcceptLanguage()
	if acceptLang == "" {
		return defaultLang
	}

	langs := strings.Split(acceptLang, ",")
	if len(langs) > 0 {
		lang := strings.TrimSpace(langs[0])
		if idx := strings.Index(lang, ";"); idx != -1 {
			lang = lang[:idx]
		}
		return lang
	}

	return defaultLang
}

// GetReferer extracts Referer header
func (h *RequestHandler) GetReferer() string {
	return h.request.Header.Get(headers.Referer)
}

// GetOrigin extracts Origin header
func (h *RequestHandler) GetOrigin() string {
	return h.request.Header.Get(headers.Origin)
}

// IsWebSocket checks if the request is a WebSocket upgrade request
func (h *RequestHandler) IsWebSocket() bool {
	return strings.ToLower(h.request.Header.Get(headers.Connection)) == "upgrade" &&
		strings.ToLower(h.request.Header.Get(headers.Upgrade)) == "websocket"
}

// GetUserIDFromHeader extracts user ID from context
func (h *RequestHandler) GetUserIDFromHeader() (string, bool) {
	userID, ok := h.request.Context().Value(sContext.UserIDKey).(string)
	return userID, ok
}
