package resourcecreator_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/nais/naiserator/pkg/resourcecreator"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestVirtualService(t *testing.T) {
	t.Run("virtualservices have correct gateways", func(t *testing.T) {
		ingresses := []string{
			"https://host.no",
			"https://second.host.no",
			"https://subdomain.third.host.no",
		}

		app := fixtures.MinimalApplication()
		app.Spec.Ingresses = ingresses
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		vses, err := resourcecreator.VirtualServices(app)
		assert.Len(t, vses, 3)

		for i := range vses {
			assert.Equal(t, fmt.Sprintf(resourcecreator.IstioGatewayPrefix, "host-no"), vses[i].Spec.Gateways[0])
		}
	})

	t.Run("virtualservice gateway copes with missing TLD", func(t *testing.T) {
		ingresses := []string{
			"https://foo",
		}

		app := fixtures.MinimalApplication()
		app.Spec.Ingresses = ingresses
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		vses, err := resourcecreator.VirtualServices(app)
		assert.Len(t, vses, 1)
		assert.Equal(t, fmt.Sprintf(resourcecreator.IstioGatewayPrefix, "foo"), vses[0].Spec.Gateways[0])
	})

	t.Run("virtualservices not created on invalid ingress", func(t *testing.T) {
		ingresses := []string{
			"host.no",
		}

		app := fixtures.MinimalApplication()
		app.Spec.Ingresses = ingresses
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		vses, err := resourcecreator.VirtualServices(app)
		assert.Error(t, err)
		assert.Nil(t, vses)
	})

	t.Run("virtualservice created according to spec", func(t *testing.T) {
		ingresses := []string{
			"https://first.host.no/prefixed/with/url",
		}

		app := fixtures.MinimalApplication()
		app.Spec.Ingresses = ingresses
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		vses, err := resourcecreator.VirtualServices(app)
		assert.Len(t, vses, 1)

		assert.Equal(t, fmt.Sprintf(resourcecreator.IstioGatewayPrefix, "host-no"), vses[0].Spec.Gateways[0])
		assert.Len(t, vses[0].Spec.HTTP, 1)
		assert.Len(t, vses[0].Spec.HTTP[0].Route, 1)
		assert.Len(t, vses[0].Spec.HTTP[0].Match, 1)

		route := vses[0].Spec.HTTP[0].Route[0]
		assert.Equal(t, app.Name, route.Destination.Host)
		assert.Equal(t, "/prefixed/with/url", vses[0].Spec.HTTP[0].Match[0].URI.Prefix)
		assert.Equal(t, uint32(app.Spec.Service.Port), route.Destination.Port.Number)
		assert.Equal(t, resourcecreator.IstioVirtualServiceTotalWeight, route.Weight)
		assert.True(t, strings.HasPrefix(vses[0].Name, app.Name+"-first-host-no"))
		assert.Equal(t, app.Namespace, vses[0].Namespace)
	})
}
