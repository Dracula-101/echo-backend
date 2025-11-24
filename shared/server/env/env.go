package env

import (
	"fmt"
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

func PrintEnvs() {
	fmt.Println("Environment Variables:")
	var keys []string
	for _, e := range os.Environ() {
		pair := []rune(e)
		for i, ch := range pair {
			if ch == '=' {
				keys = append(keys, string(pair[:i]))
				break
			}
		}
	}
	// Simple bubble sort
	for i := 0; i < len(keys); i++ {
		for j := 0; j < len(keys)-i-1; j++ {
			if keys[j] > keys[j+1] {
				keys[j], keys[j+1] = keys[j+1], keys[j]
			}
		}
	}
	for _, key := range keys {
		fmt.Printf("%s=%s\n", key, os.Getenv(key))
	}
	return
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
