package model

type LocationData struct {
	Latitude   float64 `json:"latitude,omitempty"`
	Longitude  float64 `json:"longitude,omitempty"`
	City       string  `json:"city,omitempty"`
	Continent  string  `json:"continent,omitempty"`
	State      string  `json:"state,omitempty"`
	PostalCode string  `json:"postal_code,omitempty"`
	Country    string  `json:"country,omitempty"`
	Timezone   string  `json:"timezone,omitempty"`
	ISP        string  `json:"isp,omitempty"`
	IP         string  `json:"ip,omitempty"`
}

