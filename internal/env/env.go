package env

import (
	"os"
	"strconv"
	"time"
)

func GetString(key, fallback string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	return val
}

func GetInt(key string, fallback int) int {
	valStr, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	valAsInt, err := strconv.Atoi(valStr)
	if err != nil {
		return fallback
	}
	return valAsInt
}

func GetDuration(key string, fallback time.Duration) time.Duration {
	valStr, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	d, err := time.ParseDuration(valStr)
	if err != nil {
		return fallback
	}
	return d
}
