package config

import (
	"fmt"
	"time"
)

func ValidateAndSetDefaults(cfg *Config) error {
	if err := validateServiceMetadata(&cfg.Service); err != nil {
		return err
	}

	if err := validateServer(&cfg.Server); err != nil {
		return err
	}

	if err := validateServices(cfg); err != nil {
		return err
	}

	if err := validateRouterGroups(cfg); err != nil {
		return err
	}

	if err := validateRateLimit(&cfg.RateLimit); err != nil {
		return err
	}

	if err := validateSecurity(&cfg.Security); err != nil {
		return err
	}

	if err := validateLoadBalance(&cfg.LoadBalance); err != nil {
		return err
	}

	if err := validateMonitoring(&cfg.Monitoring); err != nil {
		return err
	}

	if err := validateShutdown(&cfg.Shutdown); err != nil {
		return err
	}

	return nil
}

func validateServiceMetadata(service *ServiceMetadata) error {
	if service.Name == "" {
		service.Name = "api-gateway"
	}

	if service.Version == "" {
		service.Version = "1.0.0"
	}

	if service.Environment == "" {
		service.Environment = "development"
	}

	return nil
}

func validateServer(server *ServerConfig) error {
	if server.Port <= 0 || server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", server.Port)
	}

	if server.Host == "" {
		server.Host = "0.0.0.0"
	}

	if server.ReadTimeout == 0 {
		server.ReadTimeout = 30 * time.Second
	}

	if server.WriteTimeout == 0 {
		server.WriteTimeout = 30 * time.Second
	}

	if server.IdleTimeout == 0 {
		server.IdleTimeout = 120 * time.Second
	}

	if server.ShutdownTimeout == 0 {
		server.ShutdownTimeout = 30 * time.Second
	}

	if server.MaxHeaderBytes == 0 {
		server.MaxHeaderBytes = 1048576
	}

	if server.TLSMinVersion == "" {
		server.TLSMinVersion = "1.2"
	}

	if server.TLSMinVersion != "1.2" && server.TLSMinVersion != "1.3" {
		return fmt.Errorf("invalid TLS min version: %s (must be 1.2 or 1.3)", server.TLSMinVersion)
	}

	if server.TLSEnabled {
		if server.TLSCertFile == "" {
			return fmt.Errorf("TLS enabled but no cert file provided")
		}
		if server.TLSKeyFile == "" {
			return fmt.Errorf("TLS enabled but no key file provided")
		}
	}

	return nil
}

func validateServices(cfg *Config) error {
	if len(cfg.Services) == 0 {
		return fmt.Errorf("no services configured")
	}

	for name, service := range cfg.Services {
		if len(service.Addresses) == 0 {
			return fmt.Errorf("service %s has no addresses", name)
		}

		if service.Protocol == "" {
			service.Protocol = "http"
		}

		if service.Protocol != "http" && service.Protocol != "https" && service.Protocol != "grpc" {
			return fmt.Errorf("service %s has invalid protocol: %s", name, service.Protocol)
		}

		if service.Timeout == 0 {
			service.Timeout = 10 * time.Second
		}

		if service.RetryAttempts == 0 {
			service.RetryAttempts = 3
		}

		if service.LoadBalancer == "" {
			service.LoadBalancer = cfg.LoadBalance.DefaultStrategy
		}

		if err := validateHealthCheck(name, &service.HealthCheck); err != nil {
			return err
		}

		if err := validateCircuitBreaker(name, &service.CircuitBreaker); err != nil {
			return err
		}

		cfg.Services[name] = service
	}

	return nil
}

func validateHealthCheck(serviceName string, hc *HealthCheckConfig) error {
	if !hc.Enabled {
		return nil
	}

	if hc.Path == "" {
		hc.Path = "/health"
	}

	if hc.Interval == 0 {
		hc.Interval = 10 * time.Second
	}

	if hc.Timeout == 0 {
		hc.Timeout = 5 * time.Second
	}

	if hc.FailureThreshold == 0 {
		hc.FailureThreshold = 3
	}

	if hc.Timeout >= hc.Interval {
		return fmt.Errorf("service %s: health check timeout must be less than interval", serviceName)
	}

	return nil
}

func validateCircuitBreaker(serviceName string, cb *CircuitBreakerConfig) error {
	if !cb.Enabled {
		return nil
	}

	if cb.Threshold == 0 {
		cb.Threshold = 5
	}

	if cb.Threshold < 1 {
		return fmt.Errorf("service %s: circuit breaker threshold must be at least 1", serviceName)
	}

	if cb.Timeout == 0 {
		cb.Timeout = 30 * time.Second
	}

	if cb.HalfOpenRequests == 0 {
		cb.HalfOpenRequests = 1
	}

	return nil
}

func validateRouterGroups(cfg *Config) error {
	if len(cfg.RouterGroups) == 0 {
		return fmt.Errorf("no router groups configured")
	}

	prefixMap := make(map[string]bool)

	for i := range cfg.RouterGroups {
		rg := &cfg.RouterGroups[i]

		if rg.Name == "" {
			return fmt.Errorf("router group at index %d has no name", i)
		}

		if rg.Prefix == "" {
			return fmt.Errorf("router group %s has no prefix", rg.Name)
		}

		if prefixMap[rg.Prefix] {
			return fmt.Errorf("duplicate router group prefix: %s", rg.Prefix)
		}
		prefixMap[rg.Prefix] = true

		if rg.Service == "" {
			return fmt.Errorf("router group %s has no service", rg.Name)
		}

		if _, exists := cfg.Services[rg.Service]; !exists {
			return fmt.Errorf("router group %s references non-existent service: %s", rg.Name, rg.Service)
		}

		if len(rg.Methods) == 0 {
			rg.Methods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
		}

		for _, method := range rg.Methods {
			if !isValidHTTPMethod(method) {
				return fmt.Errorf("router group %s has invalid HTTP method: %s", rg.Name, method)
			}
		}
	}

	return nil
}

