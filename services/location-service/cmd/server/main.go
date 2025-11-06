package main

import (
	"encoding/json"
	"fmt"
	locErrors "location-service/errors"
	"location-service/model"
	"location-service/service"
	"net"
	"net/http"
	"shared/pkg/logger"
	"shared/pkg/logger/adapter"
	"shared/server/env"
	"time"
)

type Server struct {
	locationService *service.LocationService
	host            string
	port            string
	log             logger.Logger
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type HealthResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func NewServer(svc *service.LocationService, host string, port string, log *logger.Logger) *Server {
	return &Server{
		locationService: svc,
		host:            host,
		port:            port,
		log:             *log,
	}
}

func (s *Server) Start() error {
	s.log.Info("Setting up HTTP routes",
		logger.String("service", locErrors.ServiceName),
	)

	mux := http.NewServeMux()

	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/lookup", s.handleLookup)

	handler := loggingMiddleware(corsMiddleware(mux), s.log)

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", s.host, s.port),
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	s.log.Info("Starting location service HTTP server",
		logger.String("service", locErrors.ServiceName),
		logger.String("host", s.host),
		logger.String("port", s.port),
		logger.String("address", server.Addr),
	)

	return server.ListenAndServe()
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := HealthResponse{
		Status:  "healthy",
		Message: "Location service is running",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleLookup(w http.ResponseWriter, r *http.Request) {
	s.log.Info("Location lookup request received",
		logger.String("service", locErrors.ServiceName),
		logger.String("remote_addr", r.RemoteAddr),
		logger.String("method", r.Method),
	)

	if r.Method != http.MethodGet {
		s.log.Warn("Method not allowed",
			logger.String("service", locErrors.ServiceName),
			logger.String("method", r.Method),
		)
		respondError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ipStr := r.URL.Query().Get("ip")
	if ipStr == "" {
		s.log.Warn("Missing IP parameter",
			logger.String("service", locErrors.ServiceName),
			logger.String("error_code", locErrors.CodeInvalidIP),
		)
		respondError(w, "IP parameter is required", http.StatusBadRequest)
		return
	}

	if net.ParseIP(ipStr) == nil {
		s.log.Warn("Invalid IP address format",
			logger.String("service", locErrors.ServiceName),
			logger.String("ip", ipStr),
			logger.String("error_code", locErrors.CodeInvalidIP),
		)
		respondError(w, "Invalid IP address format", http.StatusBadRequest)
		return
	}

	result, err := s.locationService.Lookup(ipStr)
	if err != nil {
		s.log.Error("Location lookup failed",
			logger.String("service", locErrors.ServiceName),
			logger.String("ip", ipStr),
			logger.Error(err),
		)
		statusCode := locErrors.HTTPStatus(locErrors.CodeLookupFailed)
		respondError(w, fmt.Sprintf("Lookup failed: %v", err), statusCode)
		return
	}

	s.log.Debug("Building lookup response",
		logger.String("service", locErrors.ServiceName),
		logger.String("ip", ipStr),
	)

	var response *model.LookupResult = &model.LookupResult{}
	if result.City != nil {
		response.City = result.City.GetCityName("en")
		response.State = result.City.GetSubdivisionName("en")
		response.StateCode = result.City.GetSubdivisionISOCode()
		if result.City.Location != nil {
			response.Latitude = result.City.Location.Latitude
			response.Longitude = result.City.Location.Longitude
			response.Timezone = result.City.Location.TimeZone
			response.PostalCode = result.City.Postal.Code
		}
	}
	if result.Country != nil {
		response.Country = result.Country.Country.Names["en"]
		response.CountryCode = result.Country.Country.ISOCode
		response.Continent = result.Country.Continent.Names["en"]
		response.ContinentCode = result.Country.Continent.Code
	}
	if result.ASN != nil {
		response.ISP = result.ASN.AutonomousSystemOrganization
	}
	response.IP = ipStr

	s.log.Info("Location lookup completed successfully",
		logger.String("service", locErrors.ServiceName),
		logger.String("ip", ipStr),
		logger.String("city", response.City),
		logger.String("country", response.Country),
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func respondError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

func loggingMiddleware(next http.Handler, log logger.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Debug("Received request",
			logger.String("service", locErrors.ServiceName),
			logger.String("method", r.Method),
			logger.String("url", r.URL.String()),
			logger.String("remote_addr", r.RemoteAddr),
		)
		next.ServeHTTP(w, r)
		log.Debug("Completed request",
			logger.String("service", locErrors.ServiceName),
			logger.String("method", r.Method),
			logger.String("url", r.URL.String()),
			logger.String("remote_addr", r.RemoteAddr),
			logger.Duration("duration", time.Since(start)),
		)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	env.LoadEnv()
	log, err := adapter.NewZap(logger.Config{
		Level:      logger.GetLoggerLevel(),
		Output:     logger.GetLoggerOutput(),
		Format:     logger.GetLoggerFormat(),
		TimeFormat: logger.GetLoggerTimeFormat(),
		Service:    "location-service",
	})
	if err != nil {
		log.Fatal("Failed to initialize logger:", logger.Error(err))
	}

	cityDBPath := env.MustGetEnv("GEOIP_CITY_DB_PATH")
	asnDBPath := env.MustGetEnv("GEOIP_ASN_DB_PATH")
	countryDBPath := env.MustGetEnv("GEOIP_COUNTRY_DB_PATH")
	host := env.GetEnv("HOST", "0.0.0.0")
	port := env.GetEnv("PORT", "8090")

	log.Info("Configuration",
		logger.String("city_db_path", cityDBPath),
		logger.String("asn_db_path", asnDBPath),
		logger.String("country_db_path", countryDBPath),
		logger.String("host", host),
		logger.String("port", port),
	)

	cfg := service.Config{
		CityDBPath:    cityDBPath,
		ASNDBPath:     asnDBPath,
		CountryDBPath: countryDBPath,
		Logger:        log,
	}

	svc, err := service.NewLocationService(cfg)
	if err != nil {
		log.Fatal("Failed to initialize location service:", logger.Error(err))
	}
	defer svc.Close()

	log.Info("Location service initialized successfully")

	server := NewServer(svc, host, port, &log)
	if err := server.Start(); err != nil {
		log.Fatal("Server failed:", logger.Error(err))
	}
}
