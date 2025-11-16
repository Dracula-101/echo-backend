package config

import "time"

type Config struct {
	Service  ServiceConfig  `yaml:"service" mapstructure:"service"`
	Server   ServerConfig   `yaml:"server" mapstructure:"server"`
	Database DatabaseConfig `yaml:"database" mapstructure:"database"`
	Cache    CacheConfig    `yaml:"cache" mapstructure:"cache"`
	Presence PresenceConfig `yaml:"presence" mapstructure:"presence"`
	Logging  LoggingConfig  `yaml:"logging" mapstructure:"logging"`
	Shutdown ShutdownConfig `yaml:"shutdown" mapstructure:"shutdown"`
}

type ServiceConfig struct {
	Name        string `yaml:"name" mapstructure:"name"`
	Version     string `yaml:"version" mapstructure:"version"`
	Environment string `yaml:"environment" mapstructure:"environment"`
}

type ServerConfig struct {
	Port            int           `yaml:"port" mapstructure:"port"`
	Host            string        `yaml:"host" mapstructure:"host"`
	ReadTimeout     time.Duration `yaml:"read_timeout" mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout" mapstructure:"write_timeout"`
	IdleTimeout     time.Duration `yaml:"idle_timeout" mapstructure:"idle_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" mapstructure:"shutdown_timeout"`
	MaxHeaderBytes  int           `yaml:"max_header_bytes" mapstructure:"max_header_bytes"`
}

type DatabaseConfig struct {
	Postgres PostgresConfig `yaml:"postgres" mapstructure:"postgres"`
}

type PostgresConfig struct {
	Host            string        `yaml:"host" mapstructure:"host"`
	Port            int           `yaml:"port" mapstructure:"port"`
	User            string        `yaml:"user" mapstructure:"user"`
	Password        string        `yaml:"password" mapstructure:"password"`
	DBName          string        `yaml:"db_name" mapstructure:"db_name"`
	SSLMode         string        `yaml:"ssl_mode" mapstructure:"ssl_mode"`
	MaxOpenConns    int           `yaml:"max_open_conns" mapstructure:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns" mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time" mapstructure:"conn_max_idle_time"`
}

type CacheConfig struct {
	Enabled bool        `yaml:"enabled" mapstructure:"enabled"`
	Redis   RedisConfig `yaml:"redis" mapstructure:"redis"`
}

type RedisConfig struct {
	Host         string        `yaml:"host" mapstructure:"host"`
	Port         int           `yaml:"port" mapstructure:"port"`
	Password     string        `yaml:"password" mapstructure:"password"`
	DB           int           `yaml:"db" mapstructure:"db"`
	MaxRetries   int           `yaml:"max_retries" mapstructure:"max_retries"`
	PoolSize     int           `yaml:"pool_size" mapstructure:"pool_size"`
	MinIdleConns int           `yaml:"min_idle_conns" mapstructure:"min_idle_conns"`
	DialTimeout  time.Duration `yaml:"dial_timeout" mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `yaml:"read_timeout" mapstructure:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout" mapstructure:"write_timeout"`
}

type PresenceConfig struct {
	HeartbeatInterval  time.Duration `yaml:"heartbeat_interval" mapstructure:"heartbeat_interval"`
	SessionTimeout     time.Duration `yaml:"session_timeout" mapstructure:"session_timeout"`
	CleanupInterval    time.Duration `yaml:"cleanup_interval" mapstructure:"cleanup_interval"`
	TypingIndicatorTTL time.Duration `yaml:"typing_indicator_ttl" mapstructure:"typing_indicator_ttl"`
}

type LoggingConfig struct {
	Level      string `yaml:"level" mapstructure:"level"`
	Format     string `yaml:"format" mapstructure:"format"`
	Output     string `yaml:"output" mapstructure:"output"`
	TimeFormat string `yaml:"time_format" mapstructure:"time_format"`
}

type ShutdownConfig struct {
	Timeout            time.Duration `yaml:"timeout" mapstructure:"timeout"`
	WaitForConnections bool          `yaml:"wait_for_connections" mapstructure:"wait_for_connections"`
	DrainTimeout       time.Duration `yaml:"drain_timeout" mapstructure:"drain_timeout"`
}
