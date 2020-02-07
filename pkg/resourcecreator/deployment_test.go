package resourcecreator_test

import (
	"strings"
	"testing"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
)

const clusterName = "my.test.cluster.local"

func TestDeployment(t *testing.T) {

	t.Run("vault integration is set up correctly", func(t *testing.T) {
		viper.Reset()
		viper.Set("features.vault", true)
		viper.Set("vault.address", "adr")
		viper.Set("vault.auth-path", "authpath")
		viper.Set("vault.kv-path", "/base/kv")
		viper.Set("vault.init-container-image", "image")

		app := fixtures.MinimalApplication()
		app.Spec.Vault.Enabled = true
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		opts := resourcecreator.NewResourceOptions()
		deploy, err := resourcecreator.Deployment(app, opts)
		assert.Nil(t, err)

		c := resourcecreator.GetContainerByName(deploy.Spec.Template.Spec.InitContainers, "vks-init")
		assert.NotNil(t, c, "contains vault initcontainer")

		appContainer := resourcecreator.GetContainerByName(deploy.Spec.Template.Spec.Containers, app.Name)
		assert.NotNil(t, appContainer)
	})

	viper.Reset()
	viper.Set("cluster-name", clusterName)

	t.Run("check if default port is used when liveness port is missing", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Port = 12333
		app.Spec.Liveness = &nais.Probe{
			Path: "/probe/path",
		}
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		opts := resourcecreator.NewResourceOptions()
		deploy, err := resourcecreator.Deployment(app, opts)
		assert.Nil(t, err)

		appContainer := resourcecreator.GetContainerByName(deploy.Spec.Template.Spec.Containers, app.Name)
		assert.NotNil(t, appContainer)

		assert.Equal(t, app.Spec.Port, appContainer.LivenessProbe.HTTPGet.Port.IntValue())
		assert.Nil(t, appContainer.ReadinessProbe)
	})

	t.Run("liveness configuration is set up correctly", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Liveness = &nais.Probe{
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

		appContainer := resourcecreator.GetContainerByName(deploy.Spec.Template.Spec.Containers, app.Name)
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
		app.Spec.Readiness = &nais.Probe{
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

		appContainer := resourcecreator.GetContainerByName(deploy.Spec.Template.Spec.Containers, app.Name)
		assert.NotNil(t, appContainer)

		assert.Equal(t, app.Spec.Readiness.Path, appContainer.ReadinessProbe.HTTPGet.Path)
		assert.Equal(t, int32(app.Spec.Readiness.Port), appContainer.ReadinessProbe.HTTPGet.Port.IntVal)
		assert.Equal(t, int32(app.Spec.Readiness.PeriodSeconds), appContainer.ReadinessProbe.PeriodSeconds)
		assert.Equal(t, int32(app.Spec.Readiness.Timeout), appContainer.ReadinessProbe.TimeoutSeconds)
		assert.Equal(t, int32(app.Spec.Readiness.FailureThreshold), appContainer.ReadinessProbe.FailureThreshold)
		assert.Equal(t, int32(app.Spec.Readiness.InitialDelay), appContainer.ReadinessProbe.InitialDelaySeconds)
	})

	t.Run("enabling webproxy in GCP is no-op", func(t *testing.T) {
		viper.Reset()
		viper.Set("proxy.address", httpProxy)
		viper.Set("proxy.exclude", noProxy)
		app := fixtures.MinimalApplication()
		app.Spec.WebProxy = true
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		opts := resourcecreator.NewResourceOptions()
		opts.GoogleProjectId = "googleprojectid"
		deploy, err := resourcecreator.Deployment(app, opts)
		assert.Nil(t, err)

		appContainer := resourcecreator.GetContainerByName(deploy.Spec.Template.Spec.Containers, app.Name)

		assert.Zero(t, envValue(appContainer.Env, "HTTP_PROXY"))
		assert.Zero(t, envValue(appContainer.Env, "HTTPS_PROXY"))
		assert.Zero(t, envValue(appContainer.Env, "NO_PROXY"))
		assert.Zero(t, envValue(appContainer.Env, "http_proxy"))
		assert.Zero(t, envValue(appContainer.Env, "https_proxy"))
		assert.Zero(t, envValue(appContainer.Env, "no_proxy"))
	})

	t.Run("webproxy configuration is injected into the container env", func(t *testing.T) {
		viper.Reset()
		viper.Set("proxy.address", httpProxy)
		viper.Set("proxy.exclude", noProxy)
		app := fixtures.MinimalApplication()
		app.Spec.WebProxy = true
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		opts := resourcecreator.NewResourceOptions()
		deploy, err := resourcecreator.Deployment(app, opts)
		assert.Nil(t, err)
		appContainer := resourcecreator.GetContainerByName(deploy.Spec.Template.Spec.Containers, app.Name)

		nprox := strings.Join(noProxy, ",")

		assert.Equal(t, httpProxy, envValue(appContainer.Env, "HTTP_PROXY"))
		assert.Equal(t, httpProxy, envValue(appContainer.Env, "HTTPS_PROXY"))
		assert.Equal(t, nprox, envValue(appContainer.Env, "NO_PROXY"))
		assert.Equal(t, httpProxy, envValue(appContainer.Env, "http_proxy"))
		assert.Equal(t, httpProxy, envValue(appContainer.Env, "https_proxy"))
		assert.Equal(t, nprox, envValue(appContainer.Env, "no_proxy"))
	})

	t.Run("when deploymentStrategy is set, it is used", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		app.Spec.Strategy.Type = nais.DeploymentStrategyRecreate
		deploy, err := resourcecreator.Deployment(app, resourcecreator.NewResourceOptions())

		assert.NoError(t, err)
		assert.Equal(t, appsv1.RecreateDeploymentStrategyType, deploy.Spec.Strategy.Type)
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
		assert.Len(t, spec.Volumes, 4)
		assert.Len(t, spec.Containers, 3)
	})

	t.Run("environment variables are injected", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Env = []nais.EnvVar{
			{
				Name:  "foo",
				Value: "bar",
			},
			{
				Name:  "bar",
				Value: "baz",
			},
		}
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		opts := resourcecreator.NewResourceOptions()
		deploy, err := resourcecreator.Deployment(app, opts)
		assert.Nil(t, err)

		appContainer := resourcecreator.GetContainerByName(deploy.Spec.Template.Spec.Containers, app.Name)
		assert.NotNil(t, appContainer)

		assert.Equal(t, "bar", envValue(appContainer.Env, "foo"))
		assert.Equal(t, "baz", envValue(appContainer.Env, "bar"))
	})

	t.Run("environment variables uses valueFrom.fieldRef.fieldPath if set", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Env = append(app.Spec.Env, nais.EnvVar{
			Name: "podIP",
			ValueFrom: &nais.EnvVarSource{
				FieldRef: nais.ObjectFieldSelector{FieldPath: "status.podIP"},
			},
		})
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		deployment, err := resourcecreator.Deployment(app, resourcecreator.ResourceOptions{})
		assert.NoError(t, err)
		assert.NotNil(t, deployment)

		appContainer := resourcecreator.GetContainerByName(deployment.Spec.Template.Spec.Containers, app.Name)

		for _, e := range appContainer.Env {
			if e.Name == "podIP" {
				assert.Equal(t, "status.podIP", e.ValueFrom.FieldRef.FieldPath)
				assert.Equal(t, "", e.Value)
			}
		}
	})

	t.Run("secret defaults is applied", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		customMountPath := "hello/world"
		app.Spec.FilesFrom = []nais.FilesFrom{
			{Secret: "foo"},
			{Secret: "bar", MountPath: customMountPath},
		}

		deployment, err := resourcecreator.Deployment(app, resourcecreator.ResourceOptions{NativeSecrets: true})
		assert.NoError(t, err)
		assert.NotNil(t, deployment)

		appContainer := resourcecreator.GetContainerByName(deployment.Spec.Template.Spec.Containers, app.Name)
		assert.NotNil(t, appContainer)
		assert.Equal(t, nais.DefaultSecretMountPath, getVolumeMountByName(appContainer.VolumeMounts, "foo").MountPath)
		assert.Equal(t, customMountPath, getVolumeMountByName(appContainer.VolumeMounts, "bar").MountPath)
	})

	t.Run("froms are correctly configured", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		envConfigmapName := "envconfigmap"
		envSecretName := "envsecret"
		fileConfigmapName := "fileconfigmap"
		fileSecretName := "filesecret"
		fileSecretMountPath := "/my/path"

		app.Spec.EnvFrom = []nais.EnvFrom{
			{Secret: envSecretName},
			{ConfigMap: envConfigmapName},
		}
		app.Spec.FilesFrom = []nais.FilesFrom{
			{Secret: fileSecretName, MountPath: fileSecretMountPath},
			{ConfigMap: fileConfigmapName},
		}

		deployment, err := resourcecreator.Deployment(app, resourcecreator.ResourceOptions{NativeSecrets: true})
		assert.NoError(t, err)
		assert.NotNil(t, deployment)

		appContainer := resourcecreator.GetContainerByName(deployment.Spec.Template.Spec.Containers, app.Name)

		assert.Equal(t, 2, len(appContainer.EnvFrom))
		assert.Equal(t, envSecretName, appContainer.EnvFrom[0].SecretRef.Name)
		assert.Equal(t, envConfigmapName, appContainer.EnvFrom[1].ConfigMapRef.Name)

		secretVolumeMount := getVolumeMountByName(appContainer.VolumeMounts, fileSecretName)
		secretVolume := getVolumeByName(deployment.Spec.Template.Spec.Volumes, fileSecretName)
		assert.Equal(t, fileSecretName, secretVolumeMount.Name)
		assert.Equal(t, fileSecretMountPath, secretVolumeMount.MountPath)
		assert.True(t, secretVolumeMount.ReadOnly)
		assert.Equal(t, fileSecretName, secretVolume.Name)
		assert.Equal(t, fileSecretName, secretVolume.Secret.SecretName)

		configmapVolumeMount := getVolumeMountByName(appContainer.VolumeMounts, fileConfigmapName)
		configmapVolume := getVolumeByName(deployment.Spec.Template.Spec.Volumes, fileConfigmapName)
		assert.Equal(t, fileConfigmapName, configmapVolumeMount.Name)
		assert.Equal(t, nais.GetDefaultMountPath(fileConfigmapName), configmapVolumeMount.MountPath)
		assert.True(t, configmapVolumeMount.ReadOnly)
		assert.Equal(t, fileConfigmapName, configmapVolume.Name)
		assert.Equal(t, fileConfigmapName, configmapVolume.ConfigMap.Name)
	})

	t.Run("secrets are not configured when feature flag for secrets is false", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		app.Spec.EnvFrom = []nais.EnvFrom{
			{Secret: "foo"},
		}
		app.Spec.FilesFrom = []nais.FilesFrom{
			{Secret: "bar"},
		}

		deployment, err := resourcecreator.Deployment(app, resourcecreator.ResourceOptions{NativeSecrets: false})
		assert.NoError(t, err)
		appContainer := resourcecreator.GetContainerByName(deployment.Spec.Template.Spec.Containers, app.Name)
		assert.NotNil(t, appContainer)
		assert.Equal(t, 0, len(appContainer.EnvFrom))
		volumeMount := getVolumeMountByName(appContainer.VolumeMounts, "bar")
		assert.Nil(t, volumeMount)
	})
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
