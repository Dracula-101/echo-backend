package config

import (
	sharedConfig "shared/server/config"
	env "shared/server/env"
)

const (
	envAppEnv     = "APP_ENV"
	envConfigPath = "CONFIG_PATH"
)

// Load reads and parses the configuration from disk. Validation is the
// caller's responsibility.
func Load(configPath string, environment string) (*Config, error) {
	return sharedConfig.Load[Config](sharedConfig.LoadOptions{
		ConfigPath:  configPath,
		ServiceName: "media-service",
		Environment: environment,
	})
}

// LoadFromEnv reads CONFIG_PATH and APP_ENV, loads the config, and runs
// the validator so the rest of the program can rely on a fully checked
// Config. Single entry point main.go calls.
func LoadFromEnv() (*Config, error) {
	cfg, err := Load(
		env.GetEnv(envConfigPath),
		env.GetEnv(envAppEnv),
	)
	if err != nil {
		return nil, err
	}
	if err := Validate(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
