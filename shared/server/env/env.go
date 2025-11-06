package env

import (
	"os"

	"github.com/subosito/gotenv"
)

func LoadEnv(filenames ...string) error {
	var possibleFilenames []string
	if len(filenames) == 0 {
		possibleFilenames = []string{
			".env.local",
			".env." + os.Getenv("APP_ENV") + ".local",
			".env." + os.Getenv("APP_ENV"),
			".env",
		}
	} else {
		possibleFilenames = filenames
	}

	for _, filename := range possibleFilenames {
		if _, err := os.Stat(filename); err == nil {
			return gotenv.Load(filename)
		}
	}
	return nil
}

func GetEnv(key string, def ...string) string {
	var defaultVal string
	if len(def) > 0 {
		defaultVal = def[0]
	}
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
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
