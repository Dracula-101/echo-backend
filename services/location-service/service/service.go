package service

import (
	"fmt"
	"location-service/model"
	"net"
	"sync"

	"github.com/oschwald/maxminddb-golang"
)

type LocationService struct {
	cityDB    *maxminddb.Reader
	asnDB     *maxminddb.Reader
	countryDB *maxminddb.Reader
	mu        sync.RWMutex
}

type Config struct {
	CityDBPath    string
	ASNDBPath     string
	CountryDBPath string
}

func NewLocationService(cfg Config) (*LocationService, error) {
	svc := &LocationService{}
	var err error

	if cfg.CityDBPath != "" {
		svc.cityDB, err = maxminddb.Open(cfg.CityDBPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open city DB: %w", err)
		}
	}

	if cfg.ASNDBPath != "" {
		svc.asnDB, err = maxminddb.Open(cfg.ASNDBPath)
		if err != nil {
			svc.Close()
			return nil, fmt.Errorf("failed to open ASN DB: %w", err)
		}
	}

	if cfg.CountryDBPath != "" {
		svc.countryDB, err = maxminddb.Open(cfg.CountryDBPath)
		if err != nil {
			svc.Close()
			return nil, fmt.Errorf("failed to open country DB: %w", err)
		}
	}

	return svc, nil
}

func (s *LocationService) Lookup(ipStr string) (*model.LocationResult, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ipStr)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	result := &model.LocationResult{IP: ipStr}

	if s.cityDB != nil {
		var city model.CityRecord
		if err := s.cityDB.Lookup(ip, &city); err == nil {
			result.City = &city
		}
	}

	if s.countryDB != nil {
		var country model.CountryRecord
		if err := s.countryDB.Lookup(ip, &country); err == nil {
			result.Country = &country
		}
	}

	if s.asnDB != nil {
		var asn model.ASNRecord
		if err := s.asnDB.Lookup(ip, &asn); err == nil {
			result.ASN = &asn
		}
	}

	return result, nil
}

func (s *LocationService) LookupCity(ipStr string) (*model.CityRecord, error) {
	if s.cityDB == nil {
		return nil, fmt.Errorf("city database not loaded")
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ipStr)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var city model.CityRecord
	if err := s.cityDB.Lookup(ip, &city); err != nil {
		return nil, fmt.Errorf("city lookup failed: %w", err)
	}

	return &city, nil
}

func (s *LocationService) LookupASN(ipStr string) (*model.ASNRecord, error) {
	if s.asnDB == nil {
		return nil, fmt.Errorf("ASN database not loaded")
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ipStr)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var asn model.ASNRecord
	if err := s.asnDB.Lookup(ip, &asn); err != nil {
		return nil, fmt.Errorf("ASN lookup failed: %w", err)
	}

	return &asn, nil
}

func (s *LocationService) LookupCountry(ipStr string) (*model.CountryRecord, error) {
	if s.countryDB == nil {
		return nil, fmt.Errorf("country database not loaded")
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ipStr)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var country model.CountryRecord
	if err := s.countryDB.Lookup(ip, &country); err != nil {
		return nil, fmt.Errorf("country lookup failed: %w", err)
	}

	return &country, nil
}

func (s *LocationService) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var errs []error

	if s.cityDB != nil {
		if err := s.cityDB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("city DB close: %w", err))
		}
	}

	if s.asnDB != nil {
		if err := s.asnDB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("ASN DB close: %w", err))
		}
	}

	if s.countryDB != nil {
		if err := s.countryDB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("country DB close: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing databases: %v", errs)
	}

	return nil
}
