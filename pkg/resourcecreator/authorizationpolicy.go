package resourcecreator

import (
	"fmt"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	istio "istio.io/api/security/v1beta1"
	"istio.io/api/type/v1beta1"
	istio_security_client "istio.io/client-go/pkg/apis/security/v1beta1"
	k8s_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func AuthorizationPolicy(app *nais.Application) *istio_security_client.AuthorizationPolicy {
	objectMeta := app.CreateObjectMeta()
	return &istio_security_client.AuthorizationPolicy{
		TypeMeta: k8s_meta.TypeMeta{
			Kind:       "AuthorizationPolicy",
			APIVersion: IstioAuthorizationPolicyVersion,
		},
		ObjectMeta: objectMeta,
		Spec: istio.AuthorizationPolicy{
			Selector: &v1beta1.WorkloadSelector{
				MatchLabels: map[string]string{"app": app.Name},
			},
			Rules: []*istio.Rule{
				&istio.Rule{
					From: createFromRules(app),
					To: []*istio.Rule_To{
						{
							Operation: &istio.Operation{
								Methods: []string{"*"},
								Paths:   []string{"*"},
							},
						},
					},
				},
			},
		},
	}
}

func createFromRules(app *nais.Application) []*istio.Rule_From {
	var rulesFrom []*istio.Rule_From

	for _, rule := range app.Spec.AccessPolicy.Inbound.Rules {
		var tmp istio.Rule_From
		tmp = istio.Rule_From{
			Source: &istio.Source{
				Principals: []string{fmt.Sprintf("cluster.local/ns/%s/sa/%s", rule.Namespace, rule.Application)},
				Namespaces: []string{rule.Namespace},
			},
		}
		rulesFrom = append(rulesFrom, &tmp)
	}
	return rulesFrom
}

// func createToRules(app *nais.Application) []*istio.Rule_To {
//	var rulesTo []*istio.Rule_To
//
//	for _, rule := range app.Spec.AccessPolicy.Outbound.Rules {
//		var tmp istio.Rule_To
//		tmp = istio.Rule_To{
//			Operation: &istio.Operation{
//				Methods: []string{"*"},
//				Paths:   []string{"*"},
//			},
//		}
//		rulesTo = append(rulesTo, &tmp)
//	}
//	return rulesTo
// }
