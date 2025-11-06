package config

import (
	"time"
)

type Config struct {
	Service    ServiceConfig    `yaml:"service" mapstructure:"service"`
	Server     ServerConfig     `yaml:"server" mapstructure:"server"`
	Database   DatabaseConfig   `yaml:"database" mapstructure:"database"`
	Kafka      KafkaConfig      `yaml:"kafka" mapstructure:"kafka"`
	Cache      CacheConfig      `yaml:"cache" mapstructure:"cache"`
	WebSocket  WebSocketConfig  `yaml:"websocket" mapstructure:"websocket"`
	Logging    LoggingConfig    `yaml:"logging" mapstructure:"logging"`
	Shutdown   ShutdownConfig   `yaml:"shutdown" mapstructure:"shutdown"`
	Monitoring MonitoringConfig `yaml:"monitoring" mapstructure:"monitoring"`
	Security   SecurityConfig   `yaml:"security" mapstructure:"security"`
	Features   FeaturesConfig   `yaml:"features" mapstructure:"features"`
	Limits     LimitsConfig     `yaml:"limits" mapstructure:"limits"`
}

type ServiceConfig struct {
	Name        string `yaml:"name" mapstructure:"name"`
	Version     string `yaml:"version" mapstructure:"version"`
	Description string `yaml:"description" mapstructure:"description"`
	Environment string `yaml:"environment" mapstructure:"environment"`
}

type ServerConfig struct {
	Host            string        `yaml:"host" mapstructure:"host"`
	Port            int           `yaml:"port" mapstructure:"port"`
	ReadTimeout     time.Duration `yaml:"read_timeout" mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout" mapstructure:"write_timeout"`
	IdleTimeout     time.Duration `yaml:"idle_timeout" mapstructure:"idle_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" mapstructure:"shutdown_timeout"`
	MaxHeaderBytes  int           `yaml:"max_header_bytes" mapstructure:"max_header_bytes"`
	EnableCORS      bool          `yaml:"enable_cors" mapstructure:"enable_cors"`
	AllowedOrigins  []string      `yaml:"allowed_origins" mapstructure:"allowed_origins"`
	TrustedProxies  []string      `yaml:"trusted_proxies" mapstructure:"trusted_proxies"`
}

type DatabaseConfig struct {
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
	LogQueries      bool          `yaml:"log_queries" mapstructure:"log_queries"`
}

type KafkaConfig struct {
	Brokers           []string `yaml:"brokers" mapstructure:"brokers"`
	Topic             string   `yaml:"topic" mapstructure:"topic"`
	ClientID          string   `yaml:"client_id" mapstructure:"client_id"`
	GroupID           string   `yaml:"group_id" mapstructure:"group_id"`
	Compression       string   `yaml:"compression" mapstructure:"compression"`
	BatchSize         int      `yaml:"batch_size" mapstructure:"batch_size"`
	LingerMs          int      `yaml:"linger_ms" mapstructure:"linger_ms"`
	RetryMax          int      `yaml:"retry_max" mapstructure:"retry_max"`
	RetryBackoffMs    int      `yaml:"retry_backoff_ms" mapstructure:"retry_backoff_ms"`
	Acks              string   `yaml:"acks" mapstructure:"acks"`
	EnableIdempotence bool     `yaml:"enable_idempotence" mapstructure:"enable_idempotence"`
	MaxInFlight       int      `yaml:"max_in_flight" mapstructure:"max_in_flight"`
}

