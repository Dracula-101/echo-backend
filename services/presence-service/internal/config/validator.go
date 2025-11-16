package config

import (
	"errors"
	"time"
)

func ValidateAndSetDefaults(cfg *Config) error {
	if cfg.Service.Name == "" {
		cfg.Service.Name = "presence-service"
	}

	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8085
	}

	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}

	if cfg.Server.ReadTimeout == 0 {
		cfg.Server.ReadTimeout = 15 * time.Second
	}

	if cfg.Server.WriteTimeout == 0 {
		cfg.Server.WriteTimeout = 15 * time.Second
	}

	if cfg.Server.IdleTimeout == 0 {
		cfg.Server.IdleTimeout = 60 * time.Second
	}

	if cfg.Server.ShutdownTimeout == 0 {
		cfg.Server.ShutdownTimeout = 30 * time.Second
	}

	if cfg.Database.Postgres.Host == "" {
		return errors.New("database host is required")
	}

	if cfg.Database.Postgres.Port == 0 {
		cfg.Database.Postgres.Port = 5432
	}

	if cfg.Database.Postgres.User == "" {
		return errors.New("database user is required")
	}

	if cfg.Database.Postgres.DBName == "" {
		return errors.New("database name is required")
	}

	if cfg.Presence.HeartbeatInterval == 0 {
		cfg.Presence.HeartbeatInterval = 30 * time.Second
	}

	if cfg.Presence.SessionTimeout == 0 {
		cfg.Presence.SessionTimeout = 5 * time.Minute
	}

	if cfg.Presence.CleanupInterval == 0 {
		cfg.Presence.CleanupInterval = 1 * time.Minute
	}

	if cfg.Presence.TypingIndicatorTTL == 0 {
		cfg.Presence.TypingIndicatorTTL = 10 * time.Second
	}

	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}

	if cfg.Logging.Format == "" {
		cfg.Logging.Format = "json"
	}

	if cfg.Shutdown.Timeout == 0 {
		cfg.Shutdown.Timeout = 30 * time.Second
	}

	return nil
}
