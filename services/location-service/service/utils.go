package service

// adjust the import path for dbModels if necessary
import (
	"encoding/json"
	"fmt"
	"location-service/model"
	"net"
	dbModels "shared/pkg/database/postgres/models"
	"strconv"
	"time"
)

// helper constructors
func strPtr(s string) *string     { return &s }
func intPtr(i int) *int           { return &i }
func floatPtr(f float64) *float64 { return &f }

func generateModelFromLocationResult(ipStr string, city *model.CityRecord, country *model.CountryRecord, asn *model.ASNRecord) *dbModels.IPAddress {
	now := time.Now()

	var ipVersion *int
	if p := net.ParseIP(ipStr); p != nil {
		if p.To4() != nil {
			ipVersion = intPtr(4)
		} else {
			ipVersion = intPtr(6)
		}
	}

	var countryCode *string
	var countryName *string
	if city != nil && city.Country.ISOCode != "" {
		countryCode = strPtr(city.Country.ISOCode)
		if n, ok := city.Country.Names["en"]; ok && n != "" {
			countryName = strPtr(n)
		}
	}
	if countryCode == nil && country != nil && country.Country.ISOCode != "" {
		countryCode = strPtr(country.Country.ISOCode)
	}
	if countryName == nil && country != nil {
		if n, ok := country.Country.Names["en"]; ok && n != "" {
			countryName = strPtr(n)
		}
	}

	var regionCode *string
	var regionName *string
	if city != nil {
		if subISO := city.GetSubdivisionISOCode(); subISO != "" {
			regionCode = strPtr(subISO)
		}
		if subName := city.GetSubdivisionName("en"); subName != "" {
			regionName = strPtr(subName)
		}
	}

	var cityName *string
	var postalCode *string
	var latitude *float64
	var longitude *float64
	var metroCode *string
	var timezone *string

	if city != nil {
		if name := city.GetCityName("en"); name != "" {
			cityName = strPtr(name)
		}
		if city.Postal.Code != "" {
			postalCode = strPtr(city.Postal.Code)
		}
		if city.Location != nil {
			if city.Location.Latitude != 0 || city.Location.Longitude != 0 {
				latitude = floatPtr(city.Location.Latitude)
				longitude = floatPtr(city.Location.Longitude)
			}
			if city.Location.MetroCode != 0 {
				metroCode = strPtr(fmt.Sprintf("%d", city.Location.MetroCode))
			}
			if city.Location.TimeZone != "" {
				timezone = strPtr(city.Location.TimeZone)
			}
		}
	}

	var isp *string
	var organization *string
	var connectionType *string
	var userType *string
	var lookupCount int
	var userCount int
	var isProxy bool
	var isVPN bool
	var isTor bool
	var isHosting bool
	var isAnonymous bool

	if city != nil {
		t := city.Traits
		if t.ISP != "" {
			isp = strPtr(t.ISP)
		}
		if t.Organization != "" {
			organization = strPtr(t.Organization)
		}
		if t.ConnectionType != "" {
			connectionType = strPtr(t.ConnectionType)
		}
		if t.UserType != "" {
			userType = strPtr(t.UserType)
		}
		lookupCount = 0
		userCount = int(t.UserCount)
		// proxies / vpn / tor / hosting / anonymous
		isProxy = t.IsAnonymousProxy || t.IsPublicProxy || t.IsResidentialProxy
		isVPN = t.IsAnonymousVPN
		isTor = t.IsTorExitNode
		isHosting = t.IsHostingProvider
		isAnonymous = t.IsAnonymous || t.IsAnonymousProxy || t.IsAnonymousVPN
	}

	// ASN fields
	var asnStr *string
	var asnOrg *string
	if asn != nil {
		if asn.AutonomousSystemNumber != 0 {
			asnStr = strPtr(fmt.Sprintf("%d", asn.AutonomousSystemNumber))
		}
		if asn.AutonomousSystemOrganization != "" {
			asnOrg = strPtr(asn.AutonomousSystemOrganization)
		}
	}

	// choose lookup provider
	var lookupProvider *string
	if city != nil || country != nil || asn != nil {
		lp := "maxmind"
		lookupProvider = &lp
	}

	// metadata: marshal the raw records into JSON (safe to ignore errors)
	metaMap := map[string]interface{}{}
	var ipModelMetadata json.RawMessage
	if country != nil {
		metaMap["continent"] = country.Continent
		metaMap["continent_code"] = country.Continent.Code
		metaMap["geoname_id"] = country.Continent.GeoNameID
	}
	if metaBytes, err := json.Marshal(metaMap); err == nil {
		ipModelMetadata = json.RawMessage(metaBytes)
	}

	ipModel := &dbModels.IPAddress{
		IPAddress:      ipStr,
		IPVersion:      ipVersion,
		CountryCode:    countryCode,
		CountryName:    countryName,
		RegionCode:     regionCode,
		RegionName:     regionName,
		City:           cityName,
		PostalCode:     postalCode,
		Latitude:       latitude,
		Longitude:      longitude,
		MetroCode:      metroCode,
		Timezone:       timezone,
		ISP:            isp,
		Organization:   organization,
		ASN:            asnStr,
		ASOrganization: asnOrg,
		ConnectionType: connectionType,
		UserType:       userType,
		IsProxy:        isProxy,
		IsVPN:          isVPN,
		IsTor:          isTor,
		IsHosting:      isHosting,
		IsAnonymous:    isAnonymous,
		ThreatLevel:    nil, // leave unset unless you have a mapping
		RiskScore:      nil, // leave unset unless you compute one
		IsBogon:        false,
		FirstSeenAt:    now,
		LastSeenAt:     now,
		LookupCount:    lookupCount,
		UserCount:      userCount,
		LookupProvider: lookupProvider,
		LastUpdatedAt:  now,
		CreatedAt:      now,
		Metadata:       ipModelMetadata,
	}

	return ipModel
}

