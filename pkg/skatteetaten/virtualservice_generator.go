package generator

import (
	"fmt"
	"github.com/skatteetaten-trial/nebula-application-operator/pkg/api/v1alpha1"
	v1beta12 "istio.io/api/networking/v1beta1"
	"istio.io/client-go/pkg/apis/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GenerateVirtualService(application v1alpha1.Application, ingress v1alpha1.PublicIngressConfig) *v1beta1.VirtualService {
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

	vs := &v1beta1.VirtualService{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "networking.istio.io/v1beta1",
			Kind:       "VirtualService",
		},
		ObjectMeta: application.StandardObjectMeta(),
		Spec: v1beta12.VirtualService{
			Hosts:    []string{fqdn},
			Gateways: []string{"istio-system/istio-ingress-gateway"},
			Http: []*v1beta12.HTTPRoute{{
				Route: []*v1beta12.HTTPRouteDestination{{
					Destination: &v1beta12.Destination{
						Host: fmt.Sprintf("%s.%s.svc.cluster.local", application.Name, application.Namespace),
						Port: &v1beta12.PortSelector{
							Number: uint32(ingress.Port),
						},
					},
				}},
			}},
		},
	}
	return vs
}
