package config

import (
	"fmt"
	"time"
)

// ValidateAndSetDefaults validates the configuration and sets default values
func ValidateAndSetDefaults(cfg *Config) error {
	// Service validation
	if cfg.Service.Name == "" {
		cfg.Service.Name = "ws-service"
	}
	if cfg.Service.Version == "" {
		cfg.Service.Version = "1.0.0"
	}
	if cfg.Service.Environment == "" {
		cfg.Service.Environment = "development"
	}

	// Server validation
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8086
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
	if cfg.Server.MaxHeaderBytes == 0 {
		cfg.Server.MaxHeaderBytes = 1 << 20 // 1 MB
	}

	// Database validation
	if cfg.Database.Postgres.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if cfg.Database.Postgres.Port == 0 {
		cfg.Database.Postgres.Port = 5432
	}
	if cfg.Database.Postgres.User == "" {
		return fmt.Errorf("database user is required")
	}
	if cfg.Database.Postgres.DBName == "" {
		return fmt.Errorf("database name is required")
	}
	if cfg.Database.Postgres.MaxOpenConns == 0 {
		cfg.Database.Postgres.MaxOpenConns = 25
	}
	if cfg.Database.Postgres.MaxIdleConns == 0 {
		cfg.Database.Postgres.MaxIdleConns = 5
	}
	if cfg.Database.Postgres.ConnMaxLifetime == 0 {
		cfg.Database.Postgres.ConnMaxLifetime = 5 * time.Minute
	}
	if cfg.Database.Postgres.ConnMaxIdleTime == 0 {
		cfg.Database.Postgres.ConnMaxIdleTime = 5 * time.Minute
	}

	// Cache validation
	if cfg.Cache.Enabled {
		if cfg.Cache.Redis.Host == "" {
			cfg.Cache.Redis.Host = "localhost"
		}
		if cfg.Cache.Redis.Port == 0 {
			cfg.Cache.Redis.Port = 6379
		}
		if cfg.Cache.Redis.MaxRetries == 0 {
			cfg.Cache.Redis.MaxRetries = 3
		}
		if cfg.Cache.Redis.PoolSize == 0 {
			cfg.Cache.Redis.PoolSize = 10
		}
		if cfg.Cache.Redis.MinIdleConns == 0 {
			cfg.Cache.Redis.MinIdleConns = 5
		}
		if cfg.Cache.Redis.DialTimeout == 0 {
			cfg.Cache.Redis.DialTimeout = 5 * time.Second
		}
		if cfg.Cache.Redis.ReadTimeout == 0 {
			cfg.Cache.Redis.ReadTimeout = 3 * time.Second
		}
		if cfg.Cache.Redis.WriteTimeout == 0 {
			cfg.Cache.Redis.WriteTimeout = 3 * time.Second
		}
	}

	// WebSocket validation
	if cfg.WebSocket.WriteWait == 0 {
		cfg.WebSocket.WriteWait = 10 * time.Second
	}
	if cfg.WebSocket.PongWait == 0 {
		cfg.WebSocket.PongWait = 60 * time.Second
	}
	if cfg.WebSocket.PingPeriod == 0 {
		cfg.WebSocket.PingPeriod = 54 * time.Second
	}
	if cfg.WebSocket.ReadBufferSize == 0 {
		cfg.WebSocket.ReadBufferSize = 1024
	}
	if cfg.WebSocket.WriteBufferSize == 0 {
		cfg.WebSocket.WriteBufferSize = 1024
	}
	if cfg.WebSocket.MaxMessageSize == 0 {
		cfg.WebSocket.MaxMessageSize = 10 * 1024 * 1024
	}
	if cfg.WebSocket.ClientBufferSize == 0 {
		cfg.WebSocket.ClientBufferSize = 256
	}
	if cfg.WebSocket.CleanupInterval == 0 {
		cfg.WebSocket.CleanupInterval = 30 * time.Second
	}
	if cfg.WebSocket.StaleConnectionTimeout == 0 {
		cfg.WebSocket.StaleConnectionTimeout = 90 * time.Second
	}
	if cfg.WebSocket.RegisterBuffer == 0 {
		cfg.WebSocket.RegisterBuffer = 256
	}
	if cfg.WebSocket.UnregisterBuffer == 0 {
		cfg.WebSocket.UnregisterBuffer = 256
	}
	if cfg.WebSocket.BroadcastBuffer == 0 {
		cfg.WebSocket.BroadcastBuffer = 1024
	}

	// Logging validation
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}
	if cfg.Logging.Format == "" {
		cfg.Logging.Format = "json"
	}
	if cfg.Logging.Output == "" {
		cfg.Logging.Output = "stdout"
	}

	// Shutdown validation
	if cfg.Shutdown.Timeout == 0 {
		cfg.Shutdown.Timeout = 30 * time.Second
	}
	if cfg.Shutdown.DrainTimeout == 0 {
		cfg.Shutdown.DrainTimeout = 5 * time.Second
	}

	return nil
}
