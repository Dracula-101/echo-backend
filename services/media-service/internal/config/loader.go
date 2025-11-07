package config

import (
	"shared/server/config"
)

// Load loads configuration from a file with optional environment-specific overrides
func Load(configPath string, environment string) (*Config, error) {
	return config.Load[Config](config.LoadOptions{
		ConfigPath:  configPath,
		ServiceName: "media-service",
		Environment: environment,
	})
}
