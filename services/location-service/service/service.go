package service

import (
	"context"
	locErrors "location-service/errors"
	"location-service/model"
	"net"
	"shared/pkg/database"
	"shared/pkg/database/postgres"
	dbModels "shared/pkg/database/postgres/models"
	"shared/pkg/logger"
	"sync"

	"github.com/oschwald/maxminddb-golang"
)

// ============================================================================
// Service Definition
// ============================================================================

type LocationService struct {
	cityDB    *maxminddb.Reader
	asnDB     *maxminddb.Reader
	countryDB *maxminddb.Reader
	db        database.Database
	mu        sync.RWMutex
	log       logger.Logger
}

type Config struct {
	CityDBPath    string
	ASNDBPath     string
	CountryDBPath string
	Logger        logger.Logger
}

func NewLocationService(cfg Config, dbConfig database.Config) (*LocationService, error) {
	if cfg.Logger == nil {
		panic("Logger is required for LocationService")
	}

	cfg.Logger.Info("Initializing LocationService",
		logger.String("service", locErrors.ServiceName),
		logger.String("city_db", cfg.CityDBPath),
		logger.String("asn_db", cfg.ASNDBPath),
		logger.String("country_db", cfg.CountryDBPath),
	)

	cfg.Logger.Debug("Connecting to database",
		logger.String("service", locErrors.ServiceName),
		logger.String("host", dbConfig.Host),
		logger.Int("port", dbConfig.Port),
		logger.String("database", dbConfig.Database),
	)

	db, dbErr := postgres.New(dbConfig)
	if dbErr != nil {
		cfg.Logger.Error("Failed to connect to database",
			logger.String("service", locErrors.ServiceName),
			logger.Error(dbErr),
		)
		return nil, dbErr
	}

	cfg.Logger.Info("Database connection established",
		logger.String("service", locErrors.ServiceName),
	)

	svc := &LocationService{
		db:  db,
		log: cfg.Logger,
	}
	var err error

	if cfg.CityDBPath != "" {
		cfg.Logger.Debug("Loading city database",
			logger.String("service", locErrors.ServiceName),
			logger.String("path", cfg.CityDBPath),
		)
		svc.cityDB, err = maxminddb.Open(cfg.CityDBPath)
		if err != nil {
			cfg.Logger.Error("Failed to open city database",
				logger.String("service", locErrors.ServiceName),
				logger.String("path", cfg.CityDBPath),
				logger.String("error_code", locErrors.CodeDatabaseLoadFailed),
				logger.Error(err),
			)
			return nil, err
		}
		cfg.Logger.Info("City database loaded successfully",
			logger.String("service", locErrors.ServiceName),
		)
	}

	if cfg.ASNDBPath != "" {
		cfg.Logger.Debug("Loading ASN database",
			logger.String("service", locErrors.ServiceName),
			logger.String("path", cfg.ASNDBPath),
		)
		svc.asnDB, err = maxminddb.Open(cfg.ASNDBPath)
		if err != nil {
			cfg.Logger.Error("Failed to open ASN database",
				logger.String("service", locErrors.ServiceName),
				logger.String("path", cfg.ASNDBPath),
				logger.String("error_code", locErrors.CodeDatabaseLoadFailed),
				logger.Error(err),
			)
			svc.Close()
			return nil, err
		}
		cfg.Logger.Info("ASN database loaded successfully",
			logger.String("service", locErrors.ServiceName),
		)
	}

	if cfg.CountryDBPath != "" {
		cfg.Logger.Debug("Loading country database",
			logger.String("service", locErrors.ServiceName),
			logger.String("path", cfg.CountryDBPath),
		)
		svc.countryDB, err = maxminddb.Open(cfg.CountryDBPath)
		if err != nil {
			cfg.Logger.Error("Failed to open country database",
				logger.String("service", locErrors.ServiceName),
				logger.String("path", cfg.CountryDBPath),
				logger.String("error_code", locErrors.CodeDatabaseLoadFailed),
				logger.Error(err),
			)
			svc.Close()
			return nil, err
		}
		cfg.Logger.Info("Country database loaded successfully",
			logger.String("service", locErrors.ServiceName),
		)
	}

	cfg.Logger.Info("LocationService initialized successfully",
		logger.String("service", locErrors.ServiceName),
	)

	return svc, nil
}

// ============================================================================
// Lookup Operations
// ============================================================================

