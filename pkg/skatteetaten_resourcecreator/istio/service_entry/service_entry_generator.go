package service_entry

import (
	skatteetaten_no_v1alpha1 "github.com/nais/liberator/pkg/apis/nebula.skatteetaten.no/v1alpha1"
	networking_istio_io_v1alpha3 "github.com/nais/liberator/pkg/apis/networking.istio.io/v1alpha3"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Source interface {
	resource.Source
	GetEgress() *skatteetaten_no_v1alpha1.EgressConfig
}

func Create(app Source, ast *resource.Ast) {
	egressConfig := app.GetEgress()

	// ServiceEntry
	if egressConfig != nil && egressConfig.External != nil {
		for _, egress := range egressConfig.External {
			generateServiceEntry(app, ast, egress)
		}
	}
}

func generateServiceEntry(source resource.Source, ast *resource.Ast, config skatteetaten_no_v1alpha1.ExternalEgressConfig){

	//TODO; vi hadde beta1
	serviceentry := networking_istio_io_v1alpha3.ServiceEntry{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceEntry",
			APIVersion: "networking.istio.io/v1alpha3",
		},
		ObjectMeta: resource.CreateObjectMeta(source),
		Spec:       networking_istio_io_v1alpha3.ServiceEntrySpec{},
	}
	serviceentry.Spec.Resolution = "2" //v1beta12.ServiceEntry_DNS
	serviceentry.Spec.Location = "0" //v1beta12.ServiceEntry_MESH_EXTERNAL
	serviceentry.Spec.Hosts = append(serviceentry.Spec.Hosts, config.Host)

	for _, port := range config.Ports {
		serviceentry.Spec.Ports = append(serviceentry.Spec.Ports, networking_istio_io_v1alpha3.Port{
			Number:   uint32(port.Port),
			Protocol: port.Protocol,
			Name:     port.Name,
		})
	}
	ast.AppendOperation(resource.OperationCreateOrUpdate, &serviceentry)

}
