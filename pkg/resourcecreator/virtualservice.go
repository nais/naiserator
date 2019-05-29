package resourcecreator

import (
	"net/url"

	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	istio "github.com/nais/naiserator/pkg/apis/networking.istio.io/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func VirtualService(app *nais.Application) *istio.VirtualService {
	hosts := make([]string, len(app.Spec.Ingresses))

	for i := range app.Spec.Ingresses {
		hosts[i] = getHostByURL(app.Spec.Ingresses[i])
	}

	return &istio.VirtualService{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VirtualService",
			APIVersion: "v1alpha3",
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec: istio.VirtualServiceSpec{
			Gateways: []string{
				"ingress-gateway",
			},
			Hosts: hosts,
			HTTP: []istio.HTTPRoute{
				istio.HTTPRoute{
					Route: []istio.HTTPRouteDestination{
						istio.HTTPRouteDestination{
							Destination: istio.Destination{
								Host: app.Name,
								Port: istio.PortSelector{
									Number: uint32(app.Spec.Service.Port),
								},
							},
							Weight: 100,
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
