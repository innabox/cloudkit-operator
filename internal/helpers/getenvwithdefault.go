// Package helpers contains small utility methods that don't belong anywhere else.
package helpers

import (
	"os"
	"strconv"
	"time"
)

func GetEnvWithDefault[T any](name string, defaultValue T) (value T) {
	val := os.Getenv(name)
	if val == "" {
		return defaultValue
	}

	var result any
	var zero T

	switch any(zero).(type) {
	case string:
		result = val
	case int:
		v, err := strconv.Atoi(val)
		if err != nil {
			return defaultValue
		}
		result = v
	case bool:
		v, err := strconv.ParseBool(val)
		if err != nil {
			return defaultValue
		}
		result = v
	case float64:
		v, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return defaultValue
		}
		result = v
	case time.Duration:
		v, err := time.ParseDuration(val)
		if err != nil {
			return defaultValue
		}
		result = v
	default:
		return defaultValue
	}

	return result.(T)
}
