package env

import (
	"os"
)

func EnvVar(key string, defaultValue string) string {
	value := os.Getenv(key)

	if value == "" {
		value = defaultValue
	}

	return value
}
