package resourcecreator_test

import (
	"net/url"
	"testing"

	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/resourcecreator"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestGatewayResolving(t *testing.T) {
	dashDomain := ".dash-domain.tld"
	subDomain := ".sub.domain.tld"

	mappings := []config.GatewayMapping{
		{dashDomain, "dashdomain-gateway"},
		{subDomain, "subdomain-gateway,subdomain-gateway2"},
	}

	ingressDashDomain := "https://x" + dashDomain
	ingressDashDomainWithPath := "https://x" + dashDomain + "/path"
	ingressSubDomain := "https://x" + subDomain
	ingressSubDomainWithPath := "https://x" + subDomain + "/path"

	assert.Equal(t, []string{"dashdomain-gateway"}, resourcecreator.ResolveGateway(asUrl(ingressDashDomain), mappings))
	assert.Equal(t, []string{"dashdomain-gateway"}, resourcecreator.ResolveGateway(asUrl(ingressDashDomainWithPath), mappings))
	assert.Equal(t, []string{"subdomain-gateway", "subdomain-gateway2"}, resourcecreator.ResolveGateway(asUrl(ingressSubDomain), mappings))
	assert.Equal(t, []string{"subdomain-gateway", "subdomain-gateway2"}, resourcecreator.ResolveGateway(asUrl(ingressSubDomainWithPath), mappings))
	assert.Equal(t, []string{"subdomain-gateway", "subdomain-gateway2"}, resourcecreator.ResolveGateway(asUrl(ingressSubDomainWithPath), mappings))
}

func asUrl(ingress string) url.URL {
	u, err := url.Parse(ingress)
	if err != nil {
		panic("unable to parse url: " + ingress)
	}
	return *u
}

func TestVirtualService(t *testing.T) {
	t.Run("virtualservices not created on invalid ingress", func(t *testing.T) {
		ingresses := []nais.Ingress{
			"host.no",
		}

		app := fixtures.MinimalApplication()
		app.Spec.Ingresses = ingresses
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		vses, err := resourcecreator.VirtualServices(app, nil)
		assert.Error(t, err)
		assert.Nil(t, vses)
	})

	t.Run("virtualservice created according to spec", func(t *testing.T) {
		ingresses := []nais.Ingress{
			"https://first.host.no/prefixed/with/url",
		}

		app := fixtures.MinimalApplication()
		app.Spec.Ingresses = ingresses
		err := nais.ApplyDefaults(app)
		assert.NoError(t, err)

		vses, err := resourcecreator.VirtualServices(app, []config.GatewayMapping{{
			DomainSuffix: ".host.no",
			GatewayName:  "istio-system/ingress-gateway-host-no",
		}})

		assert.Len(t, vses, 1)

		assert.Equal(t, "istio-system/ingress-gateway-host-no", vses[0].Spec.Gateways[0])
		assert.Len(t, vses[0].Spec.HTTP, 1)
		assert.Len(t, vses[0].Spec.HTTP[0].Route, 1)
		assert.Len(t, vses[0].Spec.HTTP[0].Match, 1)

		route := vses[0].Spec.HTTP[0].Route[0]
		assert.Equal(t, app.Name, route.Destination.Host)
		assert.Equal(t, "/prefixed/with/url", vses[0].Spec.HTTP[0].Match[0].URI.Prefix)
		assert.Equal(t, uint32(app.Spec.Service.Port), route.Destination.Port.Number)
		assert.Equal(t, resourcecreator.IstioVirtualServiceTotalWeight, route.Weight)
		assert.Equal(t, app.Namespace, vses[0].Namespace)
	})
}
