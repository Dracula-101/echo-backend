package request

import (
	"net"
	"net/http"
	"shared/server/headers"
	"strings"

	"github.com/google/uuid"
)

// GetClientIP extracts the client IP address from the request
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

// GetUserAgent extracts the user agent from the request
func GetUserAgent(r *http.Request) string {
	return r.Header.Get(headers.UserAgent)
}

// DeviceInfo contains device information
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

// GetDeviceInfo extracts device information from request headers
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

// IsMobile returns true if the device is mobile
func (d DeviceInfo) IsMobile() bool {
	mobileOS := []string{"iOS", "Android", "Windows Phone", "BlackBerry", "Symbian"}
	for _, os := range mobileOS {
		if strings.EqualFold(d.OS, os) {
			return true
		}
	}
	return false
}

// BrowserInfo contains browser information
type BrowserInfo struct {
	Name    string
	Version string
}

// GetBrowserInfo extracts browser information from request headers
func GetBrowserInfo(r *http.Request) BrowserInfo {
	return BrowserInfo{
		Name:    r.Header.Get(headers.XBrowserName),
		Version: r.Header.Get(headers.XBrowserVersion),
	}
}

// IpAddressInfo contains IP address geolocation information
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
