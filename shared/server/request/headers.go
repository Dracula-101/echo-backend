package request

import (
	"net/http"
	sContext "shared/server/context"
	"shared/server/headers"
	"strings"
)

// GetRequestID extracts request ID from headers or context
func GetRequestID(r *http.Request) string {
	if reqID := r.Header.Get(headers.XRequestID); reqID != "" {
		return reqID
	}

	if reqID := r.Context().Value(sContext.RequestIDKey); reqID != nil {
		if id, ok := reqID.(string); ok {
			return id
		}
	}

	return ""
}

// GetCorrelationID extracts correlation ID from headers or context
func GetCorrelationID(r *http.Request) string {
	if corrID := r.Header.Get(headers.XCorrelationID); corrID != "" {
		return corrID
	}

	if corrID := r.Context().Value(sContext.CorrelationIDKey); corrID != nil {
		if id, ok := corrID.(string); ok {
			return id
		}
	}

	return ""
}

// GetAuthToken extracts bearer token from Authorization header
func GetAuthToken(r *http.Request) string {
	auth := r.Header.Get(headers.Authorization)
	if auth == "" {
		return ""
	}

	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}

	return parts[1]
}

// GetAcceptLanguage extracts Accept-Language header
func GetAcceptLanguage(r *http.Request) string {
	return r.Header.Get(headers.AcceptLanguage)
}

// GetPreferredLanguage extracts the preferred language from Accept-Language header
func GetPreferredLanguage(r *http.Request, defaultLang string) string {
	acceptLang := GetAcceptLanguage(r)
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
func GetReferer(r *http.Request) string {
	return r.Header.Get(headers.Referer)
}

// GetOrigin extracts Origin header
func GetOrigin(r *http.Request) string {
	return r.Header.Get(headers.Origin)
}

// IsWebSocket checks if the request is a WebSocket upgrade request
func IsWebSocket(r *http.Request) bool {
	return strings.ToLower(r.Header.Get(headers.Connection)) == "upgrade" &&
		strings.ToLower(r.Header.Get(headers.Upgrade)) == "websocket"
}
