package request

import (
	"context"
	"net"
	"net/http"
	sContext "shared/server/context"
	"shared/server/headers"
	"strings"

	"github.com/google/uuid"
)

func GetClientIP(r *http.Request) string {
	if xff := r.Header.Get(headers.XForwardedFor); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	if xri := r.Header.Get(headers.XRealIP); xri != "" {
		return xri
	}

	if cfIP := r.Header.Get(headers.XCFConnectingIP); cfIP != "" {
		return cfIP
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return ip
}

func GetUserAgent(r *http.Request) string {
	return r.Header.Get(headers.UserAgent)
}

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

type DeviceInfo struct {
	ID           string
	Name         string
	Type         string
	Platform     string
	OS           string
	OsVersion    string
	Model        string
	Manufacturer string
}

type BrowserInfo struct {
	Name    string
	Version string
}

type IpAddressInfo struct {
	Country   string
	Region    string
	City      string
	Timezone  string
	ISP       string
	IP        string
	Latitude  float64
	Longitude float64
}

func GetDeviceInfo(r *http.Request) DeviceInfo {
	id := r.Header.Get(headers.XDeviceID)
	if id == "" {
		id = uuid.NewString()
	}
	name := r.Header.Get(headers.XDeviceName)
	if name == "" {
		name = "Unknown Device"
	}
	deviceType := r.Header.Get(headers.XDeviceType)
	if deviceType == "" {
		deviceType = "unknown"
	}
	platform := r.Header.Get(headers.XDevicePlatform)
	if platform == "" {
		platform = "unknown"
	}
	os := r.Header.Get(headers.XDeviceOS)
	if os == "" {
		os = "unknown"
	}
	osVersion := r.Header.Get(headers.XDeviceOSVersion)
	if osVersion == "" {
		osVersion = "unknown"
	}
	model := r.Header.Get(headers.XDeviceModel)
	if model == "" {
		model = "unknown"
	}
	manufacturer := r.Header.Get(headers.XDeviceManufacturer)
	if manufacturer == "" {
		manufacturer = "unknown"
	}

	return DeviceInfo{
		ID:           id,
		Name:         name,
		Type:         deviceType,
		Platform:     platform,
		OS:           os,
		OsVersion:    osVersion,
		Model:        model,
		Manufacturer: manufacturer,
	}
}

func GetBrowserInfo(r *http.Request) BrowserInfo {
	return BrowserInfo{
		Name:    r.Header.Get(headers.XBrowserName),
		Version: r.Header.Get(headers.XBrowserVersion),
	}
}

func IsWebSocket(r *http.Request) bool {
	return strings.ToLower(r.Header.Get(headers.Connection)) == "upgrade" &&
		strings.ToLower(r.Header.Get(headers.Upgrade)) == "websocket"
}

func GetAcceptLanguage(r *http.Request) string {
	return r.Header.Get(headers.AcceptLanguage)
}

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

func GetReferer(r *http.Request) string {
	return r.Header.Get(headers.Referer)
}

func GetOrigin(r *http.Request) string {
	return r.Header.Get(headers.Origin)
}

func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, sContext.UserIDKey, userID)
}

func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(sContext.UserIDKey).(string)
	return userID, ok
}

func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, sContext.SessionIDKey, sessionID)
}

func GetSessionIDFromContext(ctx context.Context) (string, bool) {
	sessionID, ok := ctx.Value(sContext.SessionIDKey).(string)
	return sessionID, ok
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, sContext.RequestIDKey, requestID)
}

func GetRequestIDFromContext(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(sContext.RequestIDKey).(string)
	return requestID, ok
}

func WithClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, sContext.ClientIPKey, ip)
}

func GetClientIPFromContext(ctx context.Context) (string, bool) {
	ip, ok := ctx.Value(sContext.ClientIPKey).(string)
	return ip, ok
}
