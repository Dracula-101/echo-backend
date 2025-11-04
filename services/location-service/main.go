package main

import (
	"encoding/json"
	"fmt"
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

	s.log.Info(fmt.Sprintf("Starting location service on port %s", s.port))
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
	if r.Method != http.MethodGet {
		respondError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ipStr := r.URL.Query().Get("ip")
	if ipStr == "" {
		respondError(w, "IP parameter is required", http.StatusBadRequest)
		return
	}

	if net.ParseIP(ipStr) == nil {
		respondError(w, "Invalid IP address format", http.StatusBadRequest)
		return
	}

	result, err := s.locationService.Lookup(ipStr)
	if err != nil {
		respondError(w, fmt.Sprintf("City lookup failed: %v", err), http.StatusInternalServerError)
		return
	}
	s.log.Debug("Location lookup successful", logger.String("result", result.String()))
	var response *model.LookupResult = &model.LookupResult{}
	if result.City != nil {
		response.City = result.City.GetCityName("en")
		response.State = result.City.GetSubdivisionName("en")
		response.StateCode = result.City.GetSubdivisionISOCode()
	}
	if result.Country != nil {
		response.Country = result.Country.Country.Names["en"]
		response.CountryCode = result.Country.Country.ISOCode
		response.Continent = result.Country.Continent.Names["en"]
		response.ContinentCode = result.Country.Continent.Code
	}
	if result.City.Location != nil {
		response.Latitude = result.City.Location.Latitude
		response.Longitude = result.City.Location.Longitude
		response.Timezone = result.City.Location.TimeZone
		response.PostalCode = result.City.Postal.Code
	}
	if result.ASN != nil {
		response.ISP = result.ASN.AutonomousSystemOrganization
	}
	response.IP = ipStr

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
		log.Info("Received request:",
			logger.String("method", r.Method),
			logger.String("url", r.URL.String()),
			logger.String("remote_addr", r.RemoteAddr),
		)
		next.ServeHTTP(w, r)
		log.Info("Completed request:",
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
	if err := env.LoadEnv(); err != nil {
		panic(fmt.Sprintf("Error loading .env.location file: %v", err))
	}
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

	cityDBPath := env.MustGetEnv("CITY_DB_PATH")
	asnDBPath := env.MustGetEnv("ASN_DB_PATH")
	countryDBPath := env.MustGetEnv("COUNTRY_DB_PATH")
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
