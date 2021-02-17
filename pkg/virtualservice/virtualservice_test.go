package virtualservice_test

import (
	"testing"

	networking_istio_io_v1alpha3 "github.com/nais/liberator/pkg/apis/networking.istio.io/v1alpha3"
	"github.com/nais/naiserator/pkg/virtualservice"
	"gotest.tools/assert"
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
