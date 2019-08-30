package resourcecreator

import (
	"fmt"
	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	istio "github.com/nais/naiserator/pkg/apis/networking.istio.io/v1alpha3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/url"
	"strings"
)

func VirtualServices(app *nais.Application) (vses []*istio.VirtualService, err error) {
	if len(app.Spec.Ingresses) == 0 {
		return nil, nil
	}

	ingresses, err := sanitize(app.Spec.Ingresses)
	if err != nil {
		return nil, err
	}

	for _, ingress := range ingresses {
		vses = append(vses, virtualService(ingress, app))
	}

	return
}

func virtualService(ingress *url.URL, app *nais.Application) *istio.VirtualService {
	return &istio.VirtualService{
		TypeMeta: v1.TypeMeta{
			Kind:       "VirtualService",
			APIVersion: IstioNetworkingAPIVersion,
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec: istio.VirtualServiceSpec{
			Gateways: []string{
				fmt.Sprintf(IstioGatewayPrefix, domain(ingress)),
			},
			Hosts: []string{ingress.Hostname()},
			HTTP: []istio.HTTPRoute{
				{
					Route: []istio.HTTPRouteDestination{
						{
							Destination: istio.Destination{
								Host: app.Name,
								Port: istio.PortSelector{
									Number: uint32(app.Spec.Service.Port),
								},
							},
							Weight: IstioVirtualServiceTotalWeight,
						},
					},
				},
			},
		},
	}
}

// sanitize takes a slice of string assumed to either be hostnames or URLs and returns valid URLs
func sanitize(hosts []string) (sanitized []*url.URL, err error) {
	for _, host := range hosts {
		// parse raw input, and return error if found
		if _, err := url.Parse(host); err != nil {
			return nil, fmt.Errorf("unable to parse host: '%s'. %s", host, err)
		}

		// Remove and re-append http(s) as protocol to unify the host format.
		// This allows us to use url.Parse function for validating and operating on the parts of the URL.
		// Assumes we are only dealing with http in this context.
		host = fmt.Sprintf("https://%s", strings.TrimPrefix(strings.TrimPrefix(host, "http://"), "https://"))
		parsedUrl, err := url.ParseRequestURI(host)
		if err != nil {
			return nil, fmt.Errorf("unable to parse host: '%s'. %s", host, err)
		}

		sanitized = append(sanitized, parsedUrl)
	}

	return
}

// domain returns the mid-level and top-level domain separated with a dash
func domain(ingress *url.URL) string {
	parts := strings.Split(ingress.Hostname(), ".")

	return parts[len(parts)-2] + "-" + parts[len(parts)-1]
}
