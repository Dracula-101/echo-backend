package domain

type LocationData struct {
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
