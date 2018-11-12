package resourcecreator_test

import (
	"strconv"
	"testing"

	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test"

	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/nais/naiserator/pkg/vault"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
)

func TestGetDeployment(t *testing.T) {
	app := fixtures.Application()
	app.Spec.Secrets = true

	t.Run("Test deployment with vault", test.EnvWrapper(map[string]string{
		vault.EnvVaultAddr:          "a",
		vault.EnvVaultAuthPath:      "b",
		vault.EnvInitContainerImage: "c",
		vault.EnvVaultKVPath:        "d",
		vault.EnvVaultEnabled:       "e",
	}, func(t *testing.T) {
		deploy, err := resourcecreator.Deployment(app)
		assert.Nil(t, err)
		appContainer := getContainerByName(deploy.Spec.Template.Spec.Containers, app.Name)

		t.Run("user settings is applied", func(t *testing.T) {

			assert.Equal(t, int32(app.Spec.Port), appContainer.Ports[0].ContainerPort)
			assert.Equal(t, app.Name, deploy.Name)
			assert.Equal(t, app.Namespace, deploy.Namespace)
			assert.Equal(t, app.Spec.Team, deploy.Labels["team"])
			assert.Equal(t, app.Name, deploy.Spec.Template.Spec.ServiceAccountName)
			assert.Equal(t, nais.DefaultPortName, appContainer.LivenessProbe.HTTPGet.Port.StrVal)
			assert.Equal(t, app.Spec.Liveness.Path, appContainer.LivenessProbe.HTTPGet.Path)
			assert.Equal(t, int32(app.Spec.Liveness.PeriodSeconds), appContainer.LivenessProbe.PeriodSeconds)
			assert.Equal(t, int32(app.Spec.Liveness.Timeout), appContainer.LivenessProbe.TimeoutSeconds)
			assert.Equal(t, int32(app.Spec.Liveness.FailureThreshold), appContainer.LivenessProbe.FailureThreshold)
			assert.Equal(t, int32(app.Spec.Liveness.InitialDelay), appContainer.LivenessProbe.InitialDelaySeconds)
			assert.Equal(t, nais.DefaultPortName, appContainer.ReadinessProbe.HTTPGet.Port.StrVal)
			assert.Equal(t, app.Spec.Readiness.Path, appContainer.ReadinessProbe.HTTPGet.Path)
			assert.Equal(t, int32(app.Spec.Readiness.PeriodSeconds), appContainer.ReadinessProbe.PeriodSeconds)
			assert.Equal(t, int32(app.Spec.Readiness.Timeout), appContainer.ReadinessProbe.TimeoutSeconds)
			assert.Equal(t, int32(app.Spec.Readiness.FailureThreshold), appContainer.ReadinessProbe.FailureThreshold)
			assert.Equal(t, int32(app.Spec.Readiness.InitialDelay), appContainer.ReadinessProbe.InitialDelaySeconds)
			assert.Equal(t, app.Spec.Resources.Limits.Cpu, appContainer.Resources.Limits.Cpu().String())
			assert.Equal(t, app.Spec.Resources.Limits.Memory, appContainer.Resources.Limits.Memory().String())
			assert.Equal(t, app.Spec.Resources.Requests.Cpu, appContainer.Resources.Requests.Cpu().String())
			assert.Equal(t, app.Spec.Resources.Requests.Memory, appContainer.Resources.Requests.Memory().String())
			assert.Equal(t, strconv.FormatBool(app.Spec.Prometheus.Enabled), deploy.Spec.Template.Annotations["prometheus.io/scrape"])
			assert.Equal(t, app.Spec.Prometheus.Path, deploy.Spec.Template.Annotations["prometheus.io/path"])
			assert.Equal(t, app.Spec.Prometheus.Port, deploy.Spec.Template.Annotations["prometheus.io/port"])
			assert.NotNil(t, getContainerByName(deploy.Spec.Template.Spec.Containers, "elector"), "contains sidecar for leader election")
			assert.NotNil(t, getContainerByName(deploy.Spec.Template.Spec.InitContainers, "vks"), "contains vault initcontainer")
			assert.Equal(t, app.Spec.Env[0].Name, appContainer.Env[0].Name)
			assert.Equal(t, app.Spec.Env[0].Value, appContainer.Env[0].Value)
		})

		t.Run("certificate authority files and configuration is injected", func(t *testing.T) {

			assert.Equal(t, resourcecreator.NAV_TRUSTSTORE_PATH, envValue(appContainer.Env, "NAV_TRUSTSTORE_PATH"))
			assert.Equal(t, resourcecreator.NAV_TRUSTSTORE_PASSWORD, envValue(appContainer.Env, "NAV_TRUSTSTORE_PASSWORD"))
			assert.Equal(t, resourcecreator.CA_BUNDLE_CONFIGMAP_NAME, appContainer.VolumeMounts[0].Name)
			assert.Equal(t, resourcecreator.CA_BUNDLE_CONFIGMAP_NAME, deploy.Spec.Template.Spec.Volumes[0].Name)
			assert.Equal(t, resourcecreator.CA_BUNDLE_CONFIGMAP_NAME, deploy.Spec.Template.Spec.Volumes[0].ConfigMap.Name)
		})
	}))
}

func getContainerByName(containers []v1.Container, name string) *v1.Container {
	for _, v := range containers {
		if v.Name == name {
			return &v
		}
	}

	return nil
}

func envValue(envs []v1.EnvVar, name string) string {
	for _, e := range envs {
		if e.Name == name {
			return e.Value
		}
	}
	return ""
}
