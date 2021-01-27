package resourcecreator

import (
	"fmt"
	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	istio "github.com/nais/naiserator/pkg/apis/networking.istio.io/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/url"
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

		entry := &istio.ServiceEntry{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ServiceEntry",
				APIVersion: IstioNetworkingAPIVersion,
			},
			ObjectMeta: meta,
			Spec: istio.ServiceEntrySpec{
				Hosts:      []string{stripProtocolFromHost(ext.Host)},
				Location:   IstioServiceEntryLocationExternal,
				Resolution: IstioServiceEntryResolutionDNS,
				Ports:      ports,
				ExportTo:   []string{"."},
			},
		}
		entries = append(entries, entry)
	}

	return entries
}

func serviceEntryPort(rule nais.AccessPolicyPortRule) istio.Port {
	return istio.Port{
		Name:     rule.Name,
		Number:   rule.Port,
		Protocol: rule.Protocol,
	}
}

func stripProtocolFromHost(host string) string {
	u, err := url.Parse(host)
	if err != nil || len(u.Host) == 0 {
		return host
	}
	return u.Host
}