func generateLocationResultFromModel(ipModel *dbModels.IPAddress) *model.LocationResult {
	if ipModel == nil {
		return nil
	}

	result := &model.LocationResult{
		IP:      ipModel.IPAddress,
		City:    &model.CityRecord{},
		Country: &model.CountryRecord{},
		ASN:     &model.ASNRecord{},
	}

	// --- City ---
	if result.City != nil {
		if ipModel.City != nil {
			result.City.City.Names = map[string]string{"en": *ipModel.City}
		}
		if ipModel.RegionName != nil {
			sub := struct {
				Confidence uint              `maxminddb:"confidence" json:"confidence,omitempty"`
				GeoNameID  uint              `maxminddb:"geoname_id" json:"geoname_id,omitempty"`
				ISOCode    string            `maxminddb:"iso_code" json:"iso_code,omitempty"`
				Names      map[string]string `maxminddb:"names" json:"names,omitempty"`
			}{
				Names: map[string]string{"en": *ipModel.RegionName},
			}
			if ipModel.RegionCode != nil {
				sub.ISOCode = *ipModel.RegionCode
			}
			result.City.Subdivisions = []struct {
				Confidence uint              `maxminddb:"confidence" json:"confidence,omitempty"`
				GeoNameID  uint              `maxminddb:"geoname_id" json:"geoname_id,omitempty"`
				ISOCode    string            `maxminddb:"iso_code" json:"iso_code,omitempty"`
				Names      map[string]string `maxminddb:"names" json:"names,omitempty"`
			}{sub}
		}

		if ipModel.PostalCode != nil {
			result.City.Postal.Code = *ipModel.PostalCode
		}

		if ipModel.Latitude != nil || ipModel.Longitude != nil || ipModel.Timezone != nil {
			result.City.Location = &struct {
				AccuracyRadius    uint16  `maxminddb:"accuracy_radius" json:"accuracy_radius,omitempty"`
				AverageIncome     uint    `maxminddb:"average_income" json:"average_income,omitempty"`
				Latitude          float64 `maxminddb:"latitude" json:"latitude,omitempty"`
				Longitude         float64 `maxminddb:"longitude" json:"longitude,omitempty"`
				MetroCode         uint    `maxminddb:"metro_code" json:"metro_code,omitempty"`
				PopulationDensity uint    `maxminddb:"population_density" json:"population_density,omitempty"`
				TimeZone          string  `maxminddb:"time_zone" json:"time_zone,omitempty"`
			}{}

			if ipModel.Latitude != nil {
				result.City.Location.Latitude = *ipModel.Latitude
			}
			if ipModel.Longitude != nil {
				result.City.Location.Longitude = *ipModel.Longitude
			}
			if ipModel.Timezone != nil {
				result.City.Location.TimeZone = *ipModel.Timezone
			}
		}

		// isp & network traits
		result.City.Traits.ISP = derefString(ipModel.ISP)
		result.City.Traits.Organization = derefString(ipModel.Organization)
		result.City.Traits.UserType = derefString(ipModel.UserType)
		result.City.Traits.ConnectionType = derefString(ipModel.ConnectionType)

		if ipModel.ASOrganization != nil {
			result.City.Traits.AutonomousSystemOrganization = *ipModel.ASOrganization
		}
	}

	// --- Country ---
	if ipModel.CountryCode != nil || ipModel.CountryName != nil {
		if ipModel.CountryCode != nil {
			result.Country.Country.ISOCode = *ipModel.CountryCode
		}
		if ipModel.CountryName != nil {
			result.Country.Country.Names = map[string]string{"en": *ipModel.CountryName}
		}
	}

	// --- ASN ---
	if ipModel.ASN != nil {
		num, _ := strconv.Atoi(*ipModel.ASN)
		result.ASN.AutonomousSystemNumber = uint(num)
	}
	if ipModel.ASOrganization != nil {
		result.ASN.AutonomousSystemOrganization = *ipModel.ASOrganization
	}
	result.ASN.IPAddress = ipModel.IPAddress

	return result
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
