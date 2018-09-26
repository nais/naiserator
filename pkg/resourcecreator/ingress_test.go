package resourcecreator

import (
	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetIngresses(t *testing.T) {
	app := getExampleApp()
	ingresses := ingresses(app)

	assert.Equal(t, app.Name, ingresses[0].Name)
	assert.Equal(t, app.Namespace, ingresses[0].Namespace)
	assert.Equal(t, "app.nais.adeo.no", ingresses[0].Spec.Rules[0].Host)
	assert.Equal(t, "/", ingresses[0].Spec.Rules[0].HTTP.Paths[0].Path)
	assert.Equal(t, app.Name, ingresses[0].Spec.Rules[0].HTTP.Paths[0].Backend.ServiceName)
	assert.Equal(t, intstr.IntOrString{IntVal: nais.DefaultPort}, ingresses[0].Spec.Rules[0].HTTP.Paths[0].Backend.ServicePort)

	assert.Equal(t, app.Name, ingresses[1].Name)
	assert.Equal(t, app.Namespace, ingresses[1].Namespace)
	assert.Equal(t, "tjenester.nav.no", ingresses[1].Spec.Rules[0].Host)
	assert.Equal(t, "/app", ingresses[1].Spec.Rules[0].HTTP.Paths[0].Path)
	assert.Equal(t, app.Name, ingresses[1].Spec.Rules[0].HTTP.Paths[0].Backend.ServiceName)
	assert.Equal(t, intstr.IntOrString{IntVal: nais.DefaultPort}, ingresses[1].Spec.Rules[0].HTTP.Paths[0].Backend.ServicePort)
}
