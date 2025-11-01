package config

import (
	"fmt"
	"os"
	"strings"

	"path/filepath"
	"regexp"

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
		v.AddConfigPath("/etc/auth-service")
		v.AddConfigPath("$HOME/.auth-service")
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

	if override.Server.ReadTimeout != 0 {
		merged.Server.ReadTimeout = override.Server.ReadTimeout
	}

	if override.Server.WriteTimeout != 0 {
		merged.Server.WriteTimeout = override.Server.WriteTimeout
	}

	if override.Server.IdleTimeout != 0 {
		merged.Server.IdleTimeout = override.Server.IdleTimeout
	}

	if override.Server.ShutdownTimeout != 0 {
		merged.Server.ShutdownTimeout = override.Server.ShutdownTimeout
	}

	if override.Database != (DatabaseConfig{}) {
		merged.Database = override.Database
	}

	if override.Cache != (CacheConfig{}) {
		merged.Cache = override.Cache
	}

	

	return &merged
}
