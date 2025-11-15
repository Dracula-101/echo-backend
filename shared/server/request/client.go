package request

import (
	"net"
	"shared/server/headers"
	"strings"
)

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

// IpAddressInfo contains IP address geolocation information
type IpAddressInfo struct {
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`
	City          string  `json:"city"`
	Continent     string  `json:"continent"`
	ContinentCode string  `json:"continent_code"`
	State         string  `json:"state"`
	StateCode     string  `json:"state_code"`
	PostalCode    string  `json:"postal_code"`
	Country       string  `json:"country"`
	CountryCode   string  `json:"country_code"`
	Timezone      string  `json:"timezone"`
	ISP           string  `json:"isp"`
	IP            string  `json:"ip"`
}

// GetClientIP extracts the client IP address from the request
func (h *RequestHandler) GetClientIP() string {
	if xff := h.request.Header.Get(headers.XForwardedFor); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	if xri := h.request.Header.Get(headers.XRealIP); xri != "" {
		return xri
	}

	if cfIP := h.request.Header.Get(headers.XCFConnectingIP); cfIP != "" {
		return cfIP
	}

	ip, _, err := net.SplitHostPort(h.request.RemoteAddr)
	if err != nil {
		return h.request.RemoteAddr
	}

	return ip
}

// GetUserAgent extracts the user agent from the request
func (h *RequestHandler) GetUserAgent() string {
	return h.request.Header.Get(headers.UserAgent)
}

// GetDeviceInfo extracts device information from request headers
func (h *RequestHandler) GetDeviceInfo() DeviceInfo {
	id := h.request.Header.Get(headers.XDeviceID)
	if id == "" {
		id = "device-id"
	}
	name := h.request.Header.Get(headers.XDeviceName)
	if name == "" {
		name = "Unknown Device"
	}
	deviceType := h.request.Header.Get(headers.XDeviceType)
	if deviceType == "" {
		deviceType = "unknown"
	}
	platform := h.request.Header.Get(headers.XDevicePlatform)
	if platform == "" {
		platform = "unknown"
	}
	os := h.request.Header.Get(headers.XDeviceOS)
	if os == "" {
		os = "unknown"
	}
	osVersion := h.request.Header.Get(headers.XDeviceOSVersion)
	if osVersion == "" {
		osVersion = "unknown"
	}
	model := h.request.Header.Get(headers.XDeviceModel)
	if model == "" {
		model = "unknown"
	}
	manufacturer := h.request.Header.Get(headers.XDeviceManufacturer)
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

// GetBrowserInfo extracts browser information from request headers
func (h *RequestHandler) GetBrowserInfo() BrowserInfo {
	return BrowserInfo{
		Name:    h.request.Header.Get(headers.XBrowserName),
		Version: h.request.Header.Get(headers.XBrowserVersion),
	}
}
