package proxyopts_test

import (
	"strings"
	"testing"

	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator/proxyopts"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/test"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

const (
	httpProxy        = "http://foo.bar:5224"
	javaProxyOptions = "-Dhttp.proxyHost=foo.bar -Dhttps.proxyHost=foo.bar -Dhttp.proxyPort=5224 -Dhttps.proxyPort=5224 -Dhttp.nonProxyHosts=foo|bar|baz"
)

func TestProxyEnvironmentVariables(t *testing.T) {
	t.Run("Test generation of correct proxy environment variables", func(t *testing.T) {
		var err error
		noProxy := []string{"foo", "bar", "baz"}

		options := resource.Options{}
		options.Proxy = config.Proxy{
			Address: httpProxy,
			Exclude: noProxy,
		}
		var envVars []corev1.EnvVar
		envVars, err = proxyopts.EnvironmentVariables(options)
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
