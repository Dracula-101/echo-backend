package response

import (
	"os"
)

// Init initializes the response package with global configuration
// Call this once at application startup
func Init(service, version string) {
	config := &Config{
		Service:     service,
		Version:     version,
		Environment: GetEnvironmentFromEnv(),
		GitCommit:   os.Getenv("GIT_COMMIT"),
		GitBranch:   os.Getenv("GIT_BRANCH"),
		BuildTime:   os.Getenv("BUILD_TIME"),
	}

	// Auto-configure based on environment
	switch config.Environment {
	case EnvironmentDevelopment, EnvironmentTest:
		config.EnableDebugInfo = true
		config.EnableStackTraces = true
		config.EnableQueryDebug = true
		config.EnableCacheDebug = true
		config.EnableRequestDebug = true
		config.SanitizeHeaders = false
		config.SanitizeQueries = false

	case EnvironmentStaging:
		config.EnableDebugInfo = true
		config.EnableStackTraces = false
		config.EnableQueryDebug = true
		config.EnableCacheDebug = true
		config.EnableRequestDebug = false
		config.SanitizeHeaders = true
		config.SanitizeQueries = true

	case EnvironmentProduction:
		config.EnableDebugInfo = false
		config.EnableStackTraces = false
		config.EnableQueryDebug = false
		config.EnableCacheDebug = false
		config.EnableRequestDebug = false
		config.SanitizeHeaders = true
		config.SanitizeQueries = true
	}

	SetGlobalConfig(config)
}

// InitWithConfig initializes with custom configuration
func InitWithConfig(config *Config) {
	SetGlobalConfig(config)
}
