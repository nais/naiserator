package resourcecreator_test

import (
	"testing"

	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

const httpProxy = "http://foo.bar:5224"
const noProxy = "foo,bar,baz"
const javaProxyOptions = "-Dhttp.proxyHost=foo.bar -Dhttps.proxyHost=foo.bar -Dhttp.proxyPort=5224 -Dhttps.proxyPort=5224 -Dhttp.nonProxyHosts=foo|bar|baz"

func TestProxyEnvironmentVariables(t *testing.T) {
	t.Run("Test generation of correct proxy environment variables", test.EnvWrapper(map[string]string{
		resourcecreator.PodHttpProxyEnv: httpProxy,
		resourcecreator.PodNoProxyEnv:   noProxy,
	}, func(t *testing.T) {
		var err error
		envVars := make([]corev1.EnvVar, 0)
		envVars, err = resourcecreator.ProxyEnvironmentVariables(envVars)
		assert.NoError(t, err)
		assert.Len(t, envVars, 7)
		assert.Equal(t, httpProxy, envValue(envVars, "HTTP_PROXY"))
		assert.Equal(t, httpProxy, envValue(envVars, "HTTPS_PROXY"))
		assert.Equal(t, noProxy, envValue(envVars, "NO_PROXY"))
		assert.Equal(t, httpProxy, envValue(envVars, "http_proxy"))
		assert.Equal(t, httpProxy, envValue(envVars, "https_proxy"))
		assert.Equal(t, noProxy, envValue(envVars, "no_proxy"))
		assert.Equal(t, javaProxyOptions, envValue(envVars, "JAVA_PROXY_OPTIONS"))
	}))
}
