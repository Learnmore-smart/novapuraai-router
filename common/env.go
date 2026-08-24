package common

import (
	"fmt"
	"os"
	"strconv"
)

// LookupEnv reports whether an environment variable is present separately
// from whether it contains a non-empty value. Configuration that supports
// fixed, fail-closed overrides needs this distinction.
func LookupEnv(env string) (string, bool) {
	if env == "" {
		return "", false
	}
	return os.LookupEnv(env)
}

func GetEnvOrDefault(env string, defaultValue int) int {
	if env == "" || os.Getenv(env) == "" {
		return defaultValue
	}
	num, err := strconv.Atoi(os.Getenv(env))
	if err != nil {
		SysError(fmt.Sprintf("failed to parse %s: %s, using default value: %d", env, err.Error(), defaultValue))
		return defaultValue
	}
	return num
}

func GetEnvOrDefaultString(env string, defaultValue string) string {
	if env == "" || os.Getenv(env) == "" {
		return defaultValue
	}
	return os.Getenv(env)
}

func GetEnvOrDefaultBool(env string, defaultValue bool) bool {
	if env == "" || os.Getenv(env) == "" {
		return defaultValue
	}
	b, err := strconv.ParseBool(os.Getenv(env))
	if err != nil {
		SysError(fmt.Sprintf("failed to parse boolean %s: invalid boolean, using default value: %t", env, defaultValue))
		return defaultValue
	}
	return b
}
