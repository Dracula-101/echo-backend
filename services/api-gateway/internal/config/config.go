package config

import (
	"time"
)

type Config struct {
	Service      ServiceMetadata          `yaml:"service" mapstructure:"service"`
	Server       ServerConfig             `yaml:"server" mapstructure:"server"`
	Services     map[string]ServiceConfig `yaml:"services" mapstructure:"services"`
	RouterGroups []RouterGroup            `yaml:"router_groups" mapstructure:"router_groups"`
	RateLimit    RateLimitConfig          `yaml:"ratelimit" mapstructure:"ratelimit"`
	Security     SecurityConfig           `yaml:"security" mapstructure:"security"`
	LoadBalance  LoadBalanceConfig        `yaml:"loadbalance" mapstructure:"loadbalance"`
	Monitoring   MonitoringConfig         `yaml:"monitoring" mapstructure:"monitoring"`
	Discovery    DiscoveryConfig          `yaml:"discovery" mapstructure:"discovery"`
	Shutdown     ShutdownConfig           `yaml:"shutdown" mapstructure:"shutdown"`
}

type ServiceMetadata struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	Description string `yaml:"description"`
	Environment string `yaml:"environment"`
}

type ServerConfig struct {
	Host              string        `yaml:"host"`
	Port              int           `yaml:"port"`
	ReadTimeout       time.Duration `yaml:"read_timeout"`
	WriteTimeout      time.Duration `yaml:"write_timeout"`
	IdleTimeout       time.Duration `yaml:"idle_timeout"`
	ShutdownTimeout   time.Duration `yaml:"shutdown_timeout"`
	MaxHeaderBytes    int           `yaml:"max_header_bytes"`
	EnableCompression bool          `yaml:"enable_compression"`
	TLSEnabled        bool          `yaml:"tls_enabled"`
	TLSCertFile       string        `yaml:"tls_cert_file"`
	TLSKeyFile        string        `yaml:"tls_key_file"`
	TLSMinVersion     string        `yaml:"tls_min_version"`
	JWTConfig         JWTConfig     `yaml:"jwt" mapstructure:"jwt"`
}

type JWTConfig struct {
	AccessTokenTTL  time.Duration `yaml:"access_token_ttl" mapstructure:"access_token_ttl"`
	RefreshTokenTTL time.Duration `yaml:"refresh_token_ttl" mapstructure:"refresh_token_ttl"`
	SecretKey       string        `yaml:"secret_key" mapstructure:"secret_key"`
	Issuer          string        `yaml:"issuer" mapstructure:"issuer"`
	Audience        string        `yaml:"audience" mapstructure:"audience"`
	Leeway          time.Duration `yaml:"leeway" mapstructure:"leeway"`
	SkipPaths       []string      `yaml:"skip_paths" mapstructure:"skip_paths"`
}
type ServiceConfig struct {
	Protocol       string               `yaml:"protocol"`
	Addresses      []string             `yaml:"addresses"`
	HealthCheck    HealthCheckConfig    `yaml:"health_check"`
	LoadBalancer   string               `yaml:"loadbalancer"`
	CircuitBreaker CircuitBreakerConfig `yaml:"circuit_breaker"`
	Timeout        time.Duration        `yaml:"timeout"`
	RetryAttempts  int                  `yaml:"retry_attempts"`
}

type HealthCheckConfig struct {
	Enabled          bool          `yaml:"enabled"`
	Path             string        `yaml:"path"`
	Interval         time.Duration `yaml:"interval"`
	Timeout          time.Duration `yaml:"timeout"`
	FailureThreshold int           `yaml:"failure_threshold"`
}

type CircuitBreakerConfig struct {
	Enabled          bool          `yaml:"enabled"`
	Threshold        int           `yaml:"threshold"`
	Timeout          time.Duration `yaml:"timeout"`
	HalfOpenRequests int           `yaml:"half_open_requests"`
}

type RouterGroup struct {
	Name      string   `yaml:"name"`
	Prefix    string   `yaml:"prefix"`
	Service   string   `yaml:"service"`
	Transform bool     `yaml:"transform"`
	Methods   []string `yaml:"methods"`
}

type RateLimitConfig struct {
	Enabled       bool                     `yaml:"enabled"`
	Store         string                   `yaml:"store"`
	RedisAddr     string                   `yaml:"redis_addr"`
	RedisPassword string                   `yaml:"redis_password"`
	RedisDB       int                      `yaml:"redis_db"`
	Global        RateLimitRule            `yaml:"global"`
	Endpoints     map[string]RateLimitRule `yaml:"endpoints"`
}

type RateLimitRule struct {
	Requests int           `yaml:"requests"`
	Window   time.Duration `yaml:"window"`
	Strategy string        `yaml:"strategy"`
}

type SecurityConfig struct {
	AllowedOrigins   []string          `yaml:"allowed_origins"`
	AllowedMethods   []string          `yaml:"allowed_methods"`
	AllowedHeaders   []string          `yaml:"allowed_headers"`
	ExposedHeaders   []string          `yaml:"exposed_headers"`
	AllowCredentials bool              `yaml:"allow_credentials"`
	MaxAge           int               `yaml:"max_age"`
	SecurityHeaders  map[string]string `yaml:"security_headers"`
	MaxBodySize      int64             `yaml:"max_body_size"`
	PathLimits       map[string]int64  `yaml:"path_limits"`
}

type LoadBalanceConfig struct {
	DefaultStrategy string `yaml:"default_strategy"`
}

type MonitoringConfig struct {
	Enabled           bool    `yaml:"enabled"`
	MetricsEnabled    bool    `yaml:"metrics_enabled"`
	MetricsPath       string  `yaml:"metrics_path"`
	HealthPath        string  `yaml:"health_path"`
	TracingEnabled    bool    `yaml:"tracing_enabled"`
	TracingEndpoint   string  `yaml:"tracing_endpoint"`
	TracingSampleRate float64 `yaml:"tracing_sample_rate"`
	LogLevel          string  `yaml:"log_level"`
	LogFormat         string  `yaml:"log_format"`
	LogOutput         string  `yaml:"log_output"`
}

type DiscoveryConfig struct {
	Enabled         bool             `yaml:"enabled"`
	Provider        string           `yaml:"provider"`
	Consul          ConsulConfig     `yaml:"consul"`
	Etcd            EtcdConfig       `yaml:"etcd"`
	Kubernetes      KubernetesConfig `yaml:"kubernetes"`
	RefreshInterval time.Duration    `yaml:"refresh_interval"`
}

type ConsulConfig struct {
	Address    string `yaml:"address"`
	Scheme     string `yaml:"scheme"`
	Datacenter string `yaml:"datacenter"`
	Token      string `yaml:"token"`
}

type EtcdConfig struct {
	Endpoints []string      `yaml:"endpoints"`
	Timeout   time.Duration `yaml:"timeout"`
}

type KubernetesConfig struct {
	Namespace     string `yaml:"namespace"`
	LabelSelector string `yaml:"label_selector"`
}

type ShutdownConfig struct {
	Timeout            time.Duration `yaml:"timeout"`
	WaitForConnections bool          `yaml:"wait_for_connections"`
	DrainTimeout       time.Duration `yaml:"drain_timeout"`
}
