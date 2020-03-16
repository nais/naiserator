package resourcecreator

import (
	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	istio "github.com/nais/naiserator/pkg/apis/networking.istio.io/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ServiceEntry(app *nais.Application) *istio.ServiceEntry {
	if len(app.Spec.AccessPolicy.Outbound.External) == 0 {
		return nil
	}

	ports := make([]istio.Port, 0)

	hosts := make([]string, len(app.Spec.AccessPolicy.Outbound.External))
	for i, ext := range app.Spec.AccessPolicy.Outbound.External {
		hosts[i] = ext.Host
		for _, port := range ext.Ports {
			ports = append(ports, serviceEntryPort(port))
		}
	}

	if len(ports) == 0 {
		ports = append(ports, istio.Port{
			Name:     "https",
			Protocol: "HTTPS",
			Number:   443,
		})
	}

	return &istio.ServiceEntry{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceEntry",
			APIVersion: IstioNetworkingAPIVersion,
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec: istio.ServiceEntrySpec{
			Hosts:      hosts,
			Location:   IstioServiceEntryLocationExternal,
			Resolution: IstioServiceEntryResolutionDNS,
			Ports:      ports,
		},
	}
}

func serviceEntryPort(rule nais.AccessPolicyPortRule) istio.Port {
	return istio.Port{
		Name:     rule.Name,
		Number:   rule.Port,
		Protocol: rule.Protocol,
	}
}
