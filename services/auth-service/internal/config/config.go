package config

import "time"

// Config represents the complete application configuration
type Config struct {
	Service       ServiceConfig       `yaml:"service" mapstructure:"service"`
	Server        ServerConfig        `yaml:"server" mapstructure:"server"`
	Database      DatabaseConfig      `yaml:"database" mapstructure:"database"`
	Cache         CacheConfig         `yaml:"cache" mapstructure:"cache"`
	Auth          AuthConfig          `yaml:"auth" mapstructure:"auth"`
	Security      SecurityConfig      `yaml:"security" mapstructure:"security"`
	Logging       LoggingConfig       `yaml:"logging" mapstructure:"logging"`
	Email         EmailConfig         `yaml:"email" mapstructure:"email"`
	Observability ObservabilityConfig `yaml:"observability" mapstructure:"observability"`
	Shutdown      ShutdownConfig      `yaml:"shutdown" mapstructure:"shutdown"`
	Features      FeaturesConfig      `yaml:"features" mapstructure:"features"`
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

// AuthConfig contains authentication configuration
type AuthConfig struct {
	JWT               JWTConfig               `yaml:"jwt" mapstructure:"jwt"`
	Password          PasswordConfig          `yaml:"password" mapstructure:"password"`
	EmailVerification EmailVerificationConfig `yaml:"email_verification" mapstructure:"email_verification"`
	PasswordReset     PasswordResetConfig     `yaml:"password_reset" mapstructure:"password_reset"`
	Session           SessionConfig           `yaml:"session" mapstructure:"session"`
}

// JWTConfig contains JWT configuration
type JWTConfig struct {
	AccessTokenTTL  time.Duration `yaml:"access_token_ttl" mapstructure:"access_token_ttl"`
	RefreshTokenTTL time.Duration `yaml:"refresh_token_ttl" mapstructure:"refresh_token_ttl"`
	SecretKey       string        `yaml:"secret_key" mapstructure:"secret_key"`
	Issuer          string        `yaml:"issuer" mapstructure:"issuer"`
	Audience        string        `yaml:"audience" mapstructure:"audience"`
}

// PasswordConfig contains password policy configuration
type PasswordConfig struct {
	MinLength        int  `yaml:"min_length" mapstructure:"min_length"`
	RequireUppercase bool `yaml:"require_uppercase" mapstructure:"require_uppercase"`
	RequireLowercase bool `yaml:"require_lowercase" mapstructure:"require_lowercase"`
	RequireNumber    bool `yaml:"require_number" mapstructure:"require_number"`
	RequireSpecial   bool `yaml:"require_special" mapstructure:"require_special"`
	BcryptCost       int  `yaml:"bcrypt_cost" mapstructure:"bcrypt_cost"`
}

// EmailVerificationConfig contains email verification configuration
type EmailVerificationConfig struct {
	Enabled        bool          `yaml:"enabled" mapstructure:"enabled"`
	TokenTTL       time.Duration `yaml:"token_ttl" mapstructure:"token_ttl"`
	ResendCooldown time.Duration `yaml:"resend_cooldown" mapstructure:"resend_cooldown"`
}

// PasswordResetConfig contains password reset configuration
type PasswordResetConfig struct {
	Enabled  bool          `yaml:"enabled" mapstructure:"enabled"`
	TokenTTL time.Duration `yaml:"token_ttl" mapstructure:"token_ttl"`
}

// SessionConfig contains session management configuration
type SessionConfig struct {
	MaxActiveSessions int           `yaml:"max_active_sessions" mapstructure:"max_active_sessions"`
	IdleTimeout       time.Duration `yaml:"idle_timeout" mapstructure:"idle_timeout"`
	AbsoluteTimeout   time.Duration `yaml:"absolute_timeout" mapstructure:"absolute_timeout"`
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
	Register           EndpointLimit `yaml:"register" mapstructure:"register"`
	Login              EndpointLimit `yaml:"login" mapstructure:"login"`
	ForgotPassword     EndpointLimit `yaml:"forgot_password" mapstructure:"forgot_password"`
	ResendVerification EndpointLimit `yaml:"resend_verification" mapstructure:"resend_verification"`
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

// EmailConfig contains email configuration
type EmailConfig struct {
	Enabled   bool           `yaml:"enabled" mapstructure:"enabled"`
	Provider  string         `yaml:"provider" mapstructure:"provider"`
	SMTP      SMTPConfig     `yaml:"smtp" mapstructure:"smtp"`
	Templates TemplateConfig `yaml:"templates" mapstructure:"templates"`
}

// SMTPConfig contains SMTP configuration
type SMTPConfig struct {
	Host      string `yaml:"host" mapstructure:"host"`
	Port      int    `yaml:"port" mapstructure:"port"`
	Username  string `yaml:"username" mapstructure:"username"`
	Password  string `yaml:"password" mapstructure:"password"`
	FromEmail string `yaml:"from_email" mapstructure:"from_email"`
	FromName  string `yaml:"from_name" mapstructure:"from_name"`
	UseTLS    bool   `yaml:"use_tls" mapstructure:"use_tls"`
}

// TemplateConfig contains email template configuration
type TemplateConfig struct {
	VerificationSubject  string `yaml:"verification_subject" mapstructure:"verification_subject"`
	PasswordResetSubject string `yaml:"password_reset_subject" mapstructure:"password_reset_subject"`
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
	OAuth     OAuthConfig     `yaml:"oauth" mapstructure:"oauth"`
	TwoFactor TwoFactorConfig `yaml:"two_factor" mapstructure:"two_factor"`
	MagicLink MagicLinkConfig `yaml:"magic_link" mapstructure:"magic_link"`
}

// OAuthConfig contains OAuth configuration
type OAuthConfig struct {
	Enabled   bool                 `yaml:"enabled" mapstructure:"enabled"`
	Providers OAuthProvidersConfig `yaml:"providers" mapstructure:"providers"`
}

// OAuthProvidersConfig contains OAuth provider configurations
type OAuthProvidersConfig struct {
	Google OAuthProviderConfig `yaml:"google" mapstructure:"google"`
	GitHub OAuthProviderConfig `yaml:"github" mapstructure:"github"`
}

// OAuthProviderConfig contains single OAuth provider configuration
type OAuthProviderConfig struct {
	Enabled      bool   `yaml:"enabled" mapstructure:"enabled"`
	ClientID     string `yaml:"client_id" mapstructure:"client_id"`
	ClientSecret string `yaml:"client_secret" mapstructure:"client_secret"`
}

// TwoFactorConfig contains 2FA configuration
type TwoFactorConfig struct {
	Enabled bool   `yaml:"enabled" mapstructure:"enabled"`
	Issuer  string `yaml:"issuer" mapstructure:"issuer"`
}

// MagicLinkConfig contains magic link configuration
type MagicLinkConfig struct {
	Enabled  bool          `yaml:"enabled" mapstructure:"enabled"`
	TokenTTL time.Duration `yaml:"token_ttl" mapstructure:"token_ttl"`
}
