package model

import "fmt"

type LookupResult struct {
	Latitude      float64 `json:"latitude,omitempty"`
	Longitude     float64 `json:"longitude,omitempty"`
	City          string  `json:"city,omitempty"`
	Continent     string  `json:"continent,omitempty"`
	ContinentCode string  `json:"continent_code,omitempty"`
	State         string  `json:"state,omitempty"`
	StateCode     string  `json:"state_code,omitempty"`
	PostalCode    string  `json:"postal_code,omitempty"`
	Country       string  `json:"country,omitempty"`
	CountryCode   string  `json:"country_code,omitempty"`
	Timezone      string  `json:"timezone,omitempty"`
	ISP           string  `json:"isp,omitempty"`
	IP            string  `json:"ip,omitempty"`
}

type City struct {
	Name   string `json:"name,omitempty"`
	Region string `json:"region,omitempty"`
}

// LocationResult contains combined location data from all databases
type LocationResult struct {
	IP      string         `json:"ip"`
	City    *CityRecord    `json:"city,omitempty"`
	Country *CountryRecord `json:"country,omitempty"`
	ASN     *ASNRecord     `json:"asn,omitempty"`
}

func (lr *LocationResult) String() string {
	return fmt.Sprintf("IP: %s, City: %v, Country: %v, ASN: %v", lr.IP, lr.City, lr.Country, lr.ASN)
}

