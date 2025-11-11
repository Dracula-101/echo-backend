package config

import "time"

// Config represents the complete application configuration
type Config struct {
	Service       ServiceConfig       `yaml:"service" mapstructure:"service"`
	Server        ServerConfig        `yaml:"server" mapstructure:"server"`
	Database      DatabaseConfig      `yaml:"database" mapstructure:"database"`
	Cache         CacheConfig         `yaml:"cache" mapstructure:"cache"`
	Storage       StorageConfig       `yaml:"storage" mapstructure:"storage"`
	Security      SecurityConfig      `yaml:"security" mapstructure:"security"`
	Logging       LoggingConfig       `yaml:"logging" mapstructure:"logging"`
	Observability ObservabilityConfig `yaml:"observability" mapstructure:"observability"`
	Shutdown      ShutdownConfig      `yaml:"shutdown" mapstructure:"shutdown"`
	Features      FeaturesConfig      `yaml:"features" mapstructure:"features"`
	Processing    ProcessingConfig    `yaml:"processing" mapstructure:"processing"`
}

// ServiceConfig contains service metadata
type ServiceConfig struct {
	Name        string `yaml:"name" mapstructure:"name"`
	Version     string `yaml:"version" mapstructure:"version"`
	Environment string `yaml:"environment" mapstructure:"environment"`
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	Port              int           `yaml:"port" mapstructure:"port"`
	Host              string        `yaml:"host" mapstructure:"host"`
	ReadTimeout       time.Duration `yaml:"read_timeout" mapstructure:"read_timeout"`
	WriteTimeout      time.Duration `yaml:"write_timeout" mapstructure:"write_timeout"`
	IdleTimeout       time.Duration `yaml:"idle_timeout" mapstructure:"idle_timeout"`
	ShutdownTimeout   time.Duration `yaml:"shutdown_timeout" mapstructure:"shutdown_timeout"`
	MaxHeaderBytes    int           `yaml:"max_header_bytes" mapstructure:"max_header_bytes"`
	EnableCompression bool          `yaml:"enable_compression" mapstructure:"enable_compression"`
}

// DatabaseConfig contains database configuration
type DatabaseConfig struct {
	Postgres PostgresConfig `yaml:"postgres" mapstructure:"postgres"`
}

// PostgresConfig contains PostgreSQL specific configuration
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
	AutoMigrate     bool          `yaml:"auto_migrate" mapstructure:"auto_migrate"`
	MigrationPath   string        `yaml:"migration_path" mapstructure:"migration_path"`
}

// CacheConfig contains cache configuration
type CacheConfig struct {
	Enabled     bool        `yaml:"enabled" mapstructure:"enabled"`
	RedisConfig RedisConfig `yaml:"redis" mapstructure:"redis"`
}

// RedisConfig contains Redis specific configuration
type RedisConfig struct {
	RedisHost         string        `yaml:"host" mapstructure:"host"`
	RedisPort         int           `yaml:"port" mapstructure:"port"`
	RedisPassword     string        `yaml:"password" mapstructure:"password"`
	RedisDB           int           `yaml:"db" mapstructure:"db"`
	RedisDialTimeout  time.Duration `yaml:"dial_timeout" mapstructure:"dial_timeout"`
	RedisReadTimeout  time.Duration `yaml:"read_timeout" mapstructure:"read_timeout"`
	RedisWriteTimeout time.Duration `yaml:"write_timeout" mapstructure:"write_timeout"`
	RedisPoolSize     int           `yaml:"pool_size" mapstructure:"pool_size"`
	RedisMinIdleConns int           `yaml:"min_idle_conns" mapstructure:"min_idle_conns"`
	RedisMaxRetries   int           `yaml:"max_retries" mapstructure:"max_retries"`
	RedisPoolTimeout  time.Duration `yaml:"pool_timeout" mapstructure:"pool_timeout"`
}

