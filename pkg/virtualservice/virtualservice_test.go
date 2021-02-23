package virtualservice_test

import (
	"sort"
	"testing"

	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/stretchr/testify/assert"

	"github.com/nais/naiserator/pkg/naiserator/config"
	"github.com/nais/naiserator/pkg/test/fixtures"

	networking_istio_io_v1alpha3 "github.com/nais/liberator/pkg/apis/networking.istio.io/v1alpha3"

	"github.com/nais/naiserator/pkg/virtualservice"
)

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

func TestAddIngressCollision(t *testing.T) {

	const namespace = "vs-namespace"

	gatewayMappings := []config.GatewayMapping{
		{
			DomainSuffix: ".nav.no",
			GatewayName:  "istio-system/gw-nav-no",
		},
	}
	registry := virtualservice.NewRegistry(gatewayMappings, namespace)

	app1 := fixtures.MinimalApplication()
	app1.Name = "first-app"
	app1.Spec.Ingresses = []nais_io_v1alpha1.Ingress{
		"https://www.nav.no/first-app",
	}

	app2 := fixtures.MinimalApplication()
	app2.Name = "second-app"
	app2.Spec.Ingresses = []nais_io_v1alpha1.Ingress{
		"https://www.nav.no/first-app",
	}

	assert.NoError(t, registry.Add(app1))
	assert.EqualError(t, registry.Add(app2), "the ingress https://www.nav.no/first-app is already in use by first-app.mynamespace.svc.cluster.local")
}

func TestVirtualServices(t *testing.T) {
	gatewayMappings := []config.GatewayMapping{
		{
			DomainSuffix: ".nav.no",
			GatewayName:  "istio-system/gw-nav-no",
		},
		{
			DomainSuffix: ".nav2.no",
			GatewayName:  "istio-system/gw-nav2-no",
		},
	}
	registry := virtualservice.NewRegistry(gatewayMappings, "namespace")

	app := fixtures.MinimalApplication()
	app.Spec.Ingresses = []nais_io_v1alpha1.Ingress{
		"https://www.nav.no/some",
		"https://www.nav.no/other/path",
		"https://app1.nav.no/some",
		"https://www.nav2.no/some",
	}
	assert.NoError(t, registry.Add(app))

	services, err := registry.VirtualServices(app)
	assert.NoError(t, err)
	hosts := make([]string, 0)
	for _, vs := range services {
		hosts = append(hosts, vs.Spec.Hosts...)
	}

	expectedHosts := []string{"www.nav.no", "app1.nav.no", "www.nav2.no"}
	sort.Strings(expectedHosts)
	sort.Strings(hosts)
	assert.Equal(t, expectedHosts, hosts)
}

func TestVirtualServicesGatewayMappingNotFound(t *testing.T) {
	gatewayMappings := []config.GatewayMapping{
		{
			DomainSuffix: ".nav.no",
			GatewayName:  "istio-system/gw-nav-no",
		},
	}
	registry := virtualservice.NewRegistry(gatewayMappings, "namespace")

	app := fixtures.MinimalApplication()
	app.Spec.Ingresses = []nais_io_v1alpha1.Ingress{
		"https://www.nav.no/some",
		"https://www.nav.no/other/path",
		"https://app1.nav.no/some",
		"https://www.nav2.no/some",
	}
	assert.EqualError(t, registry.Add(app), "'www.nav2.no' is not a supported domain")
}

func TestVirtualServicesMultipleApps(t *testing.T) {
	const namespace = "vs-namespace"
	const appNamespace = "app-namespace"

	gatewayMappings := []config.GatewayMapping{
		{
			DomainSuffix: ".nav.no",
			GatewayName:  "istio-system/gw-nav-no",
		},
	}
	registry := virtualservice.NewRegistry(gatewayMappings, namespace)

	app1 := fixtures.MinimalApplication()
	app1.Name = "first-app"
	app1.Namespace = appNamespace
	app1.Spec.Ingresses = []nais_io_v1alpha1.Ingress{
		"https://www.nav.no/first-app",
	}

	app2 := fixtures.MinimalApplication()
	app2.Name = "second-app"
	app2.Namespace = appNamespace
	app2.Spec.Ingresses = []nais_io_v1alpha1.Ingress{
		"https://www.nav.no/second-app",
		"https://www.nav.no/second-app-other-url",
	}

	assert.NoError(t, registry.Add(app1))
	assert.NoError(t, registry.Add(app2))

	svc1, _ := registry.VirtualServices(app1)
	svc2, _ := registry.VirtualServices(app2)

	// both apps use only one domain, and they are the same, so config should be equal
	assert.Equal(t, svc2, svc1)

	// should have only one virtual service
	assert.Len(t, svc1, 1)

	vs := svc1[0]

	// should have three routes, one for first-app and two for second-app
	assert.Len(t, vs.Spec.HTTP, 3)
	assert.Equal(t, "/second-app-other-url(/.*)?", vs.Spec.HTTP[0].Match[0].URI.Regex)
	assert.Equal(t, "second-app."+appNamespace+virtualservice.ServiceSuffix, vs.Spec.HTTP[0].Route[0].Destination.Host)
	assert.Equal(t, "/second-app(/.*)?", vs.Spec.HTTP[1].Match[0].URI.Regex)
	assert.Equal(t, "second-app."+appNamespace+virtualservice.ServiceSuffix, vs.Spec.HTTP[1].Route[0].Destination.Host)
	assert.Equal(t, "/first-app(/.*)?", vs.Spec.HTTP[2].Match[0].URI.Regex)
	assert.Equal(t, "first-app."+appNamespace+virtualservice.ServiceSuffix, vs.Spec.HTTP[2].Route[0].Destination.Host)
}

