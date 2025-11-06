package config

import (
	"fmt"
	"strings"
	"time"
)

// ValidateAndSetDefaults validates the configuration and sets defaults where needed
func ValidateAndSetDefaults(cfg *Config) error {
	validators := []func(*Config) error{
		validateService,
		validateServer,
		validateDatabase,
		validateCache,
		validateSecurity,
		validateLogging,
		validateObservability,
		validateShutdown,
		validateFeatures,
	}

	for _, validator := range validators {
		if err := validator(cfg); err != nil {
			return err
		}
	}

	return nil
}

func validateService(cfg *Config) error {
	if cfg.Service.Name == "" {
		return fmt.Errorf("service.name is required")
	}

	if cfg.Service.Version == "" {
		cfg.Service.Version = "1.0.0"
	}

	if cfg.Service.Environment == "" {
		cfg.Service.Environment = "development"
	}

	validEnvs := []string{"development", "staging", "production"}
	if !contains(validEnvs, cfg.Service.Environment) {
		return fmt.Errorf("service.environment must be one of: %s", strings.Join(validEnvs, ", "))
	}

	return nil
}

func validateServer(cfg *Config) error {
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535, got %d", cfg.Server.Port)
	}

	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}

	if cfg.Server.ReadTimeout <= 0 {
		cfg.Server.ReadTimeout = 15 * time.Second
	}

	if cfg.Server.WriteTimeout <= 0 {
		cfg.Server.WriteTimeout = 15 * time.Second
	}

	if cfg.Server.IdleTimeout <= 0 {
		cfg.Server.IdleTimeout = 60 * time.Second
	}

	if cfg.Server.ShutdownTimeout <= 0 {
		cfg.Server.ShutdownTimeout = 30 * time.Second
	}

	if cfg.Server.MaxHeaderBytes <= 0 {
		cfg.Server.MaxHeaderBytes = 1 << 20 // 1MB
	}

	return nil
}

func validateDatabase(cfg *Config) error {
	db := &cfg.Database.Postgres

	if db.Host == "" {
		return fmt.Errorf("database.postgres.host is required")
	}

	if db.Port <= 0 || db.Port > 65535 {
		return fmt.Errorf("database.postgres.port must be between 1 and 65535, got %d", db.Port)
	}

	if db.User == "" {
		return fmt.Errorf("database.postgres.user is required")
	}

	if db.Password == "" {
		return fmt.Errorf("database.postgres.password is required")
	}

	if db.DBName == "" {
		return fmt.Errorf("database.postgres.db_name is required")
	}

	validSSLModes := []string{"disable", "require", "verify-ca", "verify-full"}
	if !contains(validSSLModes, db.SSLMode) {
		return fmt.Errorf("database.postgres.ssl_mode must be one of: %s", strings.Join(validSSLModes, ", "))
	}

	if db.MaxOpenConns <= 0 {
		db.MaxOpenConns = 25
	}

	if db.MaxIdleConns <= 0 {
		db.MaxIdleConns = 5
	}

	if db.MaxIdleConns > db.MaxOpenConns {
		return fmt.Errorf("database.postgres.max_idle_conns (%d) cannot exceed max_open_conns (%d)",
			db.MaxIdleConns, db.MaxOpenConns)
	}

	if db.ConnMaxLifetime <= 0 {
		db.ConnMaxLifetime = 5 * time.Minute
	}

	if db.ConnMaxIdleTime <= 0 {
		db.ConnMaxIdleTime = 5 * time.Minute
	}

	return nil
}

