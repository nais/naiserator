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
				&istio.Rule{
					From: []*istio.Rule_From{
						&istio.Rule_From{
							Source: &istio.Source{
								Namespaces: []string{IstioNamespace},
							},
						},
					},
					To: []*istio.Rule_To{
						{
							Operation: &istio.Operation{
								Ports: []string{IstioPrometheusPort},
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

	if len(app.Spec.Ingresses) > 0 {
		tmp := istio.Rule_From{
			Source: &istio.Source{
				Principals: []string{fmt.Sprintf("cluster.local/ns/%s/sa/%s", IstioNamespace, IstioIngressGatewayServiceAccount)},
			},
		}
		rulesFrom = append(rulesFrom, &tmp)
	}

	for _, rule := range app.Spec.AccessPolicy.Inbound.Rules {
		var namespace string
		if rule.Namespace == "" {
			namespace = app.Namespace
		} else {
			namespace = rule.Namespace
		}
		var tmp istio.Rule_From
		tmp = istio.Rule_From{
			Source: &istio.Source{
				Principals: []string{fmt.Sprintf("cluster.local/ns/%s/sa/%s", namespace, rule.Application)},
			},
		}
		rulesFrom = append(rulesFrom, &tmp)
	}
	return rulesFrom
}

// apiVersion: security.istio.io/v1beta1
// kind: AuthorizationPolicy
// metadata:
//  annotations:
//    kubectl.kubernetes.io/last-applied-configuration: |
//      {"apiVersion":"security.istio.io/v1beta1","kind":"AuthorizationPolicy","metadata":{"annotations":{},"creationTimestamp":"2020-01-29T12:49:11Z","generation":1,"labels":{"app":"testapp-a","team":"aura"},"name":"testapp-a","namespace":"aura"},"spec":{"rules":[{"from":[{"source":{"namespace":["istio-system"]}}],"to":[{"operation":null,"ports":["15090"]}]},{"from":[{"source":{"principals":["cluster.local/ns/istio-system/sa/istio-ingressgateway-service-account"]}}],"to":[{"operation":{"methods":["*"],"paths":["*"]}}]}],"selector":{"matchLabels":{"app":"a"}}}}
//  creationTimestamp: "2020-01-29T13:28:56Z"
//  generation: 2
//  labels:
//    app: testapp-a
//    team: aura
//  name: testapp-a
//  namespace: aura
//  resourceVersion: "45430668"
//  selfLink: /apis/security.istio.io/v1beta1/namespaces/aura/authorizationpolicies/testapp-a
//  uid: 2a5f909b-d579-4014-8711-656fd51cea75
// spec:
//  rules:
//  - from:
//    - source:
//        namespace:
//        - istio-system
//    to:
//    - ports:
//      - "15090"
//  - from:
//    - source:
//        principals:
//        - cluster.local/ns/istio-system/sa/istio-ingressgateway-service-account
//    to:
//    - operation:
//        methods:
//        - '*'
//        paths:
//        - '*'
//  selector:
//    matchLabels:
//      app: a
