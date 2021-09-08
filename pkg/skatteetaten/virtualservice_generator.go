package generator

import (
	"fmt"

	skatteetaten_no_v1alpha1 "github.com/nais/liberator/pkg/apis/nebula.skatteetaten.no/v1alpha1"
	networking_istio_io_v1alpha3 "github.com/nais/liberator/pkg/apis/networking.istio.io/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GenerateVirtualService(application skatteetaten_no_v1alpha1.Application, ingress skatteetaten_no_v1alpha1.PublicIngressConfig) *networking_istio_io_v1alpha3.VirtualService {
	domain := "istio.nebula.dev.skatteetaten.io"

	// comet-comet-utv.<domain>
	fqdn := fmt.Sprintf("%s-%s.%s", application.Name, application.Namespace, domain)

	if len(ingress.HostPrefix) > 0 {
		// prefix-comet.comet-utv.<domain>
		fqdn = fmt.Sprintf("%s-%s", ingress.HostPrefix, fqdn)
	} else if len(ingress.OverrideHostname) > 0 {
		// override.<domain>
		fqdn = fmt.Sprintf("%s.%s", ingress.OverrideHostname, domain)
	}

	vs := &networking_istio_io_v1alpha3.VirtualService{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "networking.istio.io/v1alpha3",
			Kind:       "VirtualService",
		},
		ObjectMeta: application.StandardObjectMeta(),
		Spec: networking_istio_io_v1alpha3.VirtualServiceSpec{
			Hosts:    []string{fqdn},
			Gateways: []string{"istio-system/istio-ingress-gateway"},
			HTTP: []networking_istio_io_v1alpha3.HTTPRoute{{
				Route: []networking_istio_io_v1alpha3.HTTPRouteDestination{{
					Destination: networking_istio_io_v1alpha3.Destination{
						Host: fmt.Sprintf("%s.%s.svc.cluster.local", application.Name, application.Namespace),
						Port: networking_istio_io_v1alpha3.PortSelector{
							Number: uint32(ingress.Port),
						},
					},
				}},
			}},
		},
	}
	return vs
}