// StorageConfig contains cloud storage configuration
type StorageConfig struct {
	Provider        string        `yaml:"provider" mapstructure:"provider"` // r2, s3, local
	Bucket          string        `yaml:"bucket" mapstructure:"bucket"`
	Region          string        `yaml:"region" mapstructure:"region"`
	AccessKeyID     string        `yaml:"access_key_id" mapstructure:"access_key_id"`
	SecretAccessKey string        `yaml:"secret_access_key" mapstructure:"secret_access_key"`
	Endpoint        string        `yaml:"endpoint" mapstructure:"endpoint"`
	PublicURL       string        `yaml:"public_url" mapstructure:"public_url"`
	UseCDN          bool          `yaml:"use_cdn" mapstructure:"use_cdn"`
	CDNBaseURL      string        `yaml:"cdn_base_url" mapstructure:"cdn_base_url"`
	MaxFileSize     int64         `yaml:"max_file_size" mapstructure:"max_file_size"`
	AllowedTypes    []string      `yaml:"allowed_types" mapstructure:"allowed_types"`
	UploadTimeout   time.Duration `yaml:"upload_timeout" mapstructure:"upload_timeout"`
}

// SecurityConfig contains security configuration
type SecurityConfig struct {
	AllowedOrigins   string                `yaml:"allowed_origins" mapstructure:"allowed_origins"`
	AllowedMethods   string                `yaml:"allowed_methods" mapstructure:"allowed_methods"`
	AllowedHeaders   string                `yaml:"allowed_headers" mapstructure:"allowed_headers"`
	AllowCredentials bool                  `yaml:"allow_credentials" mapstructure:"allow_credentials"`
	MaxAge           int                   `yaml:"max_age" mapstructure:"max_age"`
	SecurityHeaders  SecurityHeadersConfig `yaml:"security_headers" mapstructure:"security_headers"`
	MaxBodySize      int64                 `yaml:"max_body_size" mapstructure:"max_body_size"`
	RateLimit        RateLimitConfig       `yaml:"rate_limit" mapstructure:"rate_limit"`
}

// SecurityHeadersConfig contains security headers configuration
type SecurityHeadersConfig struct {
	XFrameOptions           string `yaml:"x_frame_options" mapstructure:"x_frame_options"`
	XContentTypeOptions     string `yaml:"x_content_type_options" mapstructure:"x_content_type_options"`
	XXSSProtection          string `yaml:"x_xss_protection" mapstructure:"x_xss_protection"`
	StrictTransportSecurity string `yaml:"strict_transport_security" mapstructure:"strict_transport_security"`
	ContentSecurityPolicy   string `yaml:"content_security_policy" mapstructure:"content_security_policy"`
}

// RateLimitConfig contains rate limiting configuration
type RateLimitConfig struct {
	Enabled   bool                    `yaml:"enabled" mapstructure:"enabled"`
	Window    time.Duration           `yaml:"window" mapstructure:"window"`
	Global    GlobalRateLimitConfig   `yaml:"global" mapstructure:"global"`
	Endpoints EndpointRateLimitConfig `yaml:"endpoints" mapstructure:"endpoints"`
}

// GlobalRateLimitConfig contains global rate limit settings
type GlobalRateLimitConfig struct {
	Requests int `yaml:"requests" mapstructure:"requests"`
}

// EndpointRateLimitConfig contains per-endpoint rate limits
type EndpointRateLimitConfig struct {
	Upload   EndpointLimit `yaml:"upload" mapstructure:"upload"`
	Download EndpointLimit `yaml:"download" mapstructure:"download"`
	Delete   EndpointLimit `yaml:"delete" mapstructure:"delete"`
	List     EndpointLimit `yaml:"list" mapstructure:"list"`
}

