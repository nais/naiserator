package deployment_test

import (
	"testing"

	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator/deployment"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/test"

	"github.com/nais/naiserator/pkg/resourcecreator/resource"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
)

func TestDeployment(t *testing.T) {
	t.Run("check if default port is used when liveness port is missing", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Port = 12333
		app.Spec.Liveness = &nais_io_v1.Probe{
			Path: "/probe/path",
		}
		err := app.ApplyDefaults()
		assert.NoError(t, err)

		opts := resource.NewOptions()
		ast := resource.NewAst()
		pod.CreateAppContainer(app, ast, opts)

		appContainer := ast.Containers[0]
		assert.NotNil(t, appContainer)

		assert.Equal(t, app.Spec.Port, appContainer.LivenessProbe.HTTPGet.Port.IntValue())
		assert.Nil(t, appContainer.ReadinessProbe)
	})

	t.Run("enabling webproxy in GCP is no-op", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.WebProxy = true
		err := app.ApplyDefaults()
		assert.NoError(t, err)

		opts := resource.NewOptions()
		opts.GoogleProjectId = "googleprojectid"
		opts.Proxy = config.Proxy{
			Address: "httpProxy",
			Exclude: []string{"foo", "bar", "baz"},
		}
		ast := resource.NewAst()
		pod.CreateAppContainer(app, ast, opts)

		appContainer := ast.Containers[0]

		assert.Zero(t, test.EnvValue(appContainer.Env, "HTTP_PROXY"))
		assert.Zero(t, test.EnvValue(appContainer.Env, "HTTPS_PROXY"))
		assert.Zero(t, test.EnvValue(appContainer.Env, "NO_PROXY"))
		assert.Zero(t, test.EnvValue(appContainer.Env, "http_proxy"))
		assert.Zero(t, test.EnvValue(appContainer.Env, "https_proxy"))
		assert.Zero(t, test.EnvValue(appContainer.Env, "no_proxy"))
	})

	t.Run("when deploymentStrategy is set, it is used", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := app.ApplyDefaults()
		assert.NoError(t, err)

		app.Spec.Strategy.Type = nais_io_v1alpha1.DeploymentStrategyRecreate
		opts := resource.NewOptions()
		ast := resource.NewAst()
		err = deployment.Create(app, ast, opts)
		assert.Nil(t, err)

		deploy := ast.Operations[0].Resource.(*appsv1.Deployment)
		assert.Equal(t, appsv1.RecreateDeploymentStrategyType, deploy.Spec.Strategy.Type)
	})

	t.Run("secret defaults are applied", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := app.ApplyDefaults()
		assert.NoError(t, err)

		customMountPath := "hello/world"
		app.Spec.FilesFrom = []nais_io_v1.FilesFrom{
			{Secret: "foo"},
			{Secret: "bar", MountPath: customMountPath},
		}

		ast := resource.NewAst()
		opts := resource.NewOptions()
		opts.NativeSecrets = true

		pod.CreateAppContainer(app, ast, opts)

		appContainer := ast.Containers[0]
		assert.NotNil(t, appContainer)
		assert.Equal(t, nais_io_v1alpha1.DefaultSecretMountPath, test.GetVolumeMountByName(appContainer.VolumeMounts, "foo").MountPath)
		assert.Equal(t, customMountPath, test.GetVolumeMountByName(appContainer.VolumeMounts, "bar").MountPath)
	})

	t.Run("secrets are not configured when feature flag for secrets is false", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		err := app.ApplyDefaults()
		assert.NoError(t, err)

		app.Spec.EnvFrom = []nais_io_v1.EnvFrom{
			{Secret: "foo"},
		}
		app.Spec.FilesFrom = []nais_io_v1.FilesFrom{
			{Secret: "bar"},
		}

		ast := resource.NewAst()
		opts := resource.NewOptions()

		pod.CreateAppContainer(app, ast, opts)
		appContainer := ast.Containers[0]

		assert.NotNil(t, appContainer)
		assert.Equal(t, 0, len(appContainer.EnvFrom))
		volumeMount := test.GetVolumeMountByName(appContainer.VolumeMounts, "bar")
		assert.Nil(t, volumeMount)
	})
}
