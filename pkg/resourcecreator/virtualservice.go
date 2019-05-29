package resourcecreator

import (
	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	istio "github.com/nais/naiserator/pkg/apis/networking.istio.io/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func VirtualService(app *nais.Application) *istio.VirtualService {
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
			Hosts: app.Spec.Ingresses,
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
						},
					},
				},
			},
		},
	}
}
