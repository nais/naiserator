package resourcecreator

import (
	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func networkPolicyRules(rules []nais.AccessPolicyGressRule) (networkPolicy []networkingv1.NetworkPolicyPeer) {
	for _, rule := range rules {
		networkPolicyPeer := networkingv1.NetworkPolicyPeer{
			PodSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": rule.Application,
				},
			},
		}

		if rule.Application == "*" {
			networkPolicyPeer = networkingv1.NetworkPolicyPeer{PodSelector: &metav1.LabelSelector{}}
		}

		if rule.Namespace != "" {
			networkPolicyPeer.NamespaceSelector = &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"name": rule.Namespace,
				},
			}
		}

		networkPolicy = append(networkPolicy, networkPolicyPeer)
	}

	return
}

func ingressPolicy(app *nais.Application) []networkingv1.NetworkPolicyIngressRule {
	rules := make([]networkingv1.NetworkPolicyIngressRule, 0)

	if len(app.Spec.AccessPolicy.Inbound.Rules) > 0 {
		rules = append(rules, networkingv1.NetworkPolicyIngressRule{
			From: networkPolicyRules(app.Spec.AccessPolicy.Inbound.Rules),
		})
	}

	if len(app.Spec.Ingresses) > 0 {
		rules = append(rules, networkingv1.NetworkPolicyIngressRule{
			From: []networkingv1.NetworkPolicyPeer{
				{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"istio": IstioIngressGatewayLabelValue,
						},
					},
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"name": IstioNamespace,
						},
					},
				},
			},
		})
	}

	return rules
}

func egressPolicy(app *nais.Application) []networkingv1.NetworkPolicyEgressRule {
	if len(app.Spec.AccessPolicy.Outbound.Rules) > 0 {
		return []networkingv1.NetworkPolicyEgressRule{
			{
				To: networkPolicyRules(app.Spec.AccessPolicy.Outbound.Rules),
			},
		}
	}

	return []networkingv1.NetworkPolicyEgressRule{}
}

func networkPolicySpec(app *nais.Application) networkingv1.NetworkPolicySpec {
	return networkingv1.NetworkPolicySpec{
		PodSelector: metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app": app.Name,
			},
		},
		PolicyTypes: []networkingv1.PolicyType{
			networkingv1.PolicyTypeIngress,
			networkingv1.PolicyTypeEgress,
		},
		Ingress: ingressPolicy(app),
		Egress:  egressPolicy(app),
	}

}

func NetworkPolicy(app *nais.Application) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NetworkPolicy",
			APIVersion: "networking.k8s.io/v1",
		},
		ObjectMeta: app.CreateObjectMeta(),
		Spec:       networkPolicySpec(app),
	}
}
