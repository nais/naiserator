package resourcecreator

import (
	"fmt"
	"net/url"
	"strings"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	istio "github.com/nais/naiserator/pkg/apis/networking.istio.io/v1alpha3"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func VirtualServices(app *nais.Application) (vses []*istio.VirtualService, err error) {
	if len(app.Spec.Ingresses) == 0 {
		return nil, nil
	}

	for _, ingress := range app.Spec.Ingresses {
		parsedUrl, err := url.Parse(ingress)
		if err != nil {
			return nil, fmt.Errorf("failed to parse URL '%s': %s", ingress, err)
		}
		if len(parsedUrl.Path) == 0 {
			parsedUrl.Path = "/"
		}
		err = validateUrl(parsedUrl)
		if err != nil {
			return nil, err
		}

		vs := virtualService(*parsedUrl, app)
		vses = append(vses, &vs)
	}

	return
}

func virtualService(ingress url.URL, app *nais.Application) istio.VirtualService {
	domainID := istioDomainID(ingress)

	objectMeta := app.CreateObjectMeta()
	objectMeta.Name = fmt.Sprintf("%s-%s", app.Name, strings.ReplaceAll(ingress.Hostname(), ".", "-"))

	return istio.VirtualService{
		TypeMeta: v1.TypeMeta{
			Kind:       "VirtualService",
			APIVersion: IstioNetworkingAPIVersion,
		},
		ObjectMeta: objectMeta,
		Spec: istio.VirtualServiceSpec{
			Gateways: []string{
				fmt.Sprintf(IstioGatewayPrefix, domainID),
			},
			Hosts: []string{ingress.Hostname()},
			HTTP: []istio.HTTPRoute{
				{
					Match: []istio.HTTPMatchRequest{
						{
							URI: istio.StringMatch{
								Prefix: ingress.Path,
							},
						},
					},
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

// returns the mid-level and top-level domain separated with a dash
func istioDomainID(ingress url.URL) string {
	parts := strings.Split(ingress.Hostname(), ".")
	if len(parts) > 2 {
		parts = parts[len(parts)-2:]
	}
	return strings.Join(parts, "-")
}
