package domain

// DeviceInfo represents device metadata
type DeviceInfo struct {
	ID        string
	Type      string
	Name      string
	OS        string
	OSVersion string
	IsMobile  bool
}

// BrowserInfo represents browser metadata
type BrowserInfo struct {
	Name    string
	Version string
}

// LocationInfo represents geographic location data
type LocationInfo struct {
	IP             string
	City           string
	Region         string
	Country        string
	CountryCode    string
	Timezone       string
	Latitude       *float64
	Longitude      *float64
	ISP            string
	Organization   string
	ASN            string
	ConnectionType string
}