// EndpointLimit represents rate limit for a specific endpoint
type EndpointLimit struct {
	Requests int           `yaml:"requests" mapstructure:"requests"`
	Window   time.Duration `yaml:"window" mapstructure:"window"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level      string        `yaml:"level" mapstructure:"level"`
	Format     string        `yaml:"format" mapstructure:"format"`
	Output     string        `yaml:"output" mapstructure:"output"`
	TimeFormat string        `yaml:"time_format" mapstructure:"time_format"`
	File       LogFileConfig `yaml:"file" mapstructure:"file"`
}

// LogFileConfig contains file logging configuration
type LogFileConfig struct {
	Path       string `yaml:"path" mapstructure:"path"`
	MaxSize    int    `yaml:"max_size" mapstructure:"max_size"`
	MaxBackups int    `yaml:"max_backups" mapstructure:"max_backups"`
	MaxAge     int    `yaml:"max_age" mapstructure:"max_age"`
	Compress   bool   `yaml:"compress" mapstructure:"compress"`
}

// ObservabilityConfig contains observability configuration
type ObservabilityConfig struct {
	Metrics MetricsConfig `yaml:"metrics" mapstructure:"metrics"`
	Tracing TracingConfig `yaml:"tracing" mapstructure:"tracing"`
	Health  HealthConfig  `yaml:"health" mapstructure:"health"`
}

// MetricsConfig contains metrics configuration
type MetricsConfig struct {
	Enabled  bool   `yaml:"enabled" mapstructure:"enabled"`
	Endpoint string `yaml:"endpoint" mapstructure:"endpoint"`
}

// TracingConfig contains tracing configuration
type TracingConfig struct {
	Enabled    bool    `yaml:"enabled" mapstructure:"enabled"`
	Provider   string  `yaml:"provider" mapstructure:"provider"`
	Endpoint   string  `yaml:"endpoint" mapstructure:"endpoint"`
	SampleRate float64 `yaml:"sample_rate" mapstructure:"sample_rate"`
}

// HealthConfig contains health check configuration
type HealthConfig struct {
	Enabled  bool   `yaml:"enabled" mapstructure:"enabled"`
	Endpoint string `yaml:"endpoint" mapstructure:"endpoint"`
}

// ShutdownConfig contains graceful shutdown configuration
type ShutdownConfig struct {
	Timeout            time.Duration `yaml:"timeout" mapstructure:"timeout"`
	WaitForConnections bool          `yaml:"wait_for_connections" mapstructure:"wait_for_connections"`
	DrainTimeout       time.Duration `yaml:"drain_timeout" mapstructure:"drain_timeout"`
}

// FeaturesConfig contains feature flags
type FeaturesConfig struct {
	ImageProcessing   ImageProcessingConfig   `yaml:"image_processing" mapstructure:"image_processing"`
	VideoProcessing   VideoProcessingConfig   `yaml:"video_processing" mapstructure:"video_processing"`
	Thumbnails        ThumbnailsConfig        `yaml:"thumbnails" mapstructure:"thumbnails"`
	VirusScanning     VirusScanningConfig     `yaml:"virus_scanning" mapstructure:"virus_scanning"`
	ContentModeration ContentModerationConfig `yaml:"content_moderation" mapstructure:"content_moderation"`
	Encryption        EncryptionConfig        `yaml:"encryption" mapstructure:"encryption"`
	Deduplication     DeduplicationConfig     `yaml:"deduplication" mapstructure:"deduplication"`
	Albums            AlbumsConfig            `yaml:"albums" mapstructure:"albums"`
	StickerPacks      StickerPacksConfig      `yaml:"sticker_packs" mapstructure:"sticker_packs"`
}

// ImageProcessingConfig contains image processing configuration
type ImageProcessingConfig struct {
	Enabled        bool     `yaml:"enabled" mapstructure:"enabled"`
	MaxWidth       int      `yaml:"max_width" mapstructure:"max_width"`
	MaxHeight      int      `yaml:"max_height" mapstructure:"max_height"`
	Quality        int      `yaml:"quality" mapstructure:"quality"`
	AllowedFormats []string `yaml:"allowed_formats" mapstructure:"allowed_formats"`
	AutoOrient     bool     `yaml:"auto_orient" mapstructure:"auto_orient"`
	StripMetadata  bool     `yaml:"strip_metadata" mapstructure:"strip_metadata"`
}

// VideoProcessingConfig contains video processing configuration
type VideoProcessingConfig struct {
	Enabled            bool     `yaml:"enabled" mapstructure:"enabled"`
	MaxDuration        int      `yaml:"max_duration" mapstructure:"max_duration"`
	AllowedFormats     []string `yaml:"allowed_formats" mapstructure:"allowed_formats"`
	TranscodingEnabled bool     `yaml:"transcoding_enabled" mapstructure:"transcoding_enabled"`
}

// ThumbnailsConfig contains thumbnail generation configuration
type ThumbnailsConfig struct {
	Enabled    bool          `yaml:"enabled" mapstructure:"enabled"`
	SmallSize  ThumbnailSize `yaml:"small_size" mapstructure:"small_size"`
	MediumSize ThumbnailSize `yaml:"medium_size" mapstructure:"medium_size"`
	LargeSize  ThumbnailSize `yaml:"large_size" mapstructure:"large_size"`
	Format     string        `yaml:"format" mapstructure:"format"`
	Quality    int           `yaml:"quality" mapstructure:"quality"`
}

// ThumbnailSize represents thumbnail dimensions
type ThumbnailSize struct {
	Width  int `yaml:"width" mapstructure:"width"`
	Height int `yaml:"height" mapstructure:"height"`
}

// VirusScanningConfig contains virus scanning configuration
type VirusScanningConfig struct {
	Enabled  bool   `yaml:"enabled" mapstructure:"enabled"`
	Provider string `yaml:"provider" mapstructure:"provider"`
	Endpoint string `yaml:"endpoint" mapstructure:"endpoint"`
	APIKey   string `yaml:"api_key" mapstructure:"api_key"`
}

// ContentModerationConfig contains content moderation configuration
type ContentModerationConfig struct {
	Enabled   bool    `yaml:"enabled" mapstructure:"enabled"`
	Provider  string  `yaml:"provider" mapstructure:"provider"`
	Endpoint  string  `yaml:"endpoint" mapstructure:"endpoint"`
	APIKey    string  `yaml:"api_key" mapstructure:"api_key"`
	Threshold float64 `yaml:"threshold" mapstructure:"threshold"`
}

// EncryptionConfig contains encryption configuration
type EncryptionConfig struct {
	Enabled   bool   `yaml:"enabled" mapstructure:"enabled"`
	Algorithm string `yaml:"algorithm" mapstructure:"algorithm"`
	KeyID     string `yaml:"key_id" mapstructure:"key_id"`
}

// DeduplicationConfig contains deduplication configuration
type DeduplicationConfig struct {
	Enabled   bool   `yaml:"enabled" mapstructure:"enabled"`
	Algorithm string `yaml:"algorithm" mapstructure:"algorithm"`
}

// AlbumsConfig contains albums feature configuration
type AlbumsConfig struct {
	Enabled          bool `yaml:"enabled" mapstructure:"enabled"`
	MaxAlbumsPerUser int  `yaml:"max_albums_per_user" mapstructure:"max_albums_per_user"`
}

// StickerPacksConfig contains sticker packs configuration
type StickerPacksConfig struct {
	Enabled             bool `yaml:"enabled" mapstructure:"enabled"`
	MaxStickersPerPack  int  `yaml:"max_stickers_per_pack" mapstructure:"max_stickers_per_pack"`
	MaxInstalledPerUser int  `yaml:"max_installed_per_user" mapstructure:"max_installed_per_user"`
}

// ProcessingConfig contains media processing configuration
type ProcessingConfig struct {
	Workers           int           `yaml:"workers" mapstructure:"workers"`
	QueueSize         int           `yaml:"queue_size" mapstructure:"queue_size"`
	MaxAttempts       int           `yaml:"max_attempts" mapstructure:"max_attempts"`
	ProcessingTimeout time.Duration `yaml:"processing_timeout" mapstructure:"processing_timeout"`
}
