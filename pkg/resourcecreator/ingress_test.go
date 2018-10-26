package resourcecreator_test

import (
	"testing"

	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/stretchr/testify/assert"
)

func TestIngress(t *testing.T) {
	app := fixtures.Application()
	ingress, err := resourcecreator.Ingress(app)

	assert.Nil(t, err)

	assert.Equal(t, app.Name, ingress.Name)
	assert.Equal(t, app.Namespace, ingress.Namespace)
	assert.Equal(t, "app.nais.adeo.no", ingress.Spec.Rules[0].Host)
	assert.Equal(t, "/", ingress.Spec.Rules[0].HTTP.Paths[0].Path)
	assert.Equal(t, app.Name, ingress.Spec.Rules[0].HTTP.Paths[0].Backend.ServiceName)
	assert.Equal(t, intstr.IntOrString{IntVal: nais.DefaultPort}, ingress.Spec.Rules[0].HTTP.Paths[0].Backend.ServicePort)
	assert.Equal(t, "true", ingress.ObjectMeta.Annotations["prometheus.io/scrape"])
	assert.Equal(t, app.Spec.Liveness.Path, ingress.ObjectMeta.Annotations["prometheus.io/path"])

	assert.Equal(t, "tjenester.nav.no", ingress.Spec.Rules[1].Host)
	assert.Equal(t, "/app", ingress.Spec.Rules[1].HTTP.Paths[0].Path)
	assert.Equal(t, app.Name, ingress.Spec.Rules[1].HTTP.Paths[0].Backend.ServiceName)
	assert.Equal(t, intstr.IntOrString{IntVal: nais.DefaultPort}, ingress.Spec.Rules[1].HTTP.Paths[0].Backend.ServicePort)
}

func TestIngressFailure(t *testing.T) {
	app := fixtures.Application()

	for _, i := range []string{"crap", "htp:/foo", "http://valid.fqdn/foo", "ftp://test"} {
		app.Spec.Ingresses = []string{
			i,
		}
		ingress, err := resourcecreator.Ingress(app)

		assert.NotNil(t, err)
		assert.Nil(t, ingress)
	}
}