func TestDelete(t *testing.T) {
	gatewayMappings := []config.GatewayMapping{
		{
			DomainSuffix: ".nav.no",
			GatewayName:  "istio-system/gw-nav-no",
		},
	}
	registry := virtualservice.NewRegistry(gatewayMappings, "namespace")

	app1 := fixtures.MinimalApplication()
	app1.Spec.Ingresses = []nais_io_v1alpha1.Ingress{
		"https://www.nav.no/one/path",
	}
	assert.NoError(t, registry.Add(app1))

	app2 := fixtures.MinimalApplication()
	app2.Name = "app-2"
	app2.Spec.Ingresses = []nais_io_v1alpha1.Ingress{
		"https://www.nav.no/second/path",
	}
	assert.NoError(t, registry.Add(app2))

	services := registry.Remove(app1.Name, app1.Namespace)

	vs := services[0]
	assert.Len(t, services, 1)
	assert.Len(t, vs.Spec.HTTP, 1)
	assert.Len(t, vs.Spec.HTTP[0].Match, 1)
	assert.Len(t, vs.Spec.HTTP[0].Route, 1)
	assert.Equal(t, "/second/path(/.*)?", vs.Spec.HTTP[0].Match[0].URI.Regex)
	assert.Equal(t, "app-2.mynamespace.svc.cluster.local", vs.Spec.HTTP[0].Route[0].Destination.Host)
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
	gatewayMappings := []config.GatewayMapping{
		{
			DomainSuffix: ".nav.no",
			GatewayName:  "istio-system/gw-nav-no",
		},
	}
	registry := virtualservice.NewRegistry(gatewayMappings, "namespace")

	app := fixtures.MinimalApplication()
	app.Spec.Ingresses = []nais_io_v1alpha1.Ingress{
		"https://www.nav.no/base/some/thing",
	}

	err := registry.Add(app)
	assert.NoError(t, err)

	app.Spec.Ingresses = []nais_io_v1alpha1.Ingress{
		"https://www.nav.no/base/other/path",
	}

	app.Name = "other-app"
	err = registry.Add(app)
	assert.NoError(t, err)

	vs := registry.VirtualService("www.nav.no")

	assert.Equal(t, []string{"istio-system/gw-nav-no"}, vs.Spec.Gateways)

	// Test that two HTTP routes are set up, the longest path first
	assert.Len(t, vs.Spec.HTTP, 2)
	assert.Equal(t, "/base/other/path(/.*)?", vs.Spec.HTTP[0].Match[0].URI.Regex)
	assert.Equal(t, "/base/some/thing(/.*)?", vs.Spec.HTTP[1].Match[0].URI.Regex)
}

func TestAppLifecycle(t *testing.T) {
	gatewayMappings := []config.GatewayMapping{
		{
			DomainSuffix: ".nav.no",
			GatewayName:  "istio-system/gw-nav-no",
		},
	}
	registry := virtualservice.NewRegistry(gatewayMappings, "namespace")

	app := fixtures.MinimalApplication()
	app.Spec.Ingresses = []nais_io_v1alpha1.Ingress{
		"https://www.nav.no/base/some/thing",
	}

	err := registry.Add(app)
	assert.NoError(t, err)

	app.Spec.Ingresses = []nais_io_v1alpha1.Ingress{
		"https://www.nav.no/base/some/thing/changed",
	}

	err = registry.Add(app)
	assert.NoError(t, err)

	vs := registry.VirtualService("www.nav.no")

	// Test that only the last HTTP route is set up
	assert.Len(t, vs.Spec.HTTP, 1)
	assert.Equal(t, "/base/some/thing/changed(/.*)?", vs.Spec.HTTP[0].Match[0].URI.Regex)
}

func TestRoutes(t *testing.T) {
	app := fixtures.MinimalApplication()
	app.Spec.Ingresses = []nais_io_v1alpha1.Ingress{
		"https://www.nav.no/base/some/thing",
		"https://other.domain/base",
	}

	gatewayMappings := []config.GatewayMapping{
		{
			DomainSuffix: ".nav.no",
			GatewayName:  "istio-system/gw-nav-no",
		},
		{
			DomainSuffix: "other.domain",
			GatewayName:  "istio-system/gw-other-domain",
		},
	}
	registry := virtualservice.NewRegistry(gatewayMappings, "namespace")

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
						Host: app.Name + "." + app.Namespace + virtualservice.ServiceSuffix,
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
