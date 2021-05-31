package jwker_test

import (
	"testing"

	"github.com/nais/naiserator/pkg/resourcecreator/certificateauthority"
	"github.com/nais/naiserator/pkg/resourcecreator/deployment"
	"github.com/nais/naiserator/pkg/resourcecreator/jwker"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/test"

	"github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestJwker(t *testing.T) {
	resourceOptions := resource.Options{}
	resourceOptions.JwkerEnabled = true
	clusterName := "myCluster"
	resourceOptions.ClusterName = clusterName
	otherApplication := "a"
	otherNamespace := "othernamespace"
	otherCluster := "otherCluster"
	otherApplication2 := "b"
	otherNamespace2 := "othernamespace2"
	otherApplication3 := "c"

	fixture := func() *nais_io_v1alpha1.Application {
		app := fixtures.MinimalApplication()
		app.Spec.TokenX = &nais_io_v1alpha1.TokenX{
			Enabled: true,
		}
		return app
	}

	t.Run("jwker for app with no access policy", func(t *testing.T) {
		app := fixture()
		ops := resource.Operations{}
		dplt, _ := deployment.Create(app.CreateObjectMeta(), resourceOptions, &ops, app.Annotations, *app.Spec.Strategy, app.Spec.Image,
			app.Spec.PreStopHookPath, app.Spec.Logformat, app.Spec.Logtransform, app.Spec.Port, *app.Spec.Resources, app.Spec.Liveness, app.Spec.Readiness, app.Spec.Startup,
			app.Spec.FilesFrom, app.Spec.EnvFrom, app.Spec.Env, app.Spec.Prometheus)
		jwker.Create(app.CreateObjectMeta(), resourceOptions, dplt, &ops, *app.Spec.TokenX, app.Spec.AccessPolicy)
		jkr := ops[len(ops)-1].Resource.(*nais_io_v1.Jwker)
		assert.NotEmpty(t, jkr)
		assert.NotEmpty(t, jkr.Spec.SecretName)
		assert.Len(t, jkr.Spec.AccessPolicy.Inbound.Rules, 0)
		assert.Len(t, jkr.Spec.AccessPolicy.Outbound.Rules, 0)
	})

	t.Run("one inbound without cluster/namespace and no outbound", func(t *testing.T) {
		app := fixture()
		app.Spec.AccessPolicy.Inbound.Rules = []nais_io_v1.AccessPolicyRule{{otherApplication, "", ""}}
		ops := resource.Operations{}
		dplt, _ := deployment.Create(app.CreateObjectMeta(), resourceOptions, &ops, app.Annotations, *app.Spec.Strategy, app.Spec.Image,
			app.Spec.PreStopHookPath, app.Spec.Logformat, app.Spec.Logtransform, app.Spec.Port, *app.Spec.Resources, app.Spec.Liveness, app.Spec.Readiness, app.Spec.Startup,
			app.Spec.FilesFrom, app.Spec.EnvFrom, app.Spec.Env, app.Spec.Prometheus)
		jwker.Create(app.CreateObjectMeta(), resourceOptions, dplt, &ops, *app.Spec.TokenX, app.Spec.AccessPolicy)
		jkr := ops[len(ops)-1].Resource.(*nais_io_v1.Jwker)
		assert.Len(t, jkr.Spec.AccessPolicy.Inbound.Rules, 1)
		assert.NotEmpty(t, jkr.Spec.SecretName)
		assert.Equal(t, otherApplication, jkr.Spec.AccessPolicy.Inbound.Rules[0].Application)
		assert.Equal(t, fixtures.ApplicationNamespace, jkr.Spec.AccessPolicy.Inbound.Rules[0].Namespace)
		assert.Equal(t, clusterName, jkr.Spec.AccessPolicy.Inbound.Rules[0].Cluster)
		assert.Len(t, jkr.Spec.AccessPolicy.Outbound.Rules, 0)
	})

	t.Run("one inbound with cluster/namespace and no outbound", func(t *testing.T) {
		app := fixture()
		app.Spec.AccessPolicy.Inbound.Rules = []nais_io_v1.AccessPolicyRule{{otherApplication, otherNamespace, otherCluster}}
		ops := resource.Operations{}
		dplt, _ := deployment.Create(app.CreateObjectMeta(), resourceOptions, &ops, app.Annotations, *app.Spec.Strategy, app.Spec.Image,
			app.Spec.PreStopHookPath, app.Spec.Logformat, app.Spec.Logtransform, app.Spec.Port, *app.Spec.Resources, app.Spec.Liveness, app.Spec.Readiness, app.Spec.Startup,
			app.Spec.FilesFrom, app.Spec.EnvFrom, app.Spec.Env, app.Spec.Prometheus)
		jwker.Create(app.CreateObjectMeta(), resourceOptions, dplt, &ops, *app.Spec.TokenX, app.Spec.AccessPolicy)
		jkr := ops[len(ops)-1].Resource.(*nais_io_v1.Jwker)
		assert.Len(t, jkr.Spec.AccessPolicy.Inbound.Rules, 1)
		assert.NotEmpty(t, jkr.Spec.SecretName)
		assert.Equal(t, otherApplication, jkr.Spec.AccessPolicy.Inbound.Rules[0].Application)
		assert.Equal(t, otherNamespace, jkr.Spec.AccessPolicy.Inbound.Rules[0].Namespace)
		assert.Equal(t, otherCluster, jkr.Spec.AccessPolicy.Inbound.Rules[0].Cluster)
		assert.Len(t, jkr.Spec.AccessPolicy.Outbound.Rules, 0)
	})

	t.Run("one outbound and no inbound", func(t *testing.T) {
		app := fixture()
		app.Spec.AccessPolicy.Outbound.Rules = []nais_io_v1.AccessPolicyRule{{otherApplication, otherNamespace, otherCluster}}
		ops := resource.Operations{}
		dplt, _ := deployment.Create(app.CreateObjectMeta(), resourceOptions, &ops, app.Annotations, *app.Spec.Strategy, app.Spec.Image,
			app.Spec.PreStopHookPath, app.Spec.Logformat, app.Spec.Logtransform, app.Spec.Port, *app.Spec.Resources, app.Spec.Liveness, app.Spec.Readiness, app.Spec.Startup,
			app.Spec.FilesFrom, app.Spec.EnvFrom, app.Spec.Env, app.Spec.Prometheus)
		jwker.Create(app.CreateObjectMeta(), resourceOptions, dplt, &ops, *app.Spec.TokenX, app.Spec.AccessPolicy)
		jkr := ops[len(ops)-1].Resource.(*nais_io_v1.Jwker)
		assert.Len(t, jkr.Spec.AccessPolicy.Outbound.Rules, 1)
		assert.NotEmpty(t, jkr.Spec.SecretName)
		assert.Equal(t, otherApplication, jkr.Spec.AccessPolicy.Outbound.Rules[0].Application)
		assert.Equal(t, otherNamespace, jkr.Spec.AccessPolicy.Outbound.Rules[0].Namespace)
		assert.Equal(t, otherCluster, jkr.Spec.AccessPolicy.Outbound.Rules[0].Cluster)
		assert.Len(t, jkr.Spec.AccessPolicy.Inbound.Rules, 0)
	})

	t.Run("multiple inbound and no outbound", func(t *testing.T) {
		app := fixture()
		app.Spec.AccessPolicy.Inbound.Rules = []nais_io_v1.AccessPolicyRule{
			{
				otherApplication, otherNamespace, otherCluster,
			},
			{
				otherApplication2, otherNamespace2, "",
			},
			{
				otherApplication3, "", "",
			},
		}
		ops := resource.Operations{}
		dplt, _ := deployment.Create(app.CreateObjectMeta(), resourceOptions, &ops, app.Annotations, *app.Spec.Strategy, app.Spec.Image,
			app.Spec.PreStopHookPath, app.Spec.Logformat, app.Spec.Logtransform, app.Spec.Port, *app.Spec.Resources, app.Spec.Liveness, app.Spec.Readiness, app.Spec.Startup,
			app.Spec.FilesFrom, app.Spec.EnvFrom, app.Spec.Env, app.Spec.Prometheus)
		jwker.Create(app.CreateObjectMeta(), resourceOptions, dplt, &ops, *app.Spec.TokenX, app.Spec.AccessPolicy)
		jkr := ops[len(ops)-1].Resource.(*nais_io_v1.Jwker)
		assert.Len(t, jkr.Spec.AccessPolicy.Inbound.Rules, 3)
		assert.Len(t, jkr.Spec.AccessPolicy.Outbound.Rules, 0)
		assert.NotEmpty(t, jkr.Spec.SecretName)
		assert.Equal(t, otherApplication, jkr.Spec.AccessPolicy.Inbound.Rules[0].Application)
		assert.Equal(t, otherNamespace, jkr.Spec.AccessPolicy.Inbound.Rules[0].Namespace)
		assert.Equal(t, otherCluster, jkr.Spec.AccessPolicy.Inbound.Rules[0].Cluster)
		assert.Equal(t, otherApplication2, jkr.Spec.AccessPolicy.Inbound.Rules[1].Application)
		assert.Equal(t, otherNamespace2, jkr.Spec.AccessPolicy.Inbound.Rules[1].Namespace)
		assert.Equal(t, clusterName, jkr.Spec.AccessPolicy.Inbound.Rules[1].Cluster)
		assert.Equal(t, otherApplication3, jkr.Spec.AccessPolicy.Inbound.Rules[2].Application)
		assert.Equal(t, fixtures.ApplicationNamespace, jkr.Spec.AccessPolicy.Inbound.Rules[2].Namespace)
		assert.Equal(t, clusterName, jkr.Spec.AccessPolicy.Inbound.Rules[2].Cluster)
	})

	t.Run("multiple outbound and no inbound", func(t *testing.T) {
		app := fixture()
		app.Spec.AccessPolicy.Outbound.Rules = []nais_io_v1.AccessPolicyRule{
			{
				otherApplication, otherNamespace, otherCluster,
			},
			{
				otherApplication2, otherNamespace2, "",
			},
			{
				otherApplication3, "", "",
			},
		}
		ops := resource.Operations{}
		dplt, _ := deployment.Create(app.CreateObjectMeta(), resourceOptions, &ops, app.Annotations, *app.Spec.Strategy, app.Spec.Image,
			app.Spec.PreStopHookPath, app.Spec.Logformat, app.Spec.Logtransform, app.Spec.Port, *app.Spec.Resources, app.Spec.Liveness, app.Spec.Readiness, app.Spec.Startup,
			app.Spec.FilesFrom, app.Spec.EnvFrom, app.Spec.Env, app.Spec.Prometheus)
		jwker.Create(app.CreateObjectMeta(), resourceOptions, dplt, &ops, *app.Spec.TokenX, app.Spec.AccessPolicy)
		jkr := ops[len(ops)-1].Resource.(*nais_io_v1.Jwker)
		assert.Len(t, jkr.Spec.AccessPolicy.Outbound.Rules, 3)
		assert.Len(t, jkr.Spec.AccessPolicy.Inbound.Rules, 0)
		assert.NotEmpty(t, jkr.Spec.SecretName)
		assert.Equal(t, otherApplication, jkr.Spec.AccessPolicy.Outbound.Rules[0].Application)
		assert.Equal(t, otherNamespace, jkr.Spec.AccessPolicy.Outbound.Rules[0].Namespace)
		assert.Equal(t, otherCluster, jkr.Spec.AccessPolicy.Outbound.Rules[0].Cluster)
		assert.Equal(t, otherApplication2, jkr.Spec.AccessPolicy.Outbound.Rules[1].Application)
		assert.Equal(t, otherNamespace2, jkr.Spec.AccessPolicy.Outbound.Rules[1].Namespace)
		assert.Equal(t, clusterName, jkr.Spec.AccessPolicy.Outbound.Rules[1].Cluster)
		assert.Equal(t, otherApplication3, jkr.Spec.AccessPolicy.Outbound.Rules[2].Application)
		assert.Equal(t, fixtures.ApplicationNamespace, jkr.Spec.AccessPolicy.Outbound.Rules[2].Namespace)
		assert.Equal(t, clusterName, jkr.Spec.AccessPolicy.Outbound.Rules[2].Cluster)
	})
	//
	t.Run("multiple inbound and multiple outbound", func(t *testing.T) {
		app := fixture()
		app.Spec.AccessPolicy.Inbound.Rules = []nais_io_v1.AccessPolicyRule{
			{
				otherApplication, otherNamespace, otherCluster,
			},
			{
				otherApplication2, otherNamespace2, "",
			},
			{
				otherApplication3, "", "",
			},
		}

		app.Spec.AccessPolicy.Outbound.Rules = []nais_io_v1.AccessPolicyRule{
			{
				otherApplication, otherNamespace, otherCluster,
			},
			{
				otherApplication2, otherNamespace2, "",
			},
			{
				otherApplication3, "", "",
			},
		}
		ops := resource.Operations{}
		dplt, _ := deployment.Create(app.CreateObjectMeta(), resourceOptions, &ops, app.Annotations, *app.Spec.Strategy, app.Spec.Image,
			app.Spec.PreStopHookPath, app.Spec.Logformat, app.Spec.Logtransform, app.Spec.Port, *app.Spec.Resources, app.Spec.Liveness, app.Spec.Readiness, app.Spec.Startup,
			app.Spec.FilesFrom, app.Spec.EnvFrom, app.Spec.Env, app.Spec.Prometheus)
		jwker.Create(app.CreateObjectMeta(), resourceOptions, dplt, &ops, *app.Spec.TokenX, app.Spec.AccessPolicy)
		jkr := ops[len(ops)-1].Resource.(*nais_io_v1.Jwker)
		assert.Len(t, jkr.Spec.AccessPolicy.Inbound.Rules, 3)
		assert.Len(t, jkr.Spec.AccessPolicy.Outbound.Rules, 3)
		assert.NotEmpty(t, jkr.Spec.SecretName)
		assert.Equal(t, otherApplication, jkr.Spec.AccessPolicy.Inbound.Rules[0].Application)
		assert.Equal(t, otherNamespace, jkr.Spec.AccessPolicy.Inbound.Rules[0].Namespace)
		assert.Equal(t, otherCluster, jkr.Spec.AccessPolicy.Inbound.Rules[0].Cluster)
		assert.Equal(t, otherApplication2, jkr.Spec.AccessPolicy.Inbound.Rules[1].Application)
		assert.Equal(t, otherNamespace2, jkr.Spec.AccessPolicy.Inbound.Rules[1].Namespace)
		assert.Equal(t, clusterName, jkr.Spec.AccessPolicy.Inbound.Rules[1].Cluster)
		assert.Equal(t, otherApplication3, jkr.Spec.AccessPolicy.Inbound.Rules[2].Application)
		assert.Equal(t, fixtures.ApplicationNamespace, jkr.Spec.AccessPolicy.Inbound.Rules[2].Namespace)
		assert.Equal(t, clusterName, jkr.Spec.AccessPolicy.Inbound.Rules[2].Cluster)
		assert.Equal(t, otherApplication, jkr.Spec.AccessPolicy.Outbound.Rules[0].Application)
		assert.Equal(t, otherNamespace, jkr.Spec.AccessPolicy.Outbound.Rules[0].Namespace)
		assert.Equal(t, otherCluster, jkr.Spec.AccessPolicy.Outbound.Rules[0].Cluster)
		assert.Equal(t, otherApplication2, jkr.Spec.AccessPolicy.Outbound.Rules[1].Application)
		assert.Equal(t, otherNamespace2, jkr.Spec.AccessPolicy.Outbound.Rules[1].Namespace)
		assert.Equal(t, clusterName, jkr.Spec.AccessPolicy.Outbound.Rules[1].Cluster)
		assert.Equal(t, otherApplication3, jkr.Spec.AccessPolicy.Outbound.Rules[2].Application)
		assert.Equal(t, fixtures.ApplicationNamespace, jkr.Spec.AccessPolicy.Outbound.Rules[2].Namespace)
		assert.Equal(t, clusterName, jkr.Spec.AccessPolicy.Outbound.Rules[2].Cluster)
	})


	t.Run("jwker secrets are mounted when JwkerSecretName is set", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		ops := resource.Operations{}
		app.Spec.TokenX.Enabled = true
		dplt, err := deployment.Create(app.CreateObjectMeta(), resourceOptions, &ops, app.Annotations, *app.Spec.Strategy, app.Spec.Image,
			app.Spec.PreStopHookPath, app.Spec.Logformat, app.Spec.Logtransform, app.Spec.Port, *app.Spec.Resources, app.Spec.Liveness, app.Spec.Readiness, app.Spec.Startup,
			app.Spec.FilesFrom, app.Spec.EnvFrom, app.Spec.Env, app.Spec.Prometheus)
		jwker.Create(app.CreateObjectMeta(), resourceOptions, dplt, &ops, *app.Spec.TokenX, app.Spec.AccessPolicy)
		jwkerSecretName := ops[len(ops)-1].Resource.(*nais_io_v1.Jwker).Spec.SecretName
		assert.NoError(t, err)

		appContainer := test.GetContainerByName(dplt.Spec.Template.Spec.Containers, app.Name)
		assert.NotNil(t, appContainer)

		volumeMount := test.GetVolumeMountByName(appContainer.VolumeMounts, jwkerSecretName)
		assert.NotEmpty(t, volumeMount)
		assert.Equal(t, jwkerSecretName, volumeMount.Name)
		assert.Equal(t, "/var/run/secrets/nais.io/jwker", volumeMount.MountPath)

		jwkerVolume := test.GetVolumeByName(dplt.Spec.Template.Spec.Volumes, jwkerSecretName)
		assert.NotEmpty(t, jwkerVolume)
		assert.Equal(t, jwkerSecretName, jwkerVolume.Name)
		assert.Equal(t, jwkerSecretName, jwkerVolume.VolumeSource.Secret.SecretName)

		assert.Equal(t, 1, len(appContainer.EnvFrom))
		assert.Equal(t, jwkerSecretName, appContainer.EnvFrom[0].SecretRef.Name)
	})

	t.Run("jwker secrets are mounted as files and not exposed as env-variables if MountSecretsAsFilesOnly is true", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		ops := resource.Operations{}
		app.Spec.TokenX.Enabled = true
		app.Spec.TokenX.MountSecretsAsFilesOnly = true
		dplt, err := deployment.Create(app.CreateObjectMeta(), resourceOptions, &ops, app.Annotations, *app.Spec.Strategy, app.Spec.Image,
			app.Spec.PreStopHookPath, app.Spec.Logformat, app.Spec.Logtransform, app.Spec.Port, *app.Spec.Resources, app.Spec.Liveness, app.Spec.Readiness, app.Spec.Startup,
			app.Spec.FilesFrom, app.Spec.EnvFrom, app.Spec.Env, app.Spec.Prometheus)
		jwker.Create(app.CreateObjectMeta(), resourceOptions, dplt, &ops, *app.Spec.TokenX, app.Spec.AccessPolicy)
		jwkerSecretName := ops[len(ops)-1].Resource.(*nais_io_v1.Jwker).Spec.SecretName
		assert.NoError(t, err)

		appContainer := test.GetContainerByName(dplt.Spec.Template.Spec.Containers, app.Name)
		assert.NotNil(t, appContainer)

		volumeMount := test.GetVolumeMountByName(appContainer.VolumeMounts, jwkerSecretName)
		assert.NotEmpty(t, volumeMount)
		assert.Equal(t, jwkerSecretName, volumeMount.Name)
		assert.Equal(t, "/var/run/secrets/nais.io/jwker", volumeMount.MountPath)

		jwkerVolume := test.GetVolumeByName(dplt.Spec.Template.Spec.Volumes, jwkerSecretName)
		assert.NotEmpty(t, jwkerVolume)
		assert.Equal(t, jwkerSecretName, jwkerVolume.Name)
		assert.Equal(t, jwkerSecretName, jwkerVolume.VolumeSource.Secret.SecretName)

		assert.Equal(t, 0, len(appContainer.EnvFrom))
	})

	t.Run("when no secretName is given there should be no jwker volume mount", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		options := resource.Options{}
		ops := resource.Operations{}
		dplt, err := deployment.Create(app.CreateObjectMeta(), options, &ops, app.Annotations, *app.Spec.Strategy, app.Spec.Image,
			app.Spec.PreStopHookPath, app.Spec.Logformat, app.Spec.Logtransform, app.Spec.Port, *app.Spec.Resources, app.Spec.Liveness, app.Spec.Readiness, app.Spec.Startup,
			app.Spec.FilesFrom, app.Spec.EnvFrom, app.Spec.Env, app.Spec.Prometheus)
		jwker.Create(app.CreateObjectMeta(), resourceOptions, dplt, &ops, *app.Spec.TokenX, app.Spec.AccessPolicy)
		certificateauthority.Create(dplt, app.Spec.SkipCaBundle)
		assert.NoError(t, err)

		appContainer := test.GetContainerByName(dplt.Spec.Template.Spec.Containers, app.Name)
		assert.NotNil(t, appContainer)
		assert.Len(t, appContainer.VolumeMounts, 6)
		for _, v := range appContainer.VolumeMounts {
			assert.NotEqual(t, "/var/run/secrets", v.MountPath)
		}
	})
}