func validateRateLimit(rl *RateLimitConfig) error {
	if !rl.Enabled {
		return nil
	}

	if rl.Store != "memory" && rl.Store != "redis" {
		return fmt.Errorf("invalid rate limit store: %s (must be 'memory' or 'redis')", rl.Store)
	}

	if rl.Store == "redis" && rl.RedisAddr == "" {
		return fmt.Errorf("redis store selected but no redis address provided")
	}

	if err := validateRateLimitRule("global", &rl.Global); err != nil {
		return err
	}

	for endpoint, rule := range rl.Endpoints {
		r := rule
		if err := validateRateLimitRule(endpoint, &r); err != nil {
			return err
		}
		rl.Endpoints[endpoint] = r
	}

	return nil
}

func validateRateLimitRule(name string, rule *RateLimitRule) error {
	if rule.Requests == 0 {
		rule.Requests = 1000
	}

	if rule.Requests < 1 {
		return fmt.Errorf("rate limit %s: requests must be at least 1", name)
	}

	if rule.Window == 0 {
		rule.Window = 1 * time.Minute
	}

	if rule.Window < 1*time.Second {
		return fmt.Errorf("rate limit %s: window must be at least 1 second", name)
	}

	if rule.Strategy == "" {
		rule.Strategy = "token_bucket"
	}

	validStrategies := map[string]bool{
		"token_bucket":   true,
		"sliding_window": true,
		"fixed_window":   true,
	}

	if !validStrategies[rule.Strategy] {
		return fmt.Errorf("rate limit %s: invalid strategy: %s", name, rule.Strategy)
	}

	return nil
}

func validateSecurity(sec *SecurityConfig) error {
	if sec.MaxBodySize == 0 {
		sec.MaxBodySize = 10485760
	}

	if sec.MaxBodySize < 1024 {
		return fmt.Errorf("max body size must be at least 1024 bytes")
	}

	if len(sec.AllowedOrigins) == 0 {
		sec.AllowedOrigins = []string{"*"}
	}

	if len(sec.AllowedMethods) == 0 {
		sec.AllowedMethods = []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}
	}

	for _, method := range sec.AllowedMethods {
		if !isValidHTTPMethod(method) {
			return fmt.Errorf("invalid HTTP method in allowed methods: %s", method)
		}
	}

	if len(sec.AllowedHeaders) == 0 {
		sec.AllowedHeaders = []string{"Content-Type", "Authorization"}
	}

	if sec.MaxAge == 0 {
		sec.MaxAge = 3600
	}

	if sec.MaxAge < 0 {
		return fmt.Errorf("max age cannot be negative")
	}

	return nil
}

func validateLoadBalance(lb *LoadBalanceConfig) error {
	if lb.DefaultStrategy == "" {
		lb.DefaultStrategy = "roundrobin"
	}

	validStrategies := map[string]bool{
		"roundrobin": true,
		"random":     true,
		"leastconn":  true,
		"weighted":   true,
	}

	if !validStrategies[lb.DefaultStrategy] {
		return fmt.Errorf("invalid load balance strategy: %s", lb.DefaultStrategy)
	}

	return nil
}

func validateMonitoring(mon *MonitoringConfig) error {
	if mon.MetricsPath == "" {
		mon.MetricsPath = "/metrics"
	}

	if mon.HealthPath == "" {
		mon.HealthPath = "/health"
	}

	if mon.LogLevel == "" {
		mon.LogLevel = "info"
	}

	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
		"fatal": true,
	}

	if !validLogLevels[mon.LogLevel] {
		return fmt.Errorf("invalid log level: %s", mon.LogLevel)
	}

	if mon.LogFormat == "" {
		mon.LogFormat = "json"
	}

	validLogFormats := map[string]bool{
		"json": true,
		"text": true,
	}

	if !validLogFormats[mon.LogFormat] {
		return fmt.Errorf("invalid log format: %s", mon.LogFormat)
	}

	if mon.LogOutput == "" {
		mon.LogOutput = "stdout"
	}

	validLogOutputs := map[string]bool{
		"stdout": true,
		"stderr": true,
	}

	if !validLogOutputs[mon.LogOutput] && mon.LogOutput[:1] != "/" {
		return fmt.Errorf("invalid log output: %s (must be stdout, stderr, or file path)", mon.LogOutput)
	}

	if mon.TracingSampleRate < 0 || mon.TracingSampleRate > 1 {
		return fmt.Errorf("tracing sample rate must be between 0 and 1")
	}

	return nil
}

func validateShutdown(shutdown *ShutdownConfig) error {
	if shutdown.Timeout == 0 {
		shutdown.Timeout = 30 * time.Second
	}

	if shutdown.Timeout < 1*time.Second {
		return fmt.Errorf("shutdown timeout must be at least 1 second")
	}

	if shutdown.DrainTimeout == 0 {
		shutdown.DrainTimeout = 5 * time.Second
	}

	if shutdown.DrainTimeout < 0 {
		return fmt.Errorf("drain timeout cannot be negative")
	}

	if shutdown.DrainTimeout > shutdown.Timeout {
		return fmt.Errorf("drain timeout cannot be greater than shutdown timeout")
	}

	return nil
}

func isValidHTTPMethod(method string) bool {
	validMethods := map[string]bool{
		"GET":     true,
		"POST":    true,
		"PUT":     true,
		"PATCH":   true,
		"DELETE":  true,
		"OPTIONS": true,
		"HEAD":    true,
		"CONNECT": true,
		"TRACE":   true,
	}
	return validMethods[method]
}
