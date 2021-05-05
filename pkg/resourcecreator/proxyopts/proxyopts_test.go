package proxyopts_test

import (
	"strings"
	"testing"

	"github.com/nais/naiserator/pkg/resourcecreator/proxyopts"
	"github.com/nais/naiserator/pkg/test"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

const (
	httpProxy = "http://foo.bar:5224"
	javaProxyOptions = "-Dhttp.proxyHost=foo.bar -Dhttps.proxyHost=foo.bar -Dhttp.proxyPort=5224 -Dhttps.proxyPort=5224 -Dhttp.nonProxyHosts=foo|bar|baz"
)

func TestProxyEnvironmentVariables(t *testing.T) {
	t.Run("Test generation of correct proxy environment variables", func(t *testing.T) {
		var err error
		var noProxy = []string{"foo", "bar", "baz"}

		viper.Set("proxy.address", httpProxy)
		viper.Set("proxy.exclude", noProxy)
		envVars := make([]corev1.EnvVar, 0)
		envVars, err = proxyopts.EnvironmentVariables(envVars)
		nprox := strings.Join(noProxy, ",")
		assert.NoError(t, err)
		assert.Len(t, envVars, 7)
		assert.Equal(t, httpProxy, test.EnvValue(envVars, "HTTP_PROXY"))
		assert.Equal(t, httpProxy, test.EnvValue(envVars, "HTTPS_PROXY"))
		assert.Equal(t, nprox, test.EnvValue(envVars, "NO_PROXY"))
		assert.Equal(t, httpProxy, test.EnvValue(envVars, "http_proxy"))
		assert.Equal(t, httpProxy, test.EnvValue(envVars, "https_proxy"))
		assert.Equal(t, nprox, test.EnvValue(envVars, "no_proxy"))
		assert.Equal(t, javaProxyOptions, test.EnvValue(envVars, "JAVA_PROXY_OPTIONS"))
	})
}