func (s *LocationService) Lookup(ipStr string) (*model.LocationResult, error) {
	s.log.Debug("Performing location lookup",
		logger.String("service", locErrors.ServiceName),
		logger.String("ip", ipStr),
	)

	ip := net.ParseIP(ipStr)
	if ip == nil {
		s.log.Warn("Invalid IP address provided",
			logger.String("service", locErrors.ServiceName),
			logger.String("ip", ipStr),
			logger.String("error_code", locErrors.CodeInvalidIP),
		)
		return nil, locErrors.NewLocationError(locErrors.CodeInvalidIP, "Invalid IP address format")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	result := &model.LocationResult{IP: ipStr}

	// check from db
	var ipModel dbModels.IPAddress
	query := `SELECT * FROM location.ip_addresses WHERE ip_address = $1 LIMIT 1`
	err := s.db.FindOne(context.Background(), &ipModel, query, ipStr)
	if err == nil {
		result = generateLocationResultFromModel(&ipModel)
		return result, nil
	}

	s.log.Debug("IP address queried from database",
		logger.String("service", locErrors.ServiceName),
		logger.String("ip", ipStr),
		logger.Any("ip_model", ipModel),
	)

	if s.cityDB != nil {
		var city model.CityRecord
		if err := s.cityDB.Lookup(ip, &city); err != nil {
			s.log.Warn("City lookup failed for IP",
				logger.String("service", locErrors.ServiceName),
				logger.String("ip", ipStr),
				logger.Error(err),
			)
		} else {
			s.log.Debug("City data found",
				logger.String("service", locErrors.ServiceName),
				logger.String("ip", ipStr),
			)
			result.City = &city
		}
	}

	if s.countryDB != nil {
		var country model.CountryRecord
		if err := s.countryDB.Lookup(ip, &country); err != nil {
			s.log.Warn("Country lookup failed for IP",
				logger.String("service", locErrors.ServiceName),
				logger.String("ip", ipStr),
				logger.Error(err),
			)
		} else {
			s.log.Debug("Country data found",
				logger.String("service", locErrors.ServiceName),
				logger.String("ip", ipStr),
			)
			result.Country = &country
		}
	}

	if s.asnDB != nil {
		var asn model.ASNRecord
		if err := s.asnDB.Lookup(ip, &asn); err != nil {
			s.log.Warn("ASN lookup failed for IP",
				logger.String("service", locErrors.ServiceName),
				logger.String("ip", ipStr),
				logger.Error(err),
			)
		} else {
			s.log.Debug("ASN data found",
				logger.String("service", locErrors.ServiceName),
				logger.String("ip", ipStr),
			)
			result.ASN = &asn
		}
	}

	s.log.Info("Location lookup completed",
		logger.String("service", locErrors.ServiceName),
		logger.String("ip", ipStr),
		logger.Bool("has_city", result.City != nil),
		logger.Bool("has_country", result.Country != nil),
		logger.Bool("has_asn", result.ASN != nil),
	)

	m := generateModelFromLocationResult(ipStr, result.City, result.Country, result.ASN)
	_, dbErr := s.db.Insert(context.Background(), m)
	if dbErr != nil {
		s.log.Warn("Failed to store IP address lookup in database",
			logger.String("service", locErrors.ServiceName),
			logger.String("ip", ipStr),
			logger.Error(dbErr),
		)
	} else {
		s.log.Debug("Stored IP address lookup in database",
			logger.String("service", locErrors.ServiceName),
			logger.String("ip", ipStr),
		)
	}

	return result, nil
}

func (s *LocationService) LookupCity(ipStr string) (*model.CityRecord, error) {
	s.log.Debug("Performing city lookup",
		logger.String("service", locErrors.ServiceName),
		logger.String("ip", ipStr),
	)

	if s.cityDB == nil {
		s.log.Error("City database not loaded",
			logger.String("service", locErrors.ServiceName),
			logger.String("error_code", locErrors.CodeDatabaseNotFound),
		)
		return nil, locErrors.NewLocationError(locErrors.CodeDatabaseNotFound, "City database not loaded")
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		s.log.Warn("Invalid IP address for city lookup",
			logger.String("service", locErrors.ServiceName),
			logger.String("ip", ipStr),
			logger.String("error_code", locErrors.CodeInvalidIP),
		)
		return nil, locErrors.NewLocationError(locErrors.CodeInvalidIP, "Invalid IP address format")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var city model.CityRecord
	if err := s.cityDB.Lookup(ip, &city); err != nil {
		s.log.Error("City lookup failed",
			logger.String("service", locErrors.ServiceName),
			logger.String("ip", ipStr),
			logger.String("error_code", locErrors.CodeLookupFailed),
			logger.Error(err),
		)
		return nil, err
	}

	s.log.Info("City lookup successful",
		logger.String("service", locErrors.ServiceName),
		logger.String("ip", ipStr),
	)

	return &city, nil
}

func (s *LocationService) LookupASN(ipStr string) (*model.ASNRecord, error) {
	s.log.Debug("Performing ASN lookup",
		logger.String("service", locErrors.ServiceName),
		logger.String("ip", ipStr),
	)

	if s.asnDB == nil {
		s.log.Error("ASN database not loaded",
			logger.String("service", locErrors.ServiceName),
			logger.String("error_code", locErrors.CodeDatabaseNotFound),
		)
		return nil, locErrors.NewLocationError(locErrors.CodeDatabaseNotFound, "ASN database not loaded")
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		s.log.Warn("Invalid IP address for ASN lookup",
			logger.String("service", locErrors.ServiceName),
			logger.String("ip", ipStr),
			logger.String("error_code", locErrors.CodeInvalidIP),
		)
		return nil, locErrors.NewLocationError(locErrors.CodeInvalidIP, "Invalid IP address format")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var asn model.ASNRecord
	if err := s.asnDB.Lookup(ip, &asn); err != nil {
		s.log.Error("ASN lookup failed",
			logger.String("service", locErrors.ServiceName),
			logger.String("ip", ipStr),
			logger.String("error_code", locErrors.CodeLookupFailed),
			logger.Error(err),
		)
		return nil, err
	}

	s.log.Info("ASN lookup successful",
		logger.String("service", locErrors.ServiceName),
		logger.String("ip", ipStr),
	)

	return &asn, nil
}

func (s *LocationService) LookupCountry(ipStr string) (*model.CountryRecord, error) {
	s.log.Debug("Performing country lookup",
		logger.String("service", locErrors.ServiceName),
		logger.String("ip", ipStr),
	)

	if s.countryDB == nil {
		s.log.Error("Country database not loaded",
			logger.String("service", locErrors.ServiceName),
			logger.String("error_code", locErrors.CodeDatabaseNotFound),
		)
		return nil, locErrors.NewLocationError(locErrors.CodeDatabaseNotFound, "Country database not loaded")
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		s.log.Warn("Invalid IP address for country lookup",
			logger.String("service", locErrors.ServiceName),
			logger.String("ip", ipStr),
			logger.String("error_code", locErrors.CodeInvalidIP),
		)
		return nil, locErrors.NewLocationError(locErrors.CodeInvalidIP, "Invalid IP address format")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var country model.CountryRecord
	if err := s.countryDB.Lookup(ip, &country); err != nil {
		s.log.Error("Country lookup failed",
			logger.String("service", locErrors.ServiceName),
			logger.String("ip", ipStr),
			logger.String("error_code", locErrors.CodeLookupFailed),
			logger.Error(err),
		)
		return nil, err
	}

	s.log.Info("Country lookup successful",
		logger.String("service", locErrors.ServiceName),
		logger.String("ip", ipStr),
	)

	return &country, nil
}

// ============================================================================
// Cleanup
// ============================================================================

func (s *LocationService) Close() error {
	s.log.Info("Closing LocationService databases",
		logger.String("service", locErrors.ServiceName),
	)

	s.mu.Lock()
	defer s.mu.Unlock()

	var errs []error

	if s.cityDB != nil {
		s.log.Debug("Closing city database",
			logger.String("service", locErrors.ServiceName),
		)
		if err := s.cityDB.Close(); err != nil {
			s.log.Error("Failed to close city database",
				logger.String("service", locErrors.ServiceName),
				logger.Error(err),
			)
			errs = append(errs, err)
		}
	}

	if s.asnDB != nil {
		s.log.Debug("Closing ASN database",
			logger.String("service", locErrors.ServiceName),
		)
		if err := s.asnDB.Close(); err != nil {
			s.log.Error("Failed to close ASN database",
				logger.String("service", locErrors.ServiceName),
				logger.Error(err),
			)
			errs = append(errs, err)
		}
	}

	if s.countryDB != nil {
		s.log.Debug("Closing country database",
			logger.String("service", locErrors.ServiceName),
		)
		if err := s.countryDB.Close(); err != nil {
			s.log.Error("Failed to close country database",
				logger.String("service", locErrors.ServiceName),
				logger.Error(err),
			)
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		s.log.Error("Errors occurred while closing databases",
			logger.String("service", locErrors.ServiceName),
			logger.Int("error_count", len(errs)),
		)
		return locErrors.NewLocationError(locErrors.CodeDatabaseLoadFailed, "Failed to close one or more databases")
	}

	s.log.Info("LocationService databases closed successfully",
		logger.String("service", locErrors.ServiceName),
	)

	return nil
}
