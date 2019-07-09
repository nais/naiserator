package resourcecreator

import (
	"net/url"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	istio "github.com/nais/naiserator/pkg/apis/networking.istio.io/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func VirtualService(app *nais.Application) *istio.VirtualService {
	if len(app.Spec.Ingresses) == 0 {
		return nil
	}

	hosts := make([]string, len(app.Spec.Ingresses))

	for i := range app.Spec.Ingresses {
		hosts[i] = getHostByURL(app.Spec.Ingresses[i])
	}

	return &istio.VirtualService{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VirtualService",
			APIVersion: IstioNetworkingAPIVersion,
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec: istio.VirtualServiceSpec{
			Gateways: []string{
				IstioVirtualServiceDefaultGateway,
			},
			Hosts: hosts,
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

func getHostByURL(s string) string {
	u, _ := url.Parse(s)
	return u.Host
}
