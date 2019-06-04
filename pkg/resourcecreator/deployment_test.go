package resourcecreator_test

import (
	"fmt"
	"strconv"
	"testing"

	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/nais/naiserator/pkg/vault"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
)

const clusterName = "my.test.cluster.local"

func TestDeployment(t *testing.T) {

	t.Run("Test deployment with vault", test.EnvWrapper(map[string]string{
		vault.EnvVaultAddr:          "a",
		vault.EnvVaultAuthPath:      "b",
		vault.EnvInitContainerImage: "c",
		vault.EnvVaultKVPath:        "/base/kv",
		vault.EnvVaultEnabled:       "true",
	}, func(t *testing.T) {
		app := fixtures.Application()
		app.Spec.Vault.Enabled = true
		opts := resourcecreator.NewResourceOptions()
		deploy, err := resourcecreator.Deployment(app, opts)
		assert.Nil(t, err)
		appContainer := getContainerByName(deploy.Spec.Template.Spec.Containers, app.Name)

		t.Run("user settings are applied", func(t *testing.T) {
			assert.Equal(t, int32(app.Spec.Port), appContainer.Ports[0].ContainerPort)
			assert.Equal(t, app.Name, deploy.Spec.Template.Spec.ServiceAccountName)
			assert.Equal(t, app.Spec.PreStopHookPath, appContainer.Lifecycle.PreStop.HTTPGet.Path)
			assert.Equal(t, strconv.FormatBool(app.Spec.Prometheus.Enabled), deploy.Spec.Template.Annotations["prometheus.io/scrape"])
			assert.Equal(t, app.Spec.Prometheus.Path, deploy.Spec.Template.Annotations["prometheus.io/path"])
			assert.Equal(t, app.Spec.Prometheus.Port, deploy.Spec.Template.Annotations["prometheus.io/port"])
		})

		t.Run("vault KV path is configured correctly", func(t *testing.T) {
			c := getContainerByName(deploy.Spec.Template.Spec.InitContainers, "vks-0")
			assert.NotNil(t, c, "contains vault initcontainer")
			assert.Equal(t, "/base/kv/app/default", test.EnvVar(c.Env, "VKS_KV_PATH"))
		})

	}))

	t.Run("reflection environment variables are set in the application container", test.EnvWrapper(map[string]string{
		resourcecreator.NaisClusterName: clusterName,
	}, func(t *testing.T) {

		app := fixtures.MinimalApplication()
		app.Spec.Image = "docker/image:latest"
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		opts := resourcecreator.NewResourceOptions()
		deploy, err := resourcecreator.Deployment(app, opts)
		assert.Nil(t, err)

		appContainer := getContainerByName(deploy.Spec.Template.Spec.Containers, app.Name)
		assert.NotNil(t, appContainer)

		assert.Equal(t, app.ObjectMeta.Name, envValue(appContainer.Env, resourcecreator.NaisAppName))
		assert.Equal(t, app.ObjectMeta.Namespace, envValue(appContainer.Env, resourcecreator.NaisNamespace))
		assert.Equal(t, app.Spec.Image, envValue(appContainer.Env, resourcecreator.NaisAppImage))
		assert.Equal(t, clusterName, envValue(appContainer.Env, resourcecreator.NaisClusterName))
	}))

	t.Run("certificate authority files and configuration is injected", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		opts := resourcecreator.NewResourceOptions()
		deploy, err := resourcecreator.Deployment(app, opts)
		assert.Nil(t, err)
		appContainer := getContainerByName(deploy.Spec.Template.Spec.Containers, app.Name)

		assert.Equal(t, resourcecreator.NAV_TRUSTSTORE_PATH, envValue(appContainer.Env, "NAV_TRUSTSTORE_PATH"))
		assert.Equal(t, resourcecreator.NAV_TRUSTSTORE_PASSWORD, envValue(appContainer.Env, "NAV_TRUSTSTORE_PASSWORD"))
		assert.Equal(t, resourcecreator.CA_BUNDLE_CONFIGMAP_NAME, appContainer.VolumeMounts[0].Name)
		assert.Equal(t, resourcecreator.CA_BUNDLE_CONFIGMAP_NAME, deploy.Spec.Template.Spec.Volumes[0].Name)
		assert.Equal(t, resourcecreator.CA_BUNDLE_CONFIGMAP_NAME, deploy.Spec.Template.Spec.Volumes[0].ConfigMap.Name)
	})

	t.Run("leader election sidecar is injected when LE is requested", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.LeaderElection = true
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		opts := resourcecreator.NewResourceOptions()
		deploy, err := resourcecreator.Deployment(app, opts)
		assert.Nil(t, err)

		appContainer := getContainerByName(deploy.Spec.Template.Spec.Containers, app.Name)
		assert.NotNil(t, appContainer)
		leaderElectionContainer := getContainerByName(deploy.Spec.Template.Spec.Containers, "elector")
		assert.NotNil(t, leaderElectionContainer)
	})

	t.Run("resource requests and limits are set up correctly", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Resources.Limits.Cpu = "150m"
		app.Spec.Resources.Limits.Memory = "2G"
		app.Spec.Resources.Requests.Cpu = "100m"
		app.Spec.Resources.Requests.Memory = "512M"
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		opts := resourcecreator.NewResourceOptions()
		deploy, err := resourcecreator.Deployment(app, opts)
		assert.Nil(t, err)

		appContainer := getContainerByName(deploy.Spec.Template.Spec.Containers, app.Name)
		assert.NotNil(t, appContainer)

		assert.Equal(t, app.Spec.Resources.Limits.Cpu, appContainer.Resources.Limits.Cpu().String())
		assert.Equal(t, app.Spec.Resources.Limits.Memory, appContainer.Resources.Limits.Memory().String())
		assert.Equal(t, app.Spec.Resources.Requests.Cpu, appContainer.Resources.Requests.Cpu().String())
		assert.Equal(t, app.Spec.Resources.Requests.Memory, appContainer.Resources.Requests.Memory().String())
	})

	t.Run("check if default port is used when liveness port is missing", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Liveness = nais.Probe{
			Path: "/probe/path",
		}
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		opts := resourcecreator.NewResourceOptions()
		deploy, err := resourcecreator.Deployment(app, opts)
		assert.Nil(t, err)

		appContainer := getContainerByName(deploy.Spec.Template.Spec.Containers, app.Name)
		assert.NotNil(t, appContainer)

		assert.Equal(t, nais.DefaultPortName, appContainer.LivenessProbe.HTTPGet.Port.StrVal)
	})

	t.Run("liveness configuration is set up correctly", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Liveness = nais.Probe{
			Path:             "/probe/path",
			Port:             12399,
			Timeout:          12,
			FailureThreshold: 33,
			InitialDelay:     44,
			PeriodSeconds:    55,
		}
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		opts := resourcecreator.NewResourceOptions()
		deploy, err := resourcecreator.Deployment(app, opts)
		assert.Nil(t, err)

		appContainer := getContainerByName(deploy.Spec.Template.Spec.Containers, app.Name)
		assert.NotNil(t, appContainer)

		assert.Equal(t, app.Spec.Liveness.Path, appContainer.LivenessProbe.HTTPGet.Path)
		assert.Equal(t, int32(app.Spec.Liveness.Port), appContainer.LivenessProbe.HTTPGet.Port.IntVal)
		assert.Equal(t, int32(app.Spec.Liveness.PeriodSeconds), appContainer.LivenessProbe.PeriodSeconds)
		assert.Equal(t, int32(app.Spec.Liveness.Timeout), appContainer.LivenessProbe.TimeoutSeconds)
		assert.Equal(t, int32(app.Spec.Liveness.FailureThreshold), appContainer.LivenessProbe.FailureThreshold)
		assert.Equal(t, int32(app.Spec.Liveness.InitialDelay), appContainer.LivenessProbe.InitialDelaySeconds)
	})

	t.Run("readiness configuration is set up correctly", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Readiness = nais.Probe{
			Path:             "/probe/path",
			Port:             12399,
			Timeout:          12,
			FailureThreshold: 33,
			InitialDelay:     44,
			PeriodSeconds:    55,
		}
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		opts := resourcecreator.NewResourceOptions()
		deploy, err := resourcecreator.Deployment(app, opts)
		assert.Nil(t, err)

		appContainer := getContainerByName(deploy.Spec.Template.Spec.Containers, app.Name)
		assert.NotNil(t, appContainer)

		assert.Equal(t, app.Spec.Readiness.Path, appContainer.ReadinessProbe.HTTPGet.Path)
		assert.Equal(t, int32(app.Spec.Readiness.Port), appContainer.ReadinessProbe.HTTPGet.Port.IntVal)
		assert.Equal(t, int32(app.Spec.Readiness.PeriodSeconds), appContainer.ReadinessProbe.PeriodSeconds)
		assert.Equal(t, int32(app.Spec.Readiness.Timeout), appContainer.ReadinessProbe.TimeoutSeconds)
		assert.Equal(t, int32(app.Spec.Readiness.FailureThreshold), appContainer.ReadinessProbe.FailureThreshold)
		assert.Equal(t, int32(app.Spec.Readiness.InitialDelay), appContainer.ReadinessProbe.InitialDelaySeconds)
	})

	t.Run("configMaps are mounted into the container", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.ConfigMaps.Files = []string{
			"foo",
			"bar",
		}
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		opts := resourcecreator.NewResourceOptions()
		deploy, err := resourcecreator.Deployment(app, opts)
		assert.Nil(t, err)
		appContainer := getContainerByName(deploy.Spec.Template.Spec.Containers, app.Name)

		for _, cm := range app.Spec.ConfigMaps.Files {
			name := fmt.Sprintf("nais-cm-%s", cm)
			volume := getVolumeByName(deploy.Spec.Template.Spec.Volumes, name)
			volumeMount := getVolumeMountByName(appContainer.VolumeMounts, name)

			assert.NotNil(t, volume)
			assert.NotNil(t, volumeMount)
			assert.NotNil(t, volume.ConfigMap)

			assert.Equal(t, volume.ConfigMap.LocalObjectReference.Name, cm)
			assert.Equal(t, volume.Name, volumeMount.Name)
			assert.Len(t, volume.ConfigMap.Items, 0)
		}
	})

	t.Run("webproxy configuration is injected into the container env", test.EnvWrapper(map[string]string{
		resourcecreator.PodHttpProxyEnv: httpProxy,
		resourcecreator.PodNoProxyEnv:   noProxy,
	}, func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.WebProxy = true
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		opts := resourcecreator.NewResourceOptions()
		deploy, err := resourcecreator.Deployment(app, opts)
		assert.Nil(t, err)
		appContainer := getContainerByName(deploy.Spec.Template.Spec.Containers, app.Name)

		assert.Equal(t, httpProxy, envValue(appContainer.Env, "HTTP_PROXY"))
		assert.Equal(t, httpProxy, envValue(appContainer.Env, "HTTPS_PROXY"))
		assert.Equal(t, noProxy, envValue(appContainer.Env, "NO_PROXY"))
		assert.Equal(t, httpProxy, envValue(appContainer.Env, "http_proxy"))
		assert.Equal(t, httpProxy, envValue(appContainer.Env, "https_proxy"))
		assert.Equal(t, noProxy, envValue(appContainer.Env, "no_proxy"))
	}))

	t.Run("probes are not configured when not set", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		deploy, err := resourcecreator.Deployment(app, resourcecreator.NewResourceOptions())

		assert.NoError(t, err)
		assert.Empty(t, deploy.Spec.Template.Spec.Containers[0].ReadinessProbe)
		assert.Empty(t, deploy.Spec.Template.Spec.Containers[0].LivenessProbe)
	})

	t.Run("default prestop hook applied when not provided", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		deploy, err := resourcecreator.Deployment(app, resourcecreator.NewResourceOptions())

		assert.NoError(t, err)
		assert.Empty(t, deploy.Spec.Template.Spec.Containers[0].Lifecycle.PreStop.HTTPGet)
		assert.Equal(t, []string{"sleep", "5"}, deploy.Spec.Template.Spec.Containers[0].Lifecycle.PreStop.Exec.Command)
	})

	t.Run("default deployment strategy is RollingUpdate", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)
		assert.Equal(t, appsv1.RollingUpdateDeploymentStrategyType, appsv1.DeploymentStrategyType(app.Spec.Strategy.Type))

		deploy, err := resourcecreator.Deployment(app, resourcecreator.NewResourceOptions())

		assert.NoError(t, err)
		assert.Equal(t, appsv1.RollingUpdateDeploymentStrategyType, appsv1.DeploymentStrategyType(deploy.Spec.Strategy.Type))
	})

	t.Run("when deploymentStrategy is set, it is used", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		app.Spec.Strategy.Type = nais.DeploymentStrategyRecreate
		deploy, err := resourcecreator.Deployment(app, resourcecreator.NewResourceOptions())

		assert.NoError(t, err)
		assert.Equal(t, appsv1.RecreateDeploymentStrategyType, appsv1.DeploymentStrategyType(deploy.Spec.Strategy.Type))
	})

	t.Run("ensure that secure logging sidecar is created when requesting secure logs in app spec", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.LeaderElection = false
		app.Spec.SecureLogs.Enabled = true
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		deployment, err := resourcecreator.Deployment(app, resourcecreator.ResourceOptions{})

		assert.NoError(t, err)
		assert.NotNil(t, deployment)

		spec := deployment.Spec.Template.Spec
		assert.Len(t, spec.Volumes, 3)
		assert.Len(t, spec.Containers, 3)
	})

	t.Run("environment variables are injected", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Env = []nais.EnvVar{
			{
				Name: "foo",
				Value: "bar",
			},
			{
				Name: "bar",
				Value: "baz",
			},
		}
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		opts := resourcecreator.NewResourceOptions()
		deploy, err := resourcecreator.Deployment(app, opts)
		assert.Nil(t, err)

		appContainer := getContainerByName(deploy.Spec.Template.Spec.Containers, app.Name)
		assert.NotNil(t, appContainer)

		assert.Equal(t, "bar", envValue(appContainer.Env, "foo"))
		assert.Equal(t, "baz", envValue(appContainer.Env, "bar"))
	})

	t.Run("environment variables uses valueFrom.fieldRef.fieldPath if set", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Env = append(app.Spec.Env, nais.EnvVar{
			Name: "podIP",
			ValueFrom: nais.EnvVarSource{
				FieldRef: nais.ObjectFieldSelector{FieldPath: "status.podIP"},
			},
		})
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		deployment, err := resourcecreator.Deployment(app, resourcecreator.ResourceOptions{})
		assert.NoError(t, err)
		assert.NotNil(t, deployment)

		appContainer := getContainerByName(deployment.Spec.Template.Spec.Containers, app.Name)

		for _, e := range appContainer.Env {
			if e.Name == "podIP" {
				assert.Equal(t, "status.podIP", e.ValueFrom.FieldRef.FieldPath)
				assert.Equal(t, "", e.Value)
			}
		}

	})
}

func getContainerByName(containers []v1.Container, name string) *v1.Container {
	for _, v := range containers {
		if v.Name == name {
			return &v
		}
	}

	return nil
}

func getVolumeByName(volumes []v1.Volume, name string) *v1.Volume {
	for _, v := range volumes {
		if v.Name == name {
			return &v
		}
	}

	return nil
}

func getVolumeMountByName(volumeMounts []v1.VolumeMount, name string) *v1.VolumeMount {
	for _, v := range volumeMounts {
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
