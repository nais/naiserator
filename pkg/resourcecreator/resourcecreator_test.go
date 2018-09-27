package resourcecreator_test

import (
	"testing"

	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
)

func TestCreateResourceSpecs(t *testing.T) {
	app := fixtures.Application()

	specs, e := resourcecreator.Create(app)
	assert.NoError(t, e)

	svc := test.NamedResource(specs, "Service").(*v1.Service)
	assert.Equal(t, nais.DefaultPort, int(svc.Spec.Ports[0].Port))

	deploy := test.NamedResource(specs, "Deployment").(*appsv1.Deployment)
	assert.Equal(t, fixtures.ImageName, deploy.Spec.Template.Spec.Containers[0].Image)
}
