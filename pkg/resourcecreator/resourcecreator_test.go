package resourcecreator

import (
	nais "github.com/nais/naiserator/api/types/v1alpha1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"strings"
	"testing"
)

const (
	ImageName = "user/image:version"
)

func TestCreateResourceSpecs(t *testing.T) {
	app := &nais.Application{Spec: nais.ApplicationSpec{
		Port:  nais.DefaultPort,
		Image: ImageName,
	}}

	if err := nais.ApplyDefaults(app); err != nil {
		panic(err)
	}

	specs, e := GetResources(app)
	assert.NoError(t, e)

	svc := get(specs, "service").(*v1.Service)
	assert.Equal(t, nais.DefaultPort, int(svc.Spec.Ports[0].Port))

	deploy := get(specs, "deployment").(*appsv1.Deployment)
	assert.Equal(t, ImageName, deploy.Spec.Template.Spec.Containers[0].Image)
}

func get(resources []runtime.Object, kind string) runtime.Object {
	for _, r := range resources {
		if strings.EqualFold(r.GetObjectKind().GroupVersionKind().Kind, kind) {
			return r
		}
	}
	panic("no matching resource kind found")
}