// CityRecord represents GeoIP2/GeoLite2 City database structure
type CityRecord struct {
	City struct {
		Confidence uint              `maxminddb:"confidence" json:"confidence,omitempty"`
		GeoNameID  uint              `maxminddb:"geoname_id" json:"geoname_id,omitempty"`
		Names      map[string]string `maxminddb:"names" json:"names,omitempty"`
	} `maxminddb:"city" json:"city,omitempty"`
	Continent struct {
		Code      string            `maxminddb:"code" json:"code,omitempty"`
		GeoNameID uint              `maxminddb:"geoname_id" json:"geoname_id,omitempty"`
		Names     map[string]string `maxminddb:"names" json:"names,omitempty"`
	} `maxminddb:"continent" json:"continent,omitempty"`
	Country struct {
		Confidence        uint              `maxminddb:"confidence" json:"confidence,omitempty"`
		GeoNameID         uint              `maxminddb:"geoname_id" json:"geoname_id,omitempty"`
		ISOCode           string            `maxminddb:"iso_code" json:"iso_code,omitempty"`
		Names             map[string]string `maxminddb:"names" json:"names,omitempty"`
		IsInEuropeanUnion bool              `maxminddb:"is_in_european_union" json:"is_in_european_union,omitempty"`
	} `maxminddb:"country" json:"country,omitempty"`
	Location *struct {
		AccuracyRadius    uint16  `maxminddb:"accuracy_radius" json:"accuracy_radius,omitempty"`
		AverageIncome     uint    `maxminddb:"average_income" json:"average_income,omitempty"`
		Latitude          float64 `maxminddb:"latitude" json:"latitude,omitempty"`
		Longitude         float64 `maxminddb:"longitude" json:"longitude,omitempty"`
		MetroCode         uint    `maxminddb:"metro_code" json:"metro_code,omitempty"`
		PopulationDensity uint    `maxminddb:"population_density" json:"population_density,omitempty"`
		TimeZone          string  `maxminddb:"time_zone" json:"time_zone,omitempty"`
	} `maxminddb:"location" json:"location,omitempty"`
	Postal struct {
		Code       string `maxminddb:"code" json:"code,omitempty"`
		Confidence uint   `maxminddb:"confidence" json:"confidence,omitempty"`
	} `maxminddb:"postal" json:"postal,omitempty"`
	RegisteredCountry struct {
		GeoNameID         uint              `maxminddb:"geoname_id" json:"geoname_id,omitempty"`
		ISOCode           string            `maxminddb:"iso_code" json:"iso_code,omitempty"`
		Names             map[string]string `maxminddb:"names" json:"names,omitempty"`
		IsInEuropeanUnion bool              `maxminddb:"is_in_european_union" json:"is_in_european_union,omitempty"`
	} `maxminddb:"registered_country" json:"registered_country,omitempty"`
	RepresentedCountry struct {
		GeoNameID         uint              `maxminddb:"geoname_id" json:"geoname_id,omitempty"`
		ISOCode           string            `maxminddb:"iso_code" json:"iso_code,omitempty"`
		Names             map[string]string `maxminddb:"names" json:"names,omitempty"`
		Type              string            `maxminddb:"type" json:"type,omitempty"`
		IsInEuropeanUnion bool              `maxminddb:"is_in_european_union" json:"is_in_european_union,omitempty"`
	} `maxminddb:"represented_country" json:"represented_country,omitempty"`
	Subdivisions []struct {
		Confidence uint              `maxminddb:"confidence" json:"confidence,omitempty"`
		GeoNameID  uint              `maxminddb:"geoname_id" json:"geoname_id,omitempty"`
		ISOCode    string            `maxminddb:"iso_code" json:"iso_code,omitempty"`
		Names      map[string]string `maxminddb:"names" json:"names,omitempty"`
	} `maxminddb:"subdivisions" json:"subdivisions,omitempty"`
	Traits struct {
		AutonomousSystemNumber       uint    `maxminddb:"autonomous_system_number" json:"autonomous_system_number,omitempty"`
		AutonomousSystemOrganization string  `maxminddb:"autonomous_system_organization" json:"autonomous_system_organization,omitempty"`
		ConnectionType               string  `maxminddb:"connection_type" json:"connection_type,omitempty"`
		Domain                       string  `maxminddb:"domain" json:"domain,omitempty"`
		IPAddress                    string  `maxminddb:"ip_address" json:"ip_address,omitempty"`
		IsAnonymous                  bool    `maxminddb:"is_anonymous" json:"is_anonymous,omitempty"`
		IsAnonymousProxy             bool    `maxminddb:"is_anonymous_proxy" json:"is_anonymous_proxy,omitempty"`
		IsAnonymousVPN               bool    `maxminddb:"is_anonymous_vpn" json:"is_anonymous_vpn,omitempty"`
		IsAnycast                    bool    `maxminddb:"is_anycast" json:"is_anycast,omitempty"`
		IsHostingProvider            bool    `maxminddb:"is_hosting_provider" json:"is_hosting_provider,omitempty"`
		IsPublicProxy                bool    `maxminddb:"is_public_proxy" json:"is_public_proxy,omitempty"`
		IsResidentialProxy           bool    `maxminddb:"is_residential_proxy" json:"is_residential_proxy,omitempty"`
		IsSatelliteProvider          bool    `maxminddb:"is_satellite_provider" json:"is_satellite_provider,omitempty"`
		IsTorExitNode                bool    `maxminddb:"is_tor_exit_node" json:"is_tor_exit_node,omitempty"`
		ISP                          string  `maxminddb:"isp" json:"isp,omitempty"`
		MobileCountryCode            string  `maxminddb:"mobile_country_code" json:"mobile_country_code,omitempty"`
		MobileNetworkCode            string  `maxminddb:"mobile_network_code" json:"mobile_network_code,omitempty"`
		Network                      string  `maxminddb:"network" json:"network,omitempty"`
		Organization                 string  `maxminddb:"organization" json:"organization,omitempty"`
		StaticIPScore                float64 `maxminddb:"static_ip_score" json:"static_ip_score,omitempty"`
		UserCount                    uint    `maxminddb:"user_count" json:"user_count,omitempty"`
		UserType                     string  `maxminddb:"user_type" json:"user_type,omitempty"`
	} `maxminddb:"traits" json:"traits,omitempty"`
	MaxMind struct {
		QueriesRemaining uint `maxminddb:"queries_remaining" json:"queries_remaining,omitempty"`
	} `maxminddb:"maxmind" json:"maxmind,omitempty"`
}

