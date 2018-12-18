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

func TestProxyEnvironmentVariables(t *testing.T) {
	t.Run("Test deployment with vault", test.EnvWrapper(map[string]string{
		resourcecreator.PodHttpProxyEnv: httpProxy,
		resourcecreator.PodNoProxyEnv:   noProxy,
	}, func(t *testing.T) {
		var err error
		envVars := make([]corev1.EnvVar, 0)
		envVars, err = resourcecreator.ProxyEnvironmentVariables(envVars)
		assert.NoError(t, err)
		assert.Len(t, envVars, 7)
	}))
}
