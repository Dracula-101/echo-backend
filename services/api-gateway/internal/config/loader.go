package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/viper"
)

func expandEnvVars(s string) string {
	re := regexp.MustCompile(`\$\{([^}:]+)(?::([^}]*))?\}`)
	return re.ReplaceAllStringFunc(s, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}

		varName := parts[1]
		defaultVal := ""
		if len(parts) > 2 {
			defaultVal = parts[2]
		}

		if val := os.Getenv(varName); val != "" {
			return val
		}
		return defaultVal
	})
}

func expandValue(val interface{}) interface{} {
	switch v := val.(type) {
	case string:
		return expandEnvVars(v)
	case map[string]interface{}:
		return expandAllSettings(v)
	case []interface{}:
		arr := make([]interface{}, len(v))
		for i, item := range v {
			arr[i] = expandValue(item)
		}
		return arr
	default:
		return v
	}
}

func expandAllSettings(settings map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for key, val := range settings {
		result[key] = expandValue(val)
	}
	return result
}

func Load(configPath string) (*Config, error) {
	v := viper.New()

	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./configs")
		v.AddConfigPath("./config")
		v.AddConfigPath("../configs")
		v.AddConfigPath("../../configs")
		v.AddConfigPath("/etc/api-gateway")
		v.AddConfigPath("$HOME/.api-gateway")
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	allSettings := v.AllSettings()
	expandedSettings := expandAllSettings(allSettings)

	if err := v.MergeConfigMap(expandedSettings); err != nil {
		return nil, fmt.Errorf("failed to merge expanded settings: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

func LoadFromEnv() (*Config, error) {
	v := viper.New()

	v.SetEnvPrefix("GATEWAY")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	allSettings := v.AllSettings()
	expandedSettings := expandAllSettings(allSettings)

	if err := v.MergeConfigMap(expandedSettings); err != nil {
		return nil, fmt.Errorf("failed to merge expanded settings: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

func LoadWithEnv(configPath string, env string) (*Config, error) {
	cfg, err := Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load base config: %w", err)
	}

	if env != "" {
		envConfigPath := getEnvConfigPath(configPath, env)
		if _, err := os.Stat(envConfigPath); err == nil {
			envCfg, err := Load(envConfigPath)
			if err != nil {
				return nil, fmt.Errorf("failed to load env config: %w", err)
			}
			cfg = mergeConfigs(cfg, envCfg)
		}
	}

	return cfg, nil
}

func getEnvConfigPath(configPath string, env string) string {
	dir := filepath.Dir(configPath)
	ext := filepath.Ext(configPath)
	name := strings.TrimSuffix(filepath.Base(configPath), ext)
	return filepath.Join(dir, fmt.Sprintf("%s.%s%s", name, env, ext))
}

func mergeConfigs(base, override *Config) *Config {
	merged := *base

	if override.Service.Name != "" {
		merged.Service = override.Service
	}

	if override.Server.Port != 0 {
		merged.Server = override.Server
	}

	if len(override.Services) > 0 {
		if merged.Services == nil {
			merged.Services = make(map[string]ServiceConfig)
		}
		for k, v := range override.Services {
			merged.Services[k] = v
		}
	}

	if len(override.RouterGroups) > 0 {
		merged.RouterGroups = override.RouterGroups
	}

	if override.RateLimit.Enabled {
		merged.RateLimit = override.RateLimit
	}

	if len(override.Security.AllowedOrigins) > 0 {
		merged.Security = override.Security
	}

	if override.LoadBalance.DefaultStrategy != "" {
		merged.LoadBalance = override.LoadBalance
	}

	if override.Monitoring.Enabled {
		merged.Monitoring = override.Monitoring
	}

	if override.Discovery.Enabled {
		merged.Discovery = override.Discovery
	}

	if override.Shutdown.Timeout > 0 {
		merged.Shutdown = override.Shutdown
	}

	return &merged
}
