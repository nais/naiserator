package virtualservice_test

import (
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/test/fixtures"
	"github.com/stretchr/testify/assert"
	"testing"

	networking_istio_io_v1alpha3 "github.com/nais/liberator/pkg/apis/networking.istio.io/v1alpha3"
	"github.com/nais/naiserator/pkg/virtualservice"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

var dst = networking_istio_io_v1alpha3.VirtualService{
	TypeMeta:   v1.TypeMeta{},
	ObjectMeta: v1.ObjectMeta{},
	Spec: networking_istio_io_v1alpha3.VirtualServiceSpec{
		Gateways: nil,
	},
}

func simpleRoute(uri string) networking_istio_io_v1alpha3.HTTPRoute {
	return networking_istio_io_v1alpha3.HTTPRoute{
		Match: []networking_istio_io_v1alpha3.HTTPMatchRequest{
			{
				URI: networking_istio_io_v1alpha3.StringMatch{
					Regex: uri,
				},
			},
		},
		Route: []networking_istio_io_v1alpha3.HTTPRouteDestination{
			{
				Destination: networking_istio_io_v1alpha3.Destination{
					Host: "simplehost",
					Port: networking_istio_io_v1alpha3.PortSelector{
						Number: 80,
					},
				},
				Weight: 100,
			},
		},
	}
}

func TestSortRoutes(t *testing.T) {
	routes := []networking_istio_io_v1alpha3.HTTPRoute{
		simpleRoute("/base"),
		simpleRoute("/base/some"),
		simpleRoute("/base/some/thing"),
		simpleRoute("/base/something"),
		simpleRoute("/base/verylongstring"),
		simpleRoute("/base/other"),
		simpleRoute("/base/zzz"),
		simpleRoute("/base/def"),
		simpleRoute("/base/abc"),
		simpleRoute("/zzzzzzz/many/path/segments/foo"),
	}

	correctOrder := []string{
		"/zzzzzzz/many/path/segments/foo",
		"/base/some/thing",
		"/base/verylongstring",
		"/base/something",
		"/base/other",
		"/base/some",
		"/base/abc",
		"/base/def",
		"/base/zzz",
		"/base",
	}

	if len(routes) != len(correctOrder) {
		panic("pebkac: number of routes must match number of strings in correct order")
	}

	virtualservice.SortRoutes(routes)

	for i := range routes {
		assert.Equal(t, correctOrder[i], routes[i].Match[0].URI.Regex)
	}
}

func TestVirtualService(t *testing.T) {
	app := fixtures.MinimalApplication()
	app.Spec.Ingresses = []nais_io_v1alpha1.Ingress{
		"https://www.nav.no/base/some/thing",
	}
	gatewayMappings := []config.GatewayMapping{
		{
			DomainSuffix: ".nav.no",
			GatewayName:  "istio-system/gw-nav-no",
		},
	}
	registry := virtualservice.New(gatewayMappings)

	err := registry.Add(app)
	assert.NoError(t, err)

	vs := registry.VirtualService("www.nav.no")

	assert.Equal(t, []string{"istio-system/gw-nav-no"}, vs.Spec.Gateways)
}

func TestRoutes(t *testing.T) {
	app := fixtures.MinimalApplication()
	app.Spec.Ingresses = []nais_io_v1alpha1.Ingress{
		"https://www.nav.no/base/some/thing",
	}

	gatewayMappings := []config.GatewayMapping{
		{
			DomainSuffix: ".nav.no",
			GatewayName:  "istio-system/gw-nav-no",
		},
	}
	registry := virtualservice.New(gatewayMappings)

	err := registry.Add(app)
	assert.NoError(t, err)

	routes := registry.Routes("www.nav.no")

	expectedRoutes := []networking_istio_io_v1alpha3.HTTPRoute{
		{
			Match: []networking_istio_io_v1alpha3.HTTPMatchRequest{
				{
					URI: networking_istio_io_v1alpha3.StringMatch{
						Regex: "/base/some/thing(/.*)?",
					},
				},
			},
			Route: []networking_istio_io_v1alpha3.HTTPRouteDestination{
				{
					Destination: networking_istio_io_v1alpha3.Destination{
						Host: app.Name,
						Port: networking_istio_io_v1alpha3.PortSelector{
							Number: uint32(app.Spec.Service.Port),
						},
					},
					Weight: virtualservice.IstioVirtualServiceTotalWeight,
				},
			},
		},
	}

	assert.Equal(t, expectedRoutes, routes)
}
