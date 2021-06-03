package generator_test

import (
	"testing"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	deployment "github.com/nais/naiserator/pkg/event"
	"github.com/nais/naiserator/pkg/event/generator"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

type containerImageTest struct {
	name      string
	container deployment.ContainerImage
}

var containerImageTests = []containerImageTest{
	{
		name: "nginx",
		container: deployment.ContainerImage{
			Name: "docker.io/library/nginx",
			Tag:  "latest",
		},
	},
	{
		name: "nginx:latest",
		container: deployment.ContainerImage{
			Name: "docker.io/library/nginx",
			Tag:  "latest",
		},
	},
	{
		name: "nginx:tagged",
		container: deployment.ContainerImage{
			Name: "docker.io/library/nginx",
			Tag:  "tagged",
		},
	},
	{
		name: "organization/repo:0.1.2",
		container: deployment.ContainerImage{
			Name: "docker.io/organization/repo",
			Tag:  "0.1.2",
		},
	},
	{
		name: "nginx@sha256:5c3c0bbb737db91024882667ad5acbe64230ddecaca1d019968d8df2c4adab35",
		container: deployment.ContainerImage{
			Name: "docker.io/library/nginx",
			Hash: "sha256:5c3c0bbb737db91024882667ad5acbe64230ddecaca1d019968d8df2c4adab35",
		},
	},
	{
		name: "internal.repo:12345/foo/bar/image",
		container: deployment.ContainerImage{
			Name: "internal.repo:12345/foo/bar/image",
			Tag:  "latest",
		},
	},
	{
		name: "internal.repo:12345/foo/bar/image:tagged",
		container: deployment.ContainerImage{
			Name: "internal.repo:12345/foo/bar/image",
			Tag:  "tagged",
		},
	},
	{
		name: "internal.repo:12345/foo/bar/image@sha256:5c3c0bbb737db91024882667ad5acbe64230ddecaca1d019968d8df2c4adab35",
		container: deployment.ContainerImage{
			Name: "internal.repo:12345/foo/bar/image",
			Hash: "sha256:5c3c0bbb737db91024882667ad5acbe64230ddecaca1d019968d8df2c4adab35",
		},
	},
}

func TestContainerImage(t *testing.T) {
	for _, test := range containerImageTests {
		container := generator.ContainerImage(test.name)
		assert.Equal(t, test.container, container)
	}
}

func TestNewDeploymentEvent(t *testing.T) {
	t.Run("Event defaults are picked up from Application correctly", func(t *testing.T) {
		clusterName := "test-cluster"
		viper.Set("cluster-name", clusterName)

		app := fixtures.MinimalApplication()
		app.Spec.Image = "image:version"

		event := generator.NewDeploymentEvent(app, app.Spec.Image)

		assert.Equal(t, deployment.PlatformType_nais, event.GetPlatform().GetType())
		assert.Empty(t, event.GetPlatform().GetVariant())
		assert.Equal(t, deployment.System_naiserator, event.GetSource())
		assert.Nil(t, event.GetDeployer())
		assert.Equal(t, fixtures.ApplicationTeam, event.GetTeam())
		assert.Equal(t, deployment.RolloutStatus_initialized, event.GetRolloutStatus())
		assert.Equal(t, deployment.Environment_development, event.GetEnvironment())
		assert.Equal(t, fixtures.ApplicationNamespace, event.GetNamespace())
		assert.Equal(t, clusterName, event.GetCluster())
		assert.Equal(t, fixtures.ApplicationName, event.GetApplication())
		assert.Equal(t, "version", event.GetVersion())

		image := event.GetImage()
		assert.NotEmpty(t, image)
		assert.Equal(t, deployment.ContainerImage{
			Name: "docker.io/library/image",
			Tag:  "version",
		}, *image)

		assert.True(t, event.GetTimestampAsTime().Unix() > 0)
		assert.True(t, event.GetTimestampAsTime().UnixNano() > 0)
	})

	t.Run("Prod cluster Environment", func(t *testing.T) {
		clusterName := "prod-cluster"
		viper.Set("cluster-name", clusterName)

		app := fixtures.MinimalApplication()

		event := generator.NewDeploymentEvent(app, app.Spec.Image)

		assert.Equal(t, deployment.Environment_production, event.GetEnvironment())
	})

	t.Run("Get correlationID from app annotations", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.ObjectMeta = resource.CreateObjectMeta(app)

		correlationID := "correlation-id"
		app.Annotations[nais_io_v1.DeploymentCorrelationIDAnnotation] = correlationID

		event := generator.NewDeploymentEvent(app, app.Spec.Image)

		assert.Equal(t, correlationID, event.CorrelationID)
	})

	t.Run("Get team-name from app labels", func(t *testing.T) {
		app := fixtures.MinimalApplication()

		team := "team"
		app.Labels["team"] = team

		event := generator.NewDeploymentEvent(app, app.Spec.Image)

		assert.Equal(t, team, event.Team)
	})

}
