package response

import (
	"os"
	"shared/server/env"
)

// Environment represents the application environment
type Environment string

const (
	EnvironmentDevelopment Environment = "development"
	EnvironmentStaging     Environment = "staging"
	EnvironmentProduction  Environment = "production"
	EnvironmentTest        Environment = "test"
)

// Config holds response configuration
type Config struct {
	Service     string
	Version     string
	Environment Environment

	// Debug settings
	EnableDebugInfo    bool
	EnableStackTraces  bool
	EnableQueryDebug   bool
	EnableCacheDebug   bool
	EnableRequestDebug bool

	// Sanitization
	SanitizeHeaders bool
	SanitizeQueries bool

	// Build info
	GitCommit string
	GitBranch string
	BuildTime string
}

// DefaultConfig returns a default configuration based on environment
func DefaultConfig() *Config {
	env := GetEnvironmentFromEnv()

	cfg := &Config{
		Service:     getEnvOrDefault("SERVICE_NAME", "api"),
		Version:     getEnvOrDefault("SERVICE_VERSION", "1.0.0"),
		Environment: env,
		GitCommit:   getEnvOrDefault("GIT_COMMIT", "unknown"),
		GitBranch:   getEnvOrDefault("GIT_BRANCH", "unknown"),
		BuildTime:   getEnvOrDefault("BUILD_TIME", "unknown"),
	}

	// Configure based on environment
	switch env {
	case EnvironmentDevelopment, EnvironmentTest:
		cfg.EnableDebugInfo = true
		cfg.EnableStackTraces = true
		cfg.EnableQueryDebug = true
		cfg.EnableCacheDebug = true
		cfg.EnableRequestDebug = true
		cfg.SanitizeHeaders = false
		cfg.SanitizeQueries = false

	case EnvironmentStaging:
		cfg.EnableDebugInfo = true
		cfg.EnableStackTraces = false
		cfg.EnableQueryDebug = true
		cfg.EnableCacheDebug = true
		cfg.EnableRequestDebug = false
		cfg.SanitizeHeaders = true
		cfg.SanitizeQueries = true

	case EnvironmentProduction:
		cfg.EnableDebugInfo = false
		cfg.EnableStackTraces = false
		cfg.EnableQueryDebug = false
		cfg.EnableCacheDebug = false
		cfg.EnableRequestDebug = false
		cfg.SanitizeHeaders = true
		cfg.SanitizeQueries = true
	}

	return cfg
}

// GetEnvironmentFromEnv reads environment from ENV var
func GetEnvironmentFromEnv() Environment {
	if env.IsDevelopment() {
		return EnvironmentDevelopment
	} else if env.IsTest() {
		return EnvironmentTest
	} else if env.IsProduction() {
		return EnvironmentProduction
	}
	return EnvironmentDevelopment
}

// IsDevelopment checks if environment is development
func (c *Config) IsDevelopment() bool {
	return c.Environment == EnvironmentDevelopment || c.Environment == EnvironmentTest
}

// IsProduction checks if environment is production
func (c *Config) IsProduction() bool {
	return c.Environment == EnvironmentProduction
}

// IsStaging checks if environment is staging
func (c *Config) IsStaging() bool {
	return c.Environment == EnvironmentStaging
}

// ShouldIncludeDebug determines if debug info should be included
func (c *Config) ShouldIncludeDebug() bool {
	return c.EnableDebugInfo && !c.IsProduction()
}

// ShouldIncludeStackTrace determines if stack traces should be included
func (c *Config) ShouldIncludeStackTrace() bool {
	return c.EnableStackTraces && !c.IsProduction()
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Global configuration instance
var globalConfig *Config

// SetGlobalConfig sets the global configuration
func SetGlobalConfig(cfg *Config) {
	globalConfig = cfg
}

// GetGlobalConfig returns the global configuration
func GetGlobalConfig() *Config {
	if globalConfig == nil {
		globalConfig = DefaultConfig()
	}
	return globalConfig
}
