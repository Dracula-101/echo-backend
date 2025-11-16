package config

import (
	"shared/server/config"
)

func Load(configPath string, env string) (*Config, error) {
	return config.Load[Config](config.LoadOptions{
		ConfigPath:  configPath,
		ServiceName: "presence-service",
		Environment: env,
	})
}