func validateCache(cfg *Config) error {
	if !cfg.Cache.Enabled {
		return nil
	}

	redis := &cfg.Cache.RedisConfig

	if redis.RedisHost == "" {
		return fmt.Errorf("cache.redis.host is required when cache is enabled")
	}

	if redis.RedisPort <= 0 || redis.RedisPort > 65535 {
		return fmt.Errorf("cache.redis.port must be between 1 and 65535, got %d", redis.RedisPort)
	}

	if redis.RedisDB < 0 || redis.RedisDB > 15 {
		return fmt.Errorf("cache.redis.db must be between 0 and 15, got %d", redis.RedisDB)
	}

	if redis.RedisPoolSize <= 0 {
		redis.RedisPoolSize = 10
	}

	if redis.RedisMinIdleConns < 0 {
		redis.RedisMinIdleConns = 0
	}

	if redis.RedisDialTimeout <= 0 {
		redis.RedisDialTimeout = 5 * time.Second
	}

	return nil
}

func validateSecurity(cfg *Config) error {
	// Validate CORS
	if cfg.Security.AllowedOrigins == "" {
		return fmt.Errorf("security.allowed_origins is required")
	}

	if cfg.Security.AllowedMethods == "" {
		cfg.Security.AllowedMethods = "GET,POST,PUT,PATCH,DELETE,OPTIONS"
	}

	if cfg.Security.AllowedHeaders == "" {
		cfg.Security.AllowedHeaders = "Content-Type,Authorization,X-Request-ID,X-Correlation-ID"
	}

	if cfg.Security.MaxAge <= 0 {
		cfg.Security.MaxAge = 3600
	}

	// Validate Security Headers
	if cfg.Security.SecurityHeaders.XFrameOptions == "" {
		cfg.Security.SecurityHeaders.XFrameOptions = "DENY"
	}

	if cfg.Security.SecurityHeaders.XContentTypeOptions == "" {
		cfg.Security.SecurityHeaders.XContentTypeOptions = "nosniff"
	}

	if cfg.Security.SecurityHeaders.XXSSProtection == "" {
		cfg.Security.SecurityHeaders.XXSSProtection = "1; mode=block"
	}

	// Validate Body Size
	if cfg.Security.MaxBodySize <= 0 {
		cfg.Security.MaxBodySize = 1 << 20 // 1MB
	}

	// Validate Rate Limiting
	if cfg.Security.RateLimit.Enabled {
		if cfg.Security.RateLimit.Window <= 0 {
			cfg.Security.RateLimit.Window = 1 * time.Minute
		}

		if cfg.Security.RateLimit.Global.Requests <= 0 {
			cfg.Security.RateLimit.Global.Requests = 100
		}

		// Validate endpoint rate limits
		endpoints := map[string]*EndpointLimit{
			"get_profile":    &cfg.Security.RateLimit.Endpoints.GetProfile,
			"update_profile": &cfg.Security.RateLimit.Endpoints.UpdateProfile,
			"search_users":   &cfg.Security.RateLimit.Endpoints.SearchUsers,
		}

		for name, limit := range endpoints {
			if limit.Requests > 0 && limit.Window <= 0 {
				return fmt.Errorf("security.rate_limit.endpoints.%s.window must be positive when requests is set", name)
			}
		}
	}

	return nil
}

func validateLogging(cfg *Config) error {
	validLevels := []string{"debug", "info", "warn", "error", "fatal"}
	if !contains(validLevels, cfg.Logging.Level) {
		return fmt.Errorf("logging.level must be one of: %s", strings.Join(validLevels, ", "))
	}

	validFormats := []string{"json", "console"}
	if !contains(validFormats, cfg.Logging.Format) {
		return fmt.Errorf("logging.format must be one of: %s", strings.Join(validFormats, ", "))
	}

	validOutputs := []string{"stdout", "stderr", "file"}
	if !contains(validOutputs, cfg.Logging.Output) {
		return fmt.Errorf("logging.output must be one of: %s", strings.Join(validOutputs, ", "))
	}

	validTimeFormats := []string{"rfc3339", "iso8601", "unix"}
	if cfg.Logging.TimeFormat != "" && !contains(validTimeFormats, cfg.Logging.TimeFormat) {
		return fmt.Errorf("logging.time_format must be one of: %s", strings.Join(validTimeFormats, ", "))
	}

	// Validate file logging settings
	if cfg.Logging.Output == "file" {
		if cfg.Logging.File.Path == "" {
			return fmt.Errorf("logging.file.path is required when output is 'file'")
		}

		if cfg.Logging.File.MaxSize <= 0 {
			cfg.Logging.File.MaxSize = 100
		}

		if cfg.Logging.File.MaxBackups < 0 {
			cfg.Logging.File.MaxBackups = 3
		}

		if cfg.Logging.File.MaxAge < 0 {
			cfg.Logging.File.MaxAge = 28
		}
	}

	return nil
}

