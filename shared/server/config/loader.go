package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/viper"
)

type LoadOptions struct {
	ConfigPath  string
	Environment string
	ServiceName string
	ConfigPaths []string
}

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
		result := make(map[string]interface{})
		for key, value := range v {
			result[key] = expandValue(value)
		}
		return result
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

func Load[T any](opts LoadOptions) (*T, error) {
	v := viper.New()

	if opts.ConfigPath != "" {
		v.SetConfigFile(opts.ConfigPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")

		v.AddConfigPath("./configs")
		v.AddConfigPath("./config")
		v.AddConfigPath("../configs")
		v.AddConfigPath("../../configs")

		if opts.ServiceName != "" {
			v.AddConfigPath(fmt.Sprintf("/etc/%s", opts.ServiceName))
			v.AddConfigPath(fmt.Sprintf("$HOME/.%s", opts.ServiceName))
		}

		for _, path := range opts.ConfigPaths {
			v.AddConfigPath(path)
		}
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		return nil, fmt.Errorf("config file not found: %w", err)
	}

	allSettings := v.AllSettings()
	expandedSettings := expandValue(allSettings).(map[string]interface{})

	if opts.Environment != "" {
		envConfigPath := getEnvConfigPath(v.ConfigFileUsed(), opts.Environment)
		if _, err := os.Stat(envConfigPath); err == nil {
			envViper := viper.New()
			envViper.SetConfigFile(envConfigPath)
			if err := envViper.ReadInConfig(); err == nil {
				envSettings := envViper.AllSettings()
				expandedEnvSettings := expandValue(envSettings).(map[string]interface{})
				expandedSettings = mergeMaps(expandedSettings, expandedEnvSettings)
			}
		}
	}

	v2 := viper.New()
	if err := v2.MergeConfigMap(expandedSettings); err != nil {
		return nil, fmt.Errorf("failed to merge expanded settings: %w", err)
	}

	var cfg T
	if err := v2.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

func mergeMaps(base, override map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for k, v := range base {
		result[k] = v
	}

	for k, v := range override {
		if baseVal, exists := result[k]; exists {
			if baseMap, ok := baseVal.(map[string]interface{}); ok {
				if overrideMap, ok := v.(map[string]interface{}); ok {
					result[k] = mergeMaps(baseMap, overrideMap)
					continue
				}
			}
		}
		result[k] = v
	}

	return result
}

func getEnvConfigPath(configPath string, env string) string {
	dir := filepath.Dir(configPath)
	ext := filepath.Ext(configPath)
	name := strings.TrimSuffix(filepath.Base(configPath), ext)
	return filepath.Join(dir, fmt.Sprintf("%s.%s%s", name, env, ext))
}
