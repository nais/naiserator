package resourcecreator_test

import (
	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"k8s.io/apimachinery/pkg/util/intstr"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetIngress(t *testing.T) {
	app := fixtures.Application()
	ingress := resourcecreator.Ingress(app)

	assert.Equal(t, app.Name, ingress.Name)
	assert.Equal(t, app.Namespace, ingress.Namespace)
	assert.Equal(t, "app.nais.adeo.no", ingress.Spec.Rules[0].Host)
	assert.Equal(t, "/", ingress.Spec.Rules[0].HTTP.Paths[0].Path)
	assert.Equal(t, app.Name, ingress.Spec.Rules[0].HTTP.Paths[0].Backend.ServiceName)
	assert.Equal(t, intstr.IntOrString{IntVal: nais.DefaultPort}, ingress.Spec.Rules[0].HTTP.Paths[0].Backend.ServicePort)

	assert.Equal(t, app.Name, ingress.Name)
	assert.Equal(t, app.Namespace, ingress.Namespace)
	assert.Equal(t, "tjenester.nav.no", ingress.Spec.Rules[1].Host)
	assert.Equal(t, "/app", ingress.Spec.Rules[1].HTTP.Paths[0].Path)
	assert.Equal(t, app.Name, ingress.Spec.Rules[1].HTTP.Paths[0].Backend.ServiceName)
	assert.Equal(t, intstr.IntOrString{IntVal: nais.DefaultPort}, ingress.Spec.Rules[0].HTTP.Paths[0].Backend.ServicePort)
}
