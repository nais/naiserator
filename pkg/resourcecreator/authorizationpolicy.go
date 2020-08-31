package resourcecreator

import (
	"fmt"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	istio "istio.io/api/security/v1beta1"
	"istio.io/api/type/v1beta1"
	istio_security_client "istio.io/client-go/pkg/apis/security/v1beta1"
	k8s_meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func AuthorizationPolicy(app *nais.Application, options ResourceOptions) *istio_security_client.AuthorizationPolicy {
	var rules []*istio.Rule

	// Authorization policy does not apply if app doesn't receive incoming traffic
	if len(app.Spec.AccessPolicy.Inbound.Rules) == 0 && len(app.Spec.Ingresses) == 0 {
		return nil
	}

	if len(app.Spec.Ingresses) > 0 {
		rules = append(rules, ingressGatewayRule())
	}

	if len(app.Spec.AccessPolicy.Inbound.Rules) > 0 {
		rules = append(rules, accessPolicyRules(app, options))
	}

	return &istio_security_client.AuthorizationPolicy{
		TypeMeta: k8s_meta.TypeMeta{
			Kind:       "AuthorizationPolicy",
			APIVersion: IstioAuthorizationPolicyVersion,
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec: istio.AuthorizationPolicy{
			Selector: &v1beta1.WorkloadSelector{
				MatchLabels: map[string]string{"app": app.Name},
			},
			Rules: rules,
		},
	}
}

func ingressGatewayRule() *istio.Rule {
	return &istio.Rule{
		From: []*istio.Rule_From{
			{
				Source: &istio.Source{
					Principals: []string{fmt.Sprintf("cluster.local/ns/%s/sa/%s", IstioNamespace, IstioIngressGatewayServiceAccount)},
				},
			},
		},
		To: []*istio.Rule_To{
			{
				Operation: &istio.Operation{
					Paths:   []string{"*"},
				},
			},
		},
	}
}

func accessPolicyRules(app *nais.Application, options ResourceOptions) *istio.Rule {
	return &istio.Rule{
		From: []*istio.Rule_From{
			{
				Source: &istio.Source{
					Principals: principals(app, options),
				},
			},
		},
		To: []*istio.Rule_To{
			{
				Operation: &istio.Operation{
					Paths:   []string{"*"},
				},
			},
		},
	}
}

func principals(app *nais.Application, options ResourceOptions) []string {
	var principals []string

	for _, rule := range app.Spec.AccessPolicy.Inbound.Rules {
		var namespace string
		// non-local access policy rules do not result in istio policies
		if !rule.MatchesCluster(options.ClusterName) {
			continue
		}
		if rule.Namespace == "" {
			namespace = app.Namespace
		} else {
			namespace = rule.Namespace
		}
		tmp := fmt.Sprintf("cluster.local/ns/%s/sa/%s", namespace, rule.Application)
		principals = append(principals, tmp)
	}
	return principals
}
