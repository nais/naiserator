package resourcecreator_test

import (
	"fmt"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"strings"
	"testing"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestVirtualService(t *testing.T) {
	t.Run("virtualservice created according to spec", func(t *testing.T) {
		ingresses := []string{
			"first.host.no",
			"https://second.host.no",
			"http://another.domain.tk",
		}

		app := fixtures.MinimalApplication()
		app.Spec.Ingresses = ingresses
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		vses,err  := resourcecreator.VirtualServices(app)
		assert.Len(t, vses, 3)

		assert.Equal(t, fmt.Sprintf(resourcecreator.IstioGatewayPrefix, "host-no"), vses[0].Spec.Gateways[0])
		assert.Len(t, vses[0].Spec.HTTP, 1)
		assert.Len(t, vses[0].Spec.HTTP[0].Route, 1)
		route := vses[0].Spec.HTTP[0].Route[0]
		assert.Equal(t, app.Name, route.Destination.Host)
		assert.Equal(t, uint32(app.Spec.Service.Port), route.Destination.Port.Number)
		assert.Equal(t, resourcecreator.IstioVirtualServiceTotalWeight, route.Weight)
		assert.True(t, strings.HasPrefix(vses[0].Name, app.Name + "-first-host-no"))
		assert.Equal(t, app.Namespace, vses[0].Namespace)
	})
}