func validateObservability(cfg *Config) error {
	// Validate Metrics
	if cfg.Observability.Metrics.Enabled {
		if cfg.Observability.Metrics.Endpoint == "" {
			cfg.Observability.Metrics.Endpoint = "/metrics"
		}
	}

	// Validate Tracing
	if cfg.Observability.Tracing.Enabled {
		validProviders := []string{"jaeger", "zipkin", "datadog"}
		if !contains(validProviders, cfg.Observability.Tracing.Provider) {
			return fmt.Errorf("observability.tracing.provider must be one of: %s", strings.Join(validProviders, ", "))
		}

		if cfg.Observability.Tracing.Endpoint == "" {
			return fmt.Errorf("observability.tracing.endpoint is required when tracing is enabled")
		}

		if cfg.Observability.Tracing.SampleRate < 0 || cfg.Observability.Tracing.SampleRate > 1 {
			return fmt.Errorf("observability.tracing.sample_rate must be between 0.0 and 1.0, got %f", cfg.Observability.Tracing.SampleRate)
		}
	}

	// Validate Health Check
	if cfg.Observability.Health.Enabled {
		if cfg.Observability.Health.Endpoint == "" {
			cfg.Observability.Health.Endpoint = "/health"
		}
	}

	return nil
}

func validateShutdown(cfg *Config) error {
	if cfg.Shutdown.Timeout <= 0 {
		cfg.Shutdown.Timeout = 30 * time.Second
	}

	if cfg.Shutdown.WaitForConnections && cfg.Shutdown.DrainTimeout <= 0 {
		cfg.Shutdown.DrainTimeout = 5 * time.Second
	}

	// Warn if drain timeout is longer than shutdown timeout
	if cfg.Shutdown.DrainTimeout > cfg.Shutdown.Timeout {
		return fmt.Errorf("shutdown.drain_timeout (%v) cannot exceed shutdown.timeout (%v)",
			cfg.Shutdown.DrainTimeout, cfg.Shutdown.Timeout)
	}

	return nil
}

func validateFeatures(cfg *Config) error {
	// Validate Profile Picture
	if cfg.Features.ProfilePicture.Enabled {
		if cfg.Features.ProfilePicture.MaxSizeBytes <= 0 {
			cfg.Features.ProfilePicture.MaxSizeBytes = 5 * 1024 * 1024 // 5MB default
		}
		if cfg.Features.ProfilePicture.AllowedFormats == "" {
			cfg.Features.ProfilePicture.AllowedFormats = "jpg,jpeg,png,webp"
		}
	}

	// Validate User Search
	if cfg.Features.UserSearch.Enabled {
		if cfg.Features.UserSearch.MaxResults <= 0 {
			cfg.Features.UserSearch.MaxResults = 50
		}
		if cfg.Features.UserSearch.MinQueryLength <= 0 {
			cfg.Features.UserSearch.MinQueryLength = 2
		}
	}

	return nil
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Validate validates the entire configuration
func Validate(cfg *Config) error {
	return ValidateAndSetDefaults(cfg)
}

// MustValidate validates the configuration and panics if validation fails
func MustValidate(cfg *Config) {
	if err := ValidateAndSetDefaults(cfg); err != nil {
		panic(fmt.Sprintf("configuration validation failed: %v", err))
	}
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
}

// NewValidationError creates a new validation error
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}
