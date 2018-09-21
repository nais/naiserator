package test

import (
	"os"
	"testing"
)

func EnvWrapper(envVars map[string]string, testFunc func(t *testing.T)) func(t *testing.T) {
	return func(t *testing.T) {
		originalEnvVars := make(map[string]string, len(envVars))
		for k, v := range envVars {
			if value, found := os.LookupEnv(k); found {
				originalEnvVars[k] = value
			}
			os.Setenv(k, v)
		}
		testFunc(t)

		for k := range envVars {
			if value, found := originalEnvVars[k]; found {
				os.Setenv(k, value)
			} else {
				os.Unsetenv(k)
			}

		}
	}
}
