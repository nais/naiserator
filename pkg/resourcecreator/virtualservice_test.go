package resourcecreator_test

import (
	"testing"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestVirtualService(t *testing.T) {

	t.Run("virtualservice created according to spec", func(t *testing.T) {
		ingresses := []string{
			"https://first.host",
			"https://second.host",
		}
		hosts := []string{
			"first.host",
			"second.host",
		}

		app := fixtures.MinimalApplication()
		app.Spec.Ingresses = ingresses
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		vs := resourcecreator.VirtualService(app)
		assert.NotNil(t, vs)

		assert.Equal(t, app.Name, vs.Name)
		assert.Equal(t, app.Namespace, vs.Namespace)
		assert.Equal(t, []string{resourcecreator.IstioVirtualServiceDefaultGateway}, vs.Spec.Gateways)
		assert.Equal(t, hosts, vs.Spec.Hosts)

		assert.Len(t, vs.Spec.HTTP, 1)
		assert.Len(t, vs.Spec.HTTP[0].Route, 1)
		route := vs.Spec.HTTP[0].Route[0]
		assert.Equal(t, app.Name, route.Destination.Host)
		assert.Equal(t, uint32(app.Spec.Service.Port), route.Destination.Port.Number)
		assert.Equal(t, resourcecreator.IstioVirtualServiceTotalWeight, route.Weight)
	})
}
