package utils

import (
	"fmt"
	"os"
)

// GetEnv retrieves the value of the environment variable named by key.
// If the variable variable is missing, it returns the optional fallback value and false.
func GetEnv(key string, alt ...string) (string, bool) {
	value, ok := os.LookupEnv(key)
	if !ok {
		if len(alt) > 0 {
			return alt[0], false
		}
		return "", false
	}
	return value, true
}

// MustGetEnv retrieves the value of the environment variable named by key.
// It panics if the variable is not set.
func MustGetEnv(key string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		panic(fmt.Sprintf("missing env var: %s", key))
	}
	return value
}
