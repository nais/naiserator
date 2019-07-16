package generator_test

import (
	"testing"

	"github.com/nais/naiserator/pkg/event"
	"github.com/nais/naiserator/pkg/event/generator"
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
			Hash:  "sha256:5c3c0bbb737db91024882667ad5acbe64230ddecaca1d019968d8df2c4adab35",
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
			Hash:  "sha256:5c3c0bbb737db91024882667ad5acbe64230ddecaca1d019968d8df2c4adab35",
		},
	},
}

func TestContainerImage(t *testing.T) {
	for _, test := range containerImageTests {
		container := generator.ContainerImage(test.name)
		assert.Equal(t, test.container, container)
	}
}
