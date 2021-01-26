package resourcecreator

import (
	"fmt"
	"net/url"
	"strings"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	istio "github.com/nais/naiserator/pkg/apis/networking.istio.io/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ServiceEntries(app *nais.Application) ([]*istio.ServiceEntry, error) {
	entries := make([]*istio.ServiceEntry, 0)

	if len(app.Spec.AccessPolicy.Outbound.External) == 0 {
		return entries, nil
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
		host, err := stripProtocolFromHost(ext.Host)
		if err != nil {
			return nil, err
		}
		entry := &istio.ServiceEntry{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ServiceEntry",
				APIVersion: IstioNetworkingAPIVersion,
			},
			ObjectMeta: meta,
			Spec: istio.ServiceEntrySpec{
				Hosts:      []string{host},
				Location:   IstioServiceEntryLocationExternal,
				Resolution: IstioServiceEntryResolutionDNS,
				Ports:      ports,
				ExportTo:   []string{"."},
			},
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func serviceEntryPort(rule nais.AccessPolicyPortRule) istio.Port {
	return istio.Port{
		Name:     rule.Name,
		Number:   rule.Port,
		Protocol: rule.Protocol,
	}
}

func stripProtocolFromHost(host string) (string, error) {
	if strings.HasPrefix(host, "https://") || strings.HasPrefix(host, "http://") {
		u, err := url.Parse(host)
		if err != nil {
			return "", fmt.Errorf("parsing URL from '%s': %s", host, err)
		}
		return u.Host, nil
	}
	return host, nil
}
