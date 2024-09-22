package env

import (
	"os"
	"strconv"
)

func EnvVar(key string, defaultValue string) string {
	value := os.Getenv(key)

	if value == "" {
		value = defaultValue
	}

	return value
}

func EnvVarInt(key string, defaultValue int) int {
	value := os.Getenv(key)

	if value == "" {
		return defaultValue
	}

	intVar, err := strconv.Atoi(value)

	if err != nil {
		return defaultValue
	}

	return intVar
}
