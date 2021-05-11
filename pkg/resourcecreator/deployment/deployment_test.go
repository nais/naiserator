package deployment_test

import (
	"github.com/nais/naiserator/pkg/resourcecreator/deployment"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/test"

	"testing"

	"github.com/nais/naiserator/pkg/resourcecreator/resource"

	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
)

const (
	clusterName = "my.test.cluster.local"
)

func TestDeployment(t *testing.T) {
	ops := resource.Operations{}

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

		opts := resource.NewOptions()
		deploy, err := deployment.Create(app, opts, &ops)
		assert.Nil(t, err)

		c := pod.GetContainerByName(deploy.Spec.Template.Spec.InitContainers, "vks-init")
		assert.NotNil(t, c, "contains vault initcontainer")

		appContainer := pod.GetContainerByName(deploy.Spec.Template.Spec.Containers, app.Name)
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

		opts := resource.NewOptions()
		deploy, err := deployment.Create(app, opts, &ops)
		assert.Nil(t, err)

		appContainer := pod.GetContainerByName(deploy.Spec.Template.Spec.Containers, app.Name)
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

		opts := resource.NewOptions()
		deploy, err := deployment.Create(app, opts, &ops)
		assert.Nil(t, err)

		appContainer := pod.GetContainerByName(deploy.Spec.Template.Spec.Containers, app.Name)
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

		opts := resource.NewOptions()
		deploy, err := deployment.Create(app, opts, &ops)
		assert.Nil(t, err)

		appContainer := pod.GetContainerByName(deploy.Spec.Template.Spec.Containers, app.Name)
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
		viper.Set("proxy.address", "http://foo.bar:5224")
		viper.Set("proxy.exclude", []string{"foo", "bar", "baz"})
		app := fixtures.MinimalApplication()
		app.Spec.WebProxy = true
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		opts := resource.NewOptions()
		opts.GoogleProjectId = "googleprojectid"
		deploy, err := deployment.Create(app, opts, &ops)
		assert.Nil(t, err)

		appContainer := pod.GetContainerByName(deploy.Spec.Template.Spec.Containers, app.Name)

		assert.Zero(t, test.EnvValue(appContainer.Env, "HTTP_PROXY"))
		assert.Zero(t, test.EnvValue(appContainer.Env, "HTTPS_PROXY"))
		assert.Zero(t, test.EnvValue(appContainer.Env, "NO_PROXY"))
		assert.Zero(t, test.EnvValue(appContainer.Env, "http_proxy"))
		assert.Zero(t, test.EnvValue(appContainer.Env, "https_proxy"))
		assert.Zero(t, test.EnvValue(appContainer.Env, "no_proxy"))
	})

	t.Run("when deploymentStrategy is set, it is used", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		app.Spec.Strategy.Type = nais.DeploymentStrategyRecreate
		deploy, err := deployment.Create(app, resource.NewOptions(), &ops)

		assert.NoError(t, err)
		assert.Equal(t, appsv1.RecreateDeploymentStrategyType, deploy.Spec.Strategy.Type)
	})

	t.Run("ensure that secure logging sidecar is created when requesting secure logs in app spec", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.LeaderElection = false
		app.Spec.SecureLogs.Enabled = true
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		deployment, err := deployment.Create(app, resource.Options{}, &ops)

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

		opts := resource.NewOptions()
		deploy, err := deployment.Create(app, opts, &ops)
		assert.Nil(t, err)

		appContainer := pod.GetContainerByName(deploy.Spec.Template.Spec.Containers, app.Name)
		assert.NotNil(t, appContainer)

		assert.Equal(t, "bar", test.EnvValue(appContainer.Env, "foo"))
		assert.Equal(t, "baz", test.EnvValue(appContainer.Env, "bar"))
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

		deployment, err := deployment.Create(app, resource.Options{}, &ops)
		assert.NoError(t, err)
		assert.NotNil(t, deployment)

		appContainer := pod.GetContainerByName(deployment.Spec.Template.Spec.Containers, app.Name)

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

		deployment, err := deployment.Create(app, resource.Options{NativeSecrets: true}, &ops)
		assert.NoError(t, err)
		assert.NotNil(t, deployment)

		appContainer := pod.GetContainerByName(deployment.Spec.Template.Spec.Containers, app.Name)
		assert.NotNil(t, appContainer)
		assert.Equal(t, nais.DefaultSecretMountPath, test.GetVolumeMountByName(appContainer.VolumeMounts, "foo").MountPath)
		assert.Equal(t, customMountPath, test.GetVolumeMountByName(appContainer.VolumeMounts, "bar").MountPath)
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

		deployment, err := deployment.Create(app, resource.Options{NativeSecrets: true}, &ops)
		assert.NoError(t, err)
		assert.NotNil(t, deployment)

		appContainer := pod.GetContainerByName(deployment.Spec.Template.Spec.Containers, app.Name)

		assert.Equal(t, 2, len(appContainer.EnvFrom))
		assert.Equal(t, envSecretName, appContainer.EnvFrom[0].SecretRef.Name)
		assert.Equal(t, envConfigmapName, appContainer.EnvFrom[1].ConfigMapRef.Name)

		secretVolumeMount := test.GetVolumeMountByName(appContainer.VolumeMounts, fileSecretName)
		secretVolume := test.GetVolumeByName(deployment.Spec.Template.Spec.Volumes, fileSecretName)
		assert.Equal(t, fileSecretName, secretVolumeMount.Name)
		assert.Equal(t, fileSecretMountPath, secretVolumeMount.MountPath)
		assert.True(t, secretVolumeMount.ReadOnly)
		assert.Equal(t, fileSecretName, secretVolume.Name)
		assert.Equal(t, fileSecretName, secretVolume.Secret.SecretName)

		configmapVolumeMount := test.GetVolumeMountByName(appContainer.VolumeMounts, fileConfigmapName)
		configmapVolume := test.GetVolumeByName(deployment.Spec.Template.Spec.Volumes, fileConfigmapName)
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

		deployment, err := deployment.Create(app, resource.Options{NativeSecrets: false}, &ops)
		assert.NoError(t, err)
		appContainer := pod.GetContainerByName(deployment.Spec.Template.Spec.Containers, app.Name)
		assert.NotNil(t, appContainer)
		assert.Equal(t, 0, len(appContainer.EnvFrom))
		volumeMount := test.GetVolumeMountByName(appContainer.VolumeMounts, "bar")
		assert.Nil(t, volumeMount)
	})

	t.Run("jwker secrets are mounted when JwkerSecretName is set", func(t *testing.T) {
		const jwkerSecret = "myJwkerSecret"
		app := fixtures.MinimalApplication()
		app.Spec.TokenX.Enabled = true
		deployment, err := deployment.Create(app, resource.Options{JwkerSecretName: jwkerSecret}, &ops)
		assert.NoError(t, err)

		appContainer := pod.GetContainerByName(deployment.Spec.Template.Spec.Containers, app.Name)
		assert.NotNil(t, appContainer)

		volumeMount := test.GetVolumeMountByName(appContainer.VolumeMounts, jwkerSecret)
		assert.NotEmpty(t, volumeMount)
		assert.Equal(t, jwkerSecret, volumeMount.Name)
		assert.Equal(t, "/var/run/secrets/nais.io/jwker", volumeMount.MountPath)

		jwkerVolume := test.GetVolumeByName(deployment.Spec.Template.Spec.Volumes, jwkerSecret)
		assert.NotEmpty(t, jwkerVolume)
		assert.Equal(t, jwkerSecret, jwkerVolume.Name)
		assert.Equal(t, jwkerSecret, jwkerVolume.VolumeSource.Secret.SecretName)

		assert.Equal(t, 1, len(appContainer.EnvFrom))
		assert.Equal(t, jwkerSecret, appContainer.EnvFrom[0].SecretRef.Name)
	})

	t.Run("jwker secrets are mounted as files and not exposed as env-variables if MountSecretsAsFilesOnly is true", func(t *testing.T) {
		const jwkerSecret = "myJwkerSecret"
		app := fixtures.MinimalApplication()
		app.Spec.TokenX.Enabled = true
		app.Spec.TokenX.MountSecretsAsFilesOnly = true
		deployment, err := deployment.Create(app, resource.Options{JwkerSecretName: jwkerSecret}, &ops)
		assert.NoError(t, err)

		appContainer := pod.GetContainerByName(deployment.Spec.Template.Spec.Containers, app.Name)
		assert.NotNil(t, appContainer)

		volumeMount := test.GetVolumeMountByName(appContainer.VolumeMounts, jwkerSecret)
		assert.NotEmpty(t, volumeMount)
		assert.Equal(t, jwkerSecret, volumeMount.Name)
		assert.Equal(t, "/var/run/secrets/nais.io/jwker", volumeMount.MountPath)

		jwkerVolume := test.GetVolumeByName(deployment.Spec.Template.Spec.Volumes, jwkerSecret)
		assert.NotEmpty(t, jwkerVolume)
		assert.Equal(t, jwkerSecret, jwkerVolume.Name)
		assert.Equal(t, jwkerSecret, jwkerVolume.VolumeSource.Secret.SecretName)

		assert.Equal(t, 0, len(appContainer.EnvFrom))
	})

	t.Run("when no jwkerSecretName is given there should be no jwker volume mount", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		deployment, err := deployment.Create(app, resource.Options{}, &ops)
		assert.NoError(t, err)

		appContainer := pod.GetContainerByName(deployment.Spec.Template.Spec.Containers, app.Name)
		assert.NotNil(t, appContainer)
		assert.Len(t, appContainer.VolumeMounts, 6)
		for _, v := range appContainer.VolumeMounts {
			assert.NotEqual(t, "/var/run/secrets", v.MountPath)
		}
	})
}
