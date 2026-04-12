package location

import (
	"auth-service/internal/domain"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	pkgErrors "shared/pkg/errors"
	"shared/pkg/logger"
	"shared/server/request"
	"shared/server/response"
)

// LocationServiceInterface defines the contract for location service operations
type LocationServiceInterface interface {
	// IP lookup
	Lookup(ip string) (*request.IpAddressInfo, pkgErrors.AppError)
}

func (s *LocationService) Lookup(ip string) (*request.IpAddressInfo, pkgErrors.AppError) {
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

	if resp.StatusCode != response.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, pkgErrors.New(pkgErrors.CodeServiceUnavailable, "location lookup request failed").
			WithDetail("status_code", resp.StatusCode).
			WithDetail("response_body", string(body)).
			WithDetail("ip", ip)
	}

	var locationData domain.LocationData
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&locationData); err != nil {
		return nil, pkgErrors.FromError(err, pkgErrors.CodeInternal, "failed to decode location response").
			WithDetail("ip", ip)
	}

	return &request.IpAddressInfo{
		Latitude:    locationData.Latitude,
		Longitude:   locationData.Longitude,
		City:        locationData.City,
		State:       locationData.State,
		StateCode:   locationData.StateCode,
		PostalCode:  locationData.PostalCode,
		Country:     locationData.Country,
		CountryCode: locationData.CountryCode,
		Timezone:    locationData.Timezone,
		ISP:         locationData.ISP,
		IP:          locationData.IP,
	}, nil
}
