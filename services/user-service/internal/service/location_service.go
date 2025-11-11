package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/server/request"
	"time"
)

type LocationService struct {
	Endpoint string
	client   *http.Client
	log      logger.Logger
}

func NewLocationService(endpoint string, log logger.Logger) *LocationService {
	return &LocationService{
		Endpoint: endpoint,
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		log: log,
	}
}

func (s *LocationService) Lookup(ip string) (*request.IpAddressInfo, error) {
	if ip == "" {
		return nil, pkgErrors.New(pkgErrors.CodeInvalidArgument, "ip address is required")
	}

	url := fmt.Sprintf("%s?ip=%s", s.Endpoint, url.QueryEscape(ip))
	s.log.Info("Looking up location", logger.String("url", url))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeInternal, "failed to create location lookup request").
			WithDetail("ip", ip)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeServiceUnavailable, "failed to execute location lookup request").
			WithDetail("ip", ip)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, pkgErrors.New(pkgErrors.CodeServiceUnavailable, "location lookup request failed").
			WithDetail("status_code", resp.StatusCode).
			WithDetail("response_body", string(body)).
			WithDetail("ip", ip)
	}

	var locationData LocationData
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&locationData); err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeInternal, "failed to decode location response").
			WithDetail("ip", ip)
	}

	return &request.IpAddressInfo{
		Latitude:      locationData.Latitude,
		Longitude:     locationData.Longitude,
		City:          locationData.City,
		Continent:     locationData.Continent,
		ContinentCode: locationData.ContinentCode,
		State:         locationData.State,
		StateCode:     locationData.StateCode,
		PostalCode:    locationData.PostalCode,
		Country:       locationData.Country,
		CountryCode:   locationData.CountryCode,
		Timezone:      locationData.Timezone,
		ISP:           locationData.ISP,
		IP:            locationData.IP,
	}, nil
}

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
