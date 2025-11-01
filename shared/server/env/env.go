package env

import (
	"os"

	"github.com/subosito/gotenv"
)

func LoadEnv(filenames ...string) error {
	return gotenv.Load(filenames...)
}

func GetEnv(key string, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func MustGetEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("missing required env: " + key)
	}
	return v
}

func IsProduction() bool {
	return os.Getenv("APP_ENV") == EnvProduction
}

func IsDevelopment() bool {
	return os.Getenv("APP_ENV") == EnvDevelopment
}

func IsTest() bool {
	return os.Getenv("APP_ENV") == EnvTest
}

func LogLevel() string {
	level := os.Getenv("LOG_LEVEL")
	if level == "" {
		return DefaultLogLevel
	}
	return level
}
