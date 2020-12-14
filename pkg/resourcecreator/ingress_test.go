package resourcecreator_test

import (
	"testing"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestIngress(t *testing.T) {
	t.Run("ingress creation is successful and resources look correct", func(t *testing.T) {
		app := fixtures.MinimalApplication()
		app.Spec.Ingresses = []nais.Ingress{
			"https://app.nais.adeo.no/",
			"https://tjenester.nav.no/app",
			"https://app.foo.bar",
		}
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		ingress, err := resourcecreator.Ingress(app)
		assert.Nil(t, err)

		assert.Equal(t, app.Name, ingress.Name)
		assert.Equal(t, app.Namespace, ingress.Namespace)
		assert.Equal(t, "app.nais.adeo.no", ingress.Spec.Rules[0].Host)
		assert.Equal(t, "/", ingress.Spec.Rules[0].HTTP.Paths[0].Path)
		assert.Equal(t, app.Name, ingress.Spec.Rules[0].HTTP.Paths[0].Backend.ServiceName)
		assert.Equal(t, intstr.IntOrString{IntVal: nais.DefaultServicePort}, ingress.Spec.Rules[0].HTTP.Paths[0].Backend.ServicePort)
		assert.Equal(t, "true", ingress.ObjectMeta.Annotations["prometheus.io/scrape"])
		assert.Equal(t, app.Spec.Liveness.Path, ingress.ObjectMeta.Annotations["prometheus.io/path"])

		assert.Equal(t, "tjenester.nav.no", ingress.Spec.Rules[1].Host)
		assert.Equal(t, "/app", ingress.Spec.Rules[1].HTTP.Paths[0].Path)
		assert.Equal(t, app.Name, ingress.Spec.Rules[1].HTTP.Paths[0].Backend.ServiceName)
		assert.Equal(t, intstr.IntOrString{IntVal: nais.DefaultServicePort}, ingress.Spec.Rules[1].HTTP.Paths[0].Backend.ServicePort)

		assert.Equal(t, "app.foo.bar", ingress.Spec.Rules[2].Host)
		assert.Equal(t, "/", ingress.Spec.Rules[2].HTTP.Paths[0].Path)
		assert.Equal(t, app.Name, ingress.Spec.Rules[2].HTTP.Paths[0].Backend.ServiceName)
		assert.Equal(t, intstr.IntOrString{IntVal: nais.DefaultServicePort}, ingress.Spec.Rules[2].HTTP.Paths[0].Backend.ServicePort)
	})

	t.Run("invalid ingress URLs are rejected", func(t *testing.T) {
		for _, i := range []nais.Ingress{"crap", "htp:/foo", "http://valid.fqdn/foo", "ftp://test"} {
			app := fixtures.MinimalApplication()
			app.Spec.Ingresses = []nais.Ingress{i}
			err := nais.ApplyDefaults(app)
			assert.NoError(t, err)

			ingress, err := resourcecreator.Ingress(app)

			assert.NotNil(t, err)
			assert.Nil(t, ingress)
		}
	})
}
