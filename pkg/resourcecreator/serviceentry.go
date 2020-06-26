package resourcecreator

import (
	"fmt"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	istio "github.com/nais/naiserator/pkg/apis/networking.istio.io/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ServiceEntries(app *nais.Application) []*istio.ServiceEntry {
	entries := make([]*istio.ServiceEntry, 0)

	if len(app.Spec.AccessPolicy.Outbound.External) == 0 {
		return entries
	}

	for i, ext := range app.Spec.AccessPolicy.Outbound.External {
		meta := app.CreateObjectMetaWithName(fmt.Sprintf("%s-%02d", app.Name, i+1))
		ports := make([]istio.Port, 0)
		for _, port := range ext.Ports {
			ports = append(ports, serviceEntryPort(port))
		}
		if len(ports) == 0 {
			ports = append(ports, istio.Port{
				Name:     "https",
				Protocol: "HTTPS",
				Number:   443,
			})
		}
		spec := serviceEntrySpec(ext.Host, ext.IPAddress, ports)
		entry := &istio.ServiceEntry{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ServiceEntry",
				APIVersion: IstioNetworkingAPIVersion,
			},
			ObjectMeta: meta,
			Spec:       spec,
		}
		entries = append(entries, entry)
	}

	return entries
}

func serviceEntrySpec(host, address string, ports []istio.Port) istio.ServiceEntrySpec {
	spec := istio.ServiceEntrySpec{
		Hosts:      []string{host},
		Location:   IstioServiceEntryLocationExternal,
		Resolution: IstioServiceEntryResolutionDNS,
		Ports:      ports,
	}
	if address != "" {
		spec.Addresses = []string{address}
	}
	return spec
}

func serviceEntryPort(rule nais.AccessPolicyPortRule) istio.Port {
	return istio.Port{
		Name:     rule.Name,
		Number:   rule.Port,
		Protocol: rule.Protocol,
	}
}
