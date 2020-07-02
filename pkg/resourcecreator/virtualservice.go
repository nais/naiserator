package resourcecreator

import (
	"fmt"
	"net/url"
	"strings"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	istio "github.com/nais/naiserator/pkg/apis/networking.istio.io/v1alpha3"
	"github.com/nais/naiserator/pkg/naiserator/config"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func VirtualServices(app *nais.Application, gatewayMappings []config.GatewayMapping) ([]*istio.VirtualService, error) {
	var vses []*istio.VirtualService

	for index, ingress := range app.Spec.Ingresses {
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

		name := fmt.Sprintf("%s-%02d", app.Name, index)
		vs := virtualService(*parsedUrl, gatewayMappings, app, name)
		vses = append(vses, &vs)
	}

	return vses, nil
}

func Gateway(ingress url.URL, mappings []config.GatewayMapping) string {
	for _, mapping := range mappings {
		if strings.HasSuffix(ingress.Host, mapping.DomainSuffix) {
			return mapping.GatewayName
		}
	}
	return ""
}

func virtualService(ingress url.URL, gatewayMappings []config.GatewayMapping, app *nais.Application, name string) istio.VirtualService {
	objectMeta := app.CreateObjectMetaWithName(name)

	return istio.VirtualService{
		TypeMeta: v1.TypeMeta{
			Kind:       "VirtualService",
			APIVersion: IstioNetworkingAPIVersion,
		},
		ObjectMeta: objectMeta,
		Spec: istio.VirtualServiceSpec{
			Gateways: []string{
				Gateway(ingress, gatewayMappings),
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
