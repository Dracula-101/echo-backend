package config

import (
	"shared/server/config"
)

// Load loads configuration from file
func Load(configPath string, env string) (*Config, error) {
	return config.Load[Config](config.LoadOptions{
		ConfigPath:  configPath,
		ServiceName: "message-service",
		Environment: env,
	})
}
