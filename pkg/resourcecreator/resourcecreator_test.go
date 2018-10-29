package resourcecreator_test

import (
	"testing"

	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	"k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
)

func TestCreateResourceSpecs(t *testing.T) {
	app := fixtures.Application()

	specs, err := resourcecreator.Create(app)

	assert.Nil(t, err)

	svc := test.NamedResource(specs, "Service").(*v1.Service)
	assert.Equal(t, nais.DefaultPort, int(svc.Spec.Ports[0].Port))

	deploy := test.NamedResource(specs, "Deployment").(*appsv1.Deployment)
	assert.Equal(t, fixtures.ImageName, deploy.Spec.Template.Spec.Containers[0].Image)

	sa := test.NamedResource(specs, "ServiceAccount").(*v1.ServiceAccount)
	assert.Equal(t, fixtures.Name, sa.Name)

	ingress := test.NamedResource(specs, "Ingress").(*extensionsv1beta1.Ingress)
	assert.Len(t, ingress.Spec.Rules, 3)

	hpa := test.NamedResource(specs, "HorizontalPodAutoscaler").(*autoscalingv1.HorizontalPodAutoscaler)
	assert.Equal(t, fixtures.Name, hpa.Spec.ScaleTargetRef.Name)
}