// CountryRecord represents GeoIP2/GeoLite2 Country database structure
type CountryRecord struct {
	Continent struct {
		Code      string            `maxminddb:"code" json:"code,omitempty"`
		GeoNameID uint              `maxminddb:"geoname_id" json:"geoname_id,omitempty"`
		Names     map[string]string `maxminddb:"names" json:"names,omitempty"`
	} `maxminddb:"continent" json:"continent,omitempty"`
	Country struct {
		Confidence        uint              `maxminddb:"confidence" json:"confidence,omitempty"`
		GeoNameID         uint              `maxminddb:"geoname_id" json:"geoname_id,omitempty"`
		ISOCode           string            `maxminddb:"iso_code" json:"iso_code,omitempty"`
		Names             map[string]string `maxminddb:"names" json:"names,omitempty"`
		IsInEuropeanUnion bool              `maxminddb:"is_in_european_union" json:"is_in_european_union,omitempty"`
	} `maxminddb:"country" json:"country,omitempty"`
	RegisteredCountry struct {
		GeoNameID         uint              `maxminddb:"geoname_id" json:"geoname_id,omitempty"`
		ISOCode           string            `maxminddb:"iso_code" json:"iso_code,omitempty"`
		Names             map[string]string `maxminddb:"names" json:"names,omitempty"`
		IsInEuropeanUnion bool              `maxminddb:"is_in_european_union" json:"is_in_european_union,omitempty"`
	} `maxminddb:"registered_country" json:"registered_country,omitempty"`
	RepresentedCountry struct {
		GeoNameID         uint              `maxminddb:"geoname_id" json:"geoname_id,omitempty"`
		ISOCode           string            `maxminddb:"iso_code" json:"iso_code,omitempty"`
		Names             map[string]string `maxminddb:"names" json:"names,omitempty"`
		Type              string            `maxminddb:"type" json:"type,omitempty"`
		IsInEuropeanUnion bool              `maxminddb:"is_in_european_union" json:"is_in_european_union,omitempty"`
	} `maxminddb:"represented_country" json:"represented_country,omitempty"`
	Traits struct {
		IPAddress           string  `maxminddb:"ip_address" json:"ip_address,omitempty"`
		IsAnonymous         bool    `maxminddb:"is_anonymous" json:"is_anonymous,omitempty"`
		IsAnonymousProxy    bool    `maxminddb:"is_anonymous_proxy" json:"is_anonymous_proxy,omitempty"`
		IsAnonymousVPN      bool    `maxminddb:"is_anonymous_vpn" json:"is_anonymous_vpn,omitempty"`
		IsAnycast           bool    `maxminddb:"is_anycast" json:"is_anycast,omitempty"`
		IsHostingProvider   bool    `maxminddb:"is_hosting_provider" json:"is_hosting_provider,omitempty"`
		IsPublicProxy       bool    `maxminddb:"is_public_proxy" json:"is_public_proxy,omitempty"`
		IsResidentialProxy  bool    `maxminddb:"is_residential_proxy" json:"is_residential_proxy,omitempty"`
		IsSatelliteProvider bool    `maxminddb:"is_satellite_provider" json:"is_satellite_provider,omitempty"`
		IsTorExitNode       bool    `maxminddb:"is_tor_exit_node" json:"is_tor_exit_node,omitempty"`
		Network             string  `maxminddb:"network" json:"network,omitempty"`
		StaticIPScore       float64 `maxminddb:"static_ip_score" json:"static_ip_score,omitempty"`
		UserCount           uint    `maxminddb:"user_count" json:"user_count,omitempty"`
		UserType            string  `maxminddb:"user_type" json:"user_type,omitempty"`
	} `maxminddb:"traits" json:"traits,omitempty"`
	MaxMind struct {
		QueriesRemaining uint `maxminddb:"queries_remaining" json:"queries_remaining,omitempty"`
	} `maxminddb:"maxmind" json:"maxmind,omitempty"`
}

// ASNRecord represents GeoLite2 ASN database structure
type ASNRecord struct {
	AutonomousSystemNumber       uint   `maxminddb:"autonomous_system_number" json:"autonomous_system_number,omitempty"`
	AutonomousSystemOrganization string `maxminddb:"autonomous_system_organization" json:"autonomous_system_organization,omitempty"`
	IPAddress                    string `maxminddb:"ip_address" json:"ip_address,omitempty"`
	Network                      string `maxminddb:"network" json:"network,omitempty"`
}

func (c *CityRecord) GetCityName(lang string) string {
	if lang == "" {
		lang = "en"
	}
	return c.City.Names[lang]
}

// GetCountryName returns the country name in the specified language (default "en")
func (c *CityRecord) GetCountryName(lang string) string {
	if lang == "" {
		lang = "en"
	}
	return c.Country.Names[lang]
}

// GetSubdivisionName returns the first subdivision name (e.g., state/province)
func (c *CityRecord) GetSubdivisionName(lang string) string {
	if lang == "" {
		lang = "en"
	}
	if len(c.Subdivisions) > 0 {
		return c.Subdivisions[0].Names[lang]
	}
	return ""
}

func (c *CityRecord) GetSubdivisionISOCode() string {
	if len(c.Subdivisions) > 0 {
		return c.Subdivisions[0].ISOCode
	}
	return ""
}

// GetContinentName returns the continent name in the specified language (default "en")
func (c *CityRecord) GetContinentName(lang string) string {
	if lang == "" {
		lang = "en"
	}
	return c.Continent.Names[lang]
}
