package resourcecreator

import (
	"github.com/nais/naiserator/api/types/v1alpha1"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"strings"
	"testing"
)

func TestCreateResourceSpecs(t *testing.T) {
	app := &v1alpha1.Application{Spec: v1alpha1.ApplicationSpec{
		Port: 69,
	}}
	specs, e := CreateResourceSpecs(app)
	assert.NoError(t, e)

	svc := get(specs, "service").(*v1.Service)
	assert.Equal(t, int(svc.Spec.Ports[0].Port), 69)
}

func TestCreateServiceSpec(t *testing.T) {
	app := &v1alpha1.Application{Spec: v1alpha1.ApplicationSpec{
		Port: 69,
	}}

	svc := createServiceSpec(app)

	assert.Equal(t, 69, svc.Spec.Ports[0].Port)
}

func get(resources []runtime.Object, kind string) runtime.Object {
	for _, r := range resources {
		if strings.EqualFold(r.GetObjectKind().GroupVersionKind().Kind, kind) {
			return r
		}
	}
	panic("no matching resource kind found")
}