type CacheConfig struct {
	Enabled      bool           `yaml:"enabled" mapstructure:"enabled"`
	Host         string         `yaml:"host" mapstructure:"host"`
	Port         int            `yaml:"port" mapstructure:"port"`
	Password     string         `yaml:"password" mapstructure:"password"`
	DB           int            `yaml:"db" mapstructure:"db"`
	MaxRetries   int            `yaml:"max_retries" mapstructure:"max_retries"`
	PoolSize     int            `yaml:"pool_size" mapstructure:"pool_size"`
	MinIdleConns int            `yaml:"min_idle_conns" mapstructure:"min_idle_conns"`
	DialTimeout  time.Duration  `yaml:"dial_timeout" mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration  `yaml:"read_timeout" mapstructure:"read_timeout"`
	WriteTimeout time.Duration  `yaml:"write_timeout" mapstructure:"write_timeout"`
	PoolTimeout  time.Duration  `yaml:"pool_timeout" mapstructure:"pool_timeout"`
	IdleTimeout  time.Duration  `yaml:"idle_timeout" mapstructure:"idle_timeout"`
	TTL          CacheTTLConfig `yaml:"ttl" mapstructure:"ttl"`
}

type CacheTTLConfig struct {
	Message      time.Duration `yaml:"message" mapstructure:"message"`
	Conversation time.Duration `yaml:"conversation" mapstructure:"conversation"`
	UserPresence time.Duration `yaml:"user_presence" mapstructure:"user_presence"`
}

type WebSocketConfig struct {
	ReadBufferSize     int             `yaml:"read_buffer_size" mapstructure:"read_buffer_size"`
	WriteBufferSize    int             `yaml:"write_buffer_size" mapstructure:"write_buffer_size"`
	MaxMessageSize     int64           `yaml:"max_message_size" mapstructure:"max_message_size"`
	WriteWait          time.Duration   `yaml:"write_wait" mapstructure:"write_wait"`
	PongWait           time.Duration   `yaml:"pong_wait" mapstructure:"pong_wait"`
	PingPeriod         time.Duration   `yaml:"ping_period" mapstructure:"ping_period"`
	MaxConnections     int             `yaml:"max_connections" mapstructure:"max_connections"`
	CompressionEnabled bool            `yaml:"compression_enabled" mapstructure:"compression_enabled"`
	CompressionLevel   int             `yaml:"compression_level" mapstructure:"compression_level"`
	OriginPatterns     []string        `yaml:"origin_patterns" mapstructure:"origin_patterns"`
	Heartbeat          HeartbeatConfig `yaml:"heartbeat" mapstructure:"heartbeat"`
	ClientBufferSize   int             `yaml:"client_buffer_size" mapstructure:"client_buffer_size"`
}

type HeartbeatConfig struct {
	Enabled  bool          `yaml:"enabled" mapstructure:"enabled"`
	Interval time.Duration `yaml:"interval" mapstructure:"interval"`
}

type LoggingConfig struct {
	Level            string         `yaml:"level" mapstructure:"level"`
	Format           string         `yaml:"format" mapstructure:"format"`
	OutputPath       string         `yaml:"output_path" mapstructure:"output_path"`
	ErrorOutputPath  string         `yaml:"error_output_path" mapstructure:"error_output_path"`
	EnableCaller     bool           `yaml:"enable_caller" mapstructure:"enable_caller"`
	EnableStacktrace bool           `yaml:"enable_stacktrace" mapstructure:"enable_stacktrace"`
	Sampling         SamplingConfig `yaml:"sampling" mapstructure:"sampling"`
}

type SamplingConfig struct {
	Enabled    bool `yaml:"enabled" mapstructure:"enabled"`
	Initial    int  `yaml:"initial" mapstructure:"initial"`
	Thereafter int  `yaml:"thereafter" mapstructure:"thereafter"`
}

type ShutdownConfig struct {
	Timeout                   time.Duration `yaml:"timeout" mapstructure:"timeout"`
	WaitForConnections        bool          `yaml:"wait_for_connections" mapstructure:"wait_for_connections"`
	DrainTimeout              time.Duration `yaml:"drain_timeout" mapstructure:"drain_timeout"`
	WebSocketCloseGracePeriod time.Duration `yaml:"websocket_close_grace_period" mapstructure:"websocket_close_grace_period"`
}

type MonitoringConfig struct {
	Enabled           bool    `yaml:"enabled" mapstructure:"enabled"`
	MetricsEnabled    bool    `yaml:"metrics_enabled" mapstructure:"metrics_enabled"`
	MetricsPath       string  `yaml:"metrics_path" mapstructure:"metrics_path"`
	HealthPath        string  `yaml:"health_path" mapstructure:"health_path"`
	TracingEnabled    bool    `yaml:"tracing_enabled" mapstructure:"tracing_enabled"`
	TracingEndpoint   string  `yaml:"tracing_endpoint" mapstructure:"tracing_endpoint"`
	TracingSampleRate float64 `yaml:"tracing_sample_rate" mapstructure:"tracing_sample_rate"`
}

type SecurityConfig struct {
	EnableAuth       bool            `yaml:"enable_auth" mapstructure:"enable_auth"`
	JWTSecret        string          `yaml:"jwt_secret" mapstructure:"jwt_secret"`
	JWTIssuer        string          `yaml:"jwt_issuer" mapstructure:"jwt_issuer"`
	JWTAudience      string          `yaml:"jwt_audience" mapstructure:"jwt_audience"`
	EnableEncryption bool            `yaml:"enable_encryption" mapstructure:"enable_encryption"`
	EncryptionKey    string          `yaml:"encryption_key" mapstructure:"encryption_key"`
	AllowedFileTypes []string        `yaml:"allowed_file_types" mapstructure:"allowed_file_types"`
	MaxFileSize      int64           `yaml:"max_file_size" mapstructure:"max_file_size"`
	RateLimit        RateLimitConfig `yaml:"rate_limit" mapstructure:"rate_limit"`
}

type RateLimitConfig struct {
	Enabled           bool `yaml:"enabled" mapstructure:"enabled"`
	RequestsPerMinute int  `yaml:"requests_per_minute" mapstructure:"requests_per_minute"`
	Burst             int  `yaml:"burst" mapstructure:"burst"`
}

type FeaturesConfig struct {
	TypingIndicator  bool `yaml:"typing_indicator" mapstructure:"typing_indicator"`
	ReadReceipts     bool `yaml:"read_receipts" mapstructure:"read_receipts"`
	MessageReactions bool `yaml:"message_reactions" mapstructure:"message_reactions"`
	MessageEditing   bool `yaml:"message_editing" mapstructure:"message_editing"`
	MessageDeletion  bool `yaml:"message_deletion" mapstructure:"message_deletion"`
	FileAttachments  bool `yaml:"file_attachments" mapstructure:"file_attachments"`
	VoiceMessages    bool `yaml:"voice_messages" mapstructure:"voice_messages"`
	VideoMessages    bool `yaml:"video_messages" mapstructure:"video_messages"`
	MessageSearch    bool `yaml:"message_search" mapstructure:"message_search"`
	MessageThreads   bool `yaml:"message_threads" mapstructure:"message_threads"`
}

type LimitsConfig struct {
	MaxMessageLength         int `yaml:"max_message_length" mapstructure:"max_message_length"`
	MaxAttachmentsPerMessage int `yaml:"max_attachments_per_message" mapstructure:"max_attachments_per_message"`
	MaxMessagesPerRequest    int `yaml:"max_messages_per_request" mapstructure:"max_messages_per_request"`
	ConversationHistoryDays  int `yaml:"conversation_history_days" mapstructure:"conversation_history_days"`
	UserConversationsLimit   int `yaml:"user_conversations_limit" mapstructure:"user_conversations_limit"`
}
