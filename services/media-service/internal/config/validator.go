package config

import (
	"fmt"
)

// Validate validates the configuration
func Validate(cfg *Config) error {
	if err := validateService(&cfg.Service); err != nil {
		return fmt.Errorf("service config: %w", err)
	}

	if err := validateServer(&cfg.Server); err != nil {
		return fmt.Errorf("server config: %w", err)
	}

	if err := validateDatabase(&cfg.Database); err != nil {
		return fmt.Errorf("database config: %w", err)
	}

	if cfg.Cache.Enabled {
		if err := validateCache(&cfg.Cache); err != nil {
			return fmt.Errorf("cache config: %w", err)
		}
	}

	if err := validateStorage(&cfg.Storage); err != nil {
		return fmt.Errorf("storage config: %w", err)
	}

	if err := validateProcessing(&cfg.Processing); err != nil {
		return fmt.Errorf("processing config: %w", err)
	}

	return nil
}

func validateService(cfg *ServiceConfig) error {
	if cfg.Name == "" {
		return fmt.Errorf("service name is required")
	}
	if cfg.Version == "" {
		return fmt.Errorf("service version is required")
	}
	return nil
}

func validateServer(cfg *ServerConfig) error {
	if cfg.Port <= 0 || cfg.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", cfg.Port)
	}
	if cfg.ReadTimeout <= 0 {
		return fmt.Errorf("read timeout must be positive")
	}
	if cfg.WriteTimeout <= 0 {
		return fmt.Errorf("write timeout must be positive")
	}
	return nil
}

func validateDatabase(cfg *DatabaseConfig) error {
	if cfg.Postgres.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if cfg.Postgres.Port <= 0 || cfg.Postgres.Port > 65535 {
		return fmt.Errorf("invalid database port: %d", cfg.Postgres.Port)
	}
	if cfg.Postgres.User == "" {
		return fmt.Errorf("database user is required")
	}
	if cfg.Postgres.DBName == "" {
		return fmt.Errorf("database name is required")
	}
	return nil
}

func validateCache(cfg *CacheConfig) error {
	if cfg.RedisConfig.RedisHost == "" {
		return fmt.Errorf("redis host is required")
	}
	if cfg.RedisConfig.RedisPort <= 0 || cfg.RedisConfig.RedisPort > 65535 {
		return fmt.Errorf("invalid redis port: %d", cfg.RedisConfig.RedisPort)
	}
	return nil
}

func validateStorage(cfg *StorageConfig) error {
	if cfg.Provider == "" {
		return fmt.Errorf("storage provider is required")
	}
	if cfg.Provider != "local" {
		if cfg.Bucket == "" {
			return fmt.Errorf("storage bucket is required for non-local provider")
		}
		if cfg.AccessKeyID == "" {
			return fmt.Errorf("storage access key ID is required")
		}
		if cfg.SecretAccessKey == "" {
			return fmt.Errorf("storage secret access key is required")
		}
	}
	if cfg.MaxFileSize <= 0 {
		return fmt.Errorf("max file size must be positive")
	}
	return nil
}

func validateProcessing(cfg *ProcessingConfig) error {
	if cfg.Workers <= 0 {
		return fmt.Errorf("workers must be positive")
	}
	if cfg.QueueSize <= 0 {
		return fmt.Errorf("queue size must be positive")
	}
	if cfg.MaxAttempts <= 0 {
		return fmt.Errorf("max attempts must be positive")
	}
	return nil
}
