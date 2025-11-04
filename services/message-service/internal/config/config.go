package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the message service
type Config struct {
	Service   ServiceConfig
	Server    ServerConfig
	Database  DatabaseConfig
	Kafka     KafkaConfig
	Cache     CacheConfig
	WebSocket WebSocketConfig
	Logging   LoggingConfig
	Shutdown  ShutdownConfig
}

// ServiceConfig holds service metadata
type ServiceConfig struct {
	Name    string
	Version string
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	MaxHeaderBytes  int
	EnableCORS      bool
	AllowedOrigins  []string
	TrustedProxies  []string
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// KafkaConfig holds Kafka configuration
type KafkaConfig struct {
	Brokers              []string
	Topic                string
	NotificationTopic    string
	ConsumerGroup        string
	EnableCompression    bool
	CompressionType      string
	RetryMax             int
	RequiredAcks         int
	EnableIdempotence    bool
}

// CacheConfig holds cache (Redis) configuration
type CacheConfig struct {
	Enabled      bool
	Host         string
	Port         int
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	MaxRetries   int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolTimeout  time.Duration
}

// ShutdownConfig holds shutdown configuration
type ShutdownConfig struct {
	WaitForConnections bool
	DrainTimeout       time.Duration
}

// WebSocketConfig holds WebSocket configuration
type WebSocketConfig struct {
	ReadBufferSize     int
	WriteBufferSize    int
	WriteWait          time.Duration
	PongWait           time.Duration
	PingPeriod         time.Duration
	MaxMessageSize     int64
	ClientBufferSize   int
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level      string
	Format     string // json or console
	OutputPath string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	config := &Config{
		Service: ServiceConfig{
			Name:    getEnv("SERVICE_NAME", "message-service"),
			Version: getEnv("SERVICE_VERSION", "1.0.0"),
		},
		Server: ServerConfig{
			Host:            getEnv("SERVER_HOST", "0.0.0.0"),
			Port:            getEnvAsInt("SERVER_PORT", 8083),
			ReadTimeout:     getEnvAsDuration("SERVER_READ_TIMEOUT", "15s"),
			WriteTimeout:    getEnvAsDuration("SERVER_WRITE_TIMEOUT", "15s"),
			IdleTimeout:     getEnvAsDuration("SERVER_IDLE_TIMEOUT", "60s"),
			ShutdownTimeout: getEnvAsDuration("SERVER_SHUTDOWN_TIMEOUT", "30s"),
			MaxHeaderBytes:  getEnvAsInt("SERVER_MAX_HEADER_BYTES", 1<<20), // 1 MB
			EnableCORS:      getEnvAsBool("ENABLE_CORS", true),
			AllowedOrigins:  getEnvAsSlice("ALLOWED_ORIGINS", []string{"*"}),
			TrustedProxies:  getEnvAsSlice("TRUSTED_PROXIES", []string{}),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnvAsInt("DB_PORT", 5432),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", ""),
			DBName:          getEnv("DB_NAME", "echo_backend"),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", "5m"),
			ConnMaxIdleTime: getEnvAsDuration("DB_CONN_MAX_IDLE_TIME", "5m"),
		},
		Kafka: KafkaConfig{
			Brokers:           getEnvAsSlice("KAFKA_BROKERS", []string{"localhost:9092"}),
			Topic:             getEnv("KAFKA_TOPIC", "messages"),
			NotificationTopic: getEnv("KAFKA_NOTIFICATION_TOPIC", "notifications"),
			ConsumerGroup:     getEnv("KAFKA_CONSUMER_GROUP", "message-service"),
			EnableCompression: getEnvAsBool("KAFKA_ENABLE_COMPRESSION", true),
			CompressionType:   getEnv("KAFKA_COMPRESSION_TYPE", "snappy"),
			RetryMax:          getEnvAsInt("KAFKA_RETRY_MAX", 3),
			RequiredAcks:      getEnvAsInt("KAFKA_REQUIRED_ACKS", 1),
			EnableIdempotence: getEnvAsBool("KAFKA_ENABLE_IDEMPOTENCE", true),
		},
		Cache: CacheConfig{
			Enabled:      getEnvAsBool("CACHE_ENABLED", false),
			Host:         getEnv("REDIS_HOST", "localhost"),
			Port:         getEnvAsInt("REDIS_PORT", 6379),
			Password:     getEnv("REDIS_PASSWORD", ""),
			DB:           getEnvAsInt("REDIS_DB", 0),
			PoolSize:     getEnvAsInt("REDIS_POOL_SIZE", 10),
			MinIdleConns: getEnvAsInt("REDIS_MIN_IDLE_CONNS", 2),
			MaxRetries:   getEnvAsInt("REDIS_MAX_RETRIES", 3),
			DialTimeout:  getEnvAsDuration("REDIS_DIAL_TIMEOUT", "5s"),
			ReadTimeout:  getEnvAsDuration("REDIS_READ_TIMEOUT", "3s"),
			WriteTimeout: getEnvAsDuration("REDIS_WRITE_TIMEOUT", "3s"),
			PoolTimeout:  getEnvAsDuration("REDIS_POOL_TIMEOUT", "4s"),
		},
		WebSocket: WebSocketConfig{
			ReadBufferSize:   getEnvAsInt("WS_READ_BUFFER_SIZE", 1024),
			WriteBufferSize:  getEnvAsInt("WS_WRITE_BUFFER_SIZE", 1024),
			WriteWait:        getEnvAsDuration("WS_WRITE_WAIT", "10s"),
			PongWait:         getEnvAsDuration("WS_PONG_WAIT", "60s"),
			PingPeriod:       getEnvAsDuration("WS_PING_PERIOD", "54s"),
			MaxMessageSize:   getEnvAsInt64("WS_MAX_MESSAGE_SIZE", 10*1024*1024), // 10 MB
			ClientBufferSize: getEnvAsInt("WS_CLIENT_BUFFER_SIZE", 256),
		},
		Logging: LoggingConfig{
			Level:      getEnv("LOG_LEVEL", "info"),
			Format:     getEnv("LOG_FORMAT", "json"),
			OutputPath: getEnv("LOG_OUTPUT_PATH", "stdout"),
		},
		Shutdown: ShutdownConfig{
			WaitForConnections: getEnvAsBool("SHUTDOWN_WAIT_FOR_CONNECTIONS", false),
			DrainTimeout:       getEnvAsDuration("SHUTDOWN_DRAIN_TIMEOUT", "5s"),
		},
	}

	return config, nil
}

// GetDatabaseDSN returns the database connection string
func (c *Config) GetDatabaseDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.DBName,
		c.Database.SSLMode,
	)
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue string) time.Duration {
	value := getEnv(key, defaultValue)
	duration, err := time.ParseDuration(value)
	if err != nil {
		duration, _ = time.ParseDuration(defaultValue)
	}
	return duration
}

func getEnvAsSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// Simple comma-separated parsing
		result := []string{}
		for _, v := range splitString(value, ",") {
			if trimmed := trimString(v); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return defaultValue
}

func splitString(s, sep string) []string {
	var result []string
	current := ""
	for _, char := range s {
		if string(char) == sep {
			result = append(result, current)
			current = ""
		} else {
			current += string(char)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func trimString(s string) string {
	// Simple trim implementation
	start := 0
	end := len(s)

	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n') {
		start++
	}

	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n') {
		end--
	}

	return s[start:end]
}
