package test_test

import (
	"os"
	"testing"

	"github.com/nais/naiserator/pkg/test"
	"github.com/stretchr/testify/assert"
)

func TestEnvWrapper(t *testing.T) {
	testEnvVars := map[string]string{"testEnv": "testEvn", "testEnv2": "testEnv2", "testEnv3": "testEnv3"}

	os.Setenv("testEnv", "originalValue")

	// Test function has access to env vars set by wrapper
	test.EnvWrapper(testEnvVars, func(t *testing.T) {
		for k, v := range testEnvVars {
			assert.Equal(t, os.Getenv(k), v)
		}
	})(t)

	// Original env variable are preserved
	value, found := os.LookupEnv("testEnv")
	assert.True(t, found)
	assert.Equal(t, "originalValue", value)

	// Wrapped env variables are removed
	_, env2 := os.LookupEnv("testEnv2")
	assert.False(t, env2)

	_, env3 := os.LookupEnv("testEnv3")
	assert.False(t, env3)

	os.Unsetenv("testEnv")

}
