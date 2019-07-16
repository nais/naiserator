package generator_test

import (
	"testing"

	"github.com/nais/naiserator/pkg/event"
	"github.com/nais/naiserator/pkg/event/generator"
	"github.com/nais/naiserator/pkg/test/fixtures"
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
	app := fixtures.MinimalApplication()
	app.Spec.Image = "image:version"

	event := generator.NewDeploymentEvent(*app)

	assert.Equal(t, deployment.PlatformType_nais, event.GetPlatform().GetType())
	assert.Empty(t, event.GetPlatform().GetVariant())

	assert.Equal(t, deployment.System_naiserator, event.GetSource())

	assert.Nil(t, event.GetDeployer())

	assert.Equal(t, fixtures.ApplicationTeam, event.GetTeam())

	assert.Equal(t, deployment.RolloutStatus_initialized, event.GetRolloutStatus())

	assert.Equal(t, deployment.Environment_development, event.GetEnvironment())

	assert.Equal(t, fixtures.ApplicationNamespace, event.GetNamespace())

	assert.Empty(t, event.GetCluster())

	assert.Equal(t, fixtures.ApplicationName, event.GetApplication())

	assert.Equal(t, "version", event.GetVersion())

	image := event.GetImage()
	assert.NotNil(t, image)
	assert.Equal(t, deployment.ContainerImage{
		Name: "docker.io/library/image",
		Tag:  "version",
	}, *image)

	assert.True(t, event.GetTimestamp() > 0)
}
