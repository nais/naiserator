package generator

import (
	"github.com/skatteetaten-trial/nebula-application-operator/pkg/api/v1alpha1"
	v1beta12 "istio.io/api/networking/v1beta1"
	"istio.io/client-go/pkg/apis/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GenerateServiceEntry(application v1alpha1.Application, config v1alpha1.ExternalEgressConfig) *v1beta1.ServiceEntry {

	serviceentry := v1beta1.ServiceEntry{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceEntry",
			APIVersion: "networking.istio.io/v1beta1",
		},
		ObjectMeta: application.StandardObjectMeta(),
		Spec:       v1beta12.ServiceEntry{},
	}

	serviceentry.Spec.Resolution = v1beta12.ServiceEntry_DNS
	serviceentry.Spec.Location = v1beta12.ServiceEntry_MESH_EXTERNAL
	serviceentry.Spec.Hosts = append(serviceentry.Spec.Hosts, config.Host)

	for _, port := range config.Ports {
		serviceentry.Spec.Ports = append(serviceentry.Spec.Ports, &v1beta12.Port{
			Number:   uint32(port.Port),
			Protocol: port.Protocol,
			Name:     port.Name,
		})
	}
	return &serviceentry
}
