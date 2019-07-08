package resourcecreator

import (
	nais "github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	istio "github.com/nais/naiserator/pkg/apis/networking.istio.io/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ServiceEntry(app *nais.Application) *istio.ServiceEntry {
	if len(app.Spec.AccessPolicy.Outbound.External) == 0 {
		return nil
	}

	hosts := make([]string, len(app.Spec.AccessPolicy.Outbound.External))
	for i := range app.Spec.AccessPolicy.Outbound.External {
		hosts[i] = app.Spec.AccessPolicy.Outbound.External[i].Host
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
			Ports: []istio.Port{
				{
					Name:     "https",
					Protocol: "HTTPS",
					Number:   443,
				},
			},
		},
	}
}
